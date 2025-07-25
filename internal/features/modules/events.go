// Copyright (c) The OpenTofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2024 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package modules

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-multierror"
	tfmod "github.com/opentofu/opentofu-schema/module"
	tfaddr "github.com/opentofu/registry-address"
	lsctx "github.com/opentofu/tofu-ls/internal/context"
	"github.com/opentofu/tofu-ls/internal/document"
	"github.com/opentofu/tofu-ls/internal/features/modules/ast"
	"github.com/opentofu/tofu-ls/internal/features/modules/jobs"
	"github.com/opentofu/tofu-ls/internal/job"
	"github.com/opentofu/tofu-ls/internal/lsp"
	"github.com/opentofu/tofu-ls/internal/protocol"
	"github.com/opentofu/tofu-ls/internal/schemas"
	globalState "github.com/opentofu/tofu-ls/internal/state"
	globalAst "github.com/opentofu/tofu-ls/internal/tofu/ast"
	op "github.com/opentofu/tofu-ls/internal/tofu/module/operation"
)

func (f *ModulesFeature) discover(path string, files []string) error {
	for _, file := range files {
		if ast.IsModuleFilename(file) && !globalAst.IsIgnoredFile(file) {
			f.logger.Printf("discovered module file in %s", path)

			err := f.Store.AddIfNotExists(path)
			if err != nil {
				return err
			}

			break
		}
	}

	return nil
}

func (f *ModulesFeature) didOpen(ctx context.Context, dir document.DirHandle, languageID string) (job.IDs, error) {
	ids := make(job.IDs, 0)
	path := dir.Path()
	f.logger.Printf("did open %q %q", path, languageID)

	// We need to decide if the path is relevant to us. It can be relevant because
	// a) the walker discovered module files and created a state entry for them
	// b) the opened file is a module file
	//
	// Add to state if language ID matches
	if lsp.IsValidConfigLanguage(languageID) {
		err := f.Store.AddIfNotExists(path)
		if err != nil {
			return ids, err
		}
	}

	// Schedule jobs if state entry exists
	hasModuleRecord := f.Store.Exists(path)
	if !hasModuleRecord {
		return ids, nil
	}

	return f.decodeModule(ctx, dir, false, true)
}

func (f *ModulesFeature) didChange(ctx context.Context, dir document.DirHandle) (job.IDs, error) {
	hasModuleRecord := f.Store.Exists(dir.Path())
	if !hasModuleRecord {
		return job.IDs{}, nil
	}

	return f.decodeModule(ctx, dir, true, true)
}

func (f *ModulesFeature) didChangeWatched(ctx context.Context, rawPath string, changeType protocol.FileChangeType, isDir bool) (job.IDs, error) {
	ids := make(job.IDs, 0)

	if changeType == protocol.Deleted {
		// We don't know whether file or dir is being deleted
		// 1st we just blindly try to look it up as a directory
		hasModuleRecord := f.Store.Exists(rawPath)
		if hasModuleRecord {
			f.removeIndexedModule(rawPath)
			return ids, nil
		}

		// 2nd we try again assuming it is a file
		parentDir := filepath.Dir(rawPath)
		hasModuleRecord = f.Store.Exists(parentDir)
		if !hasModuleRecord {
			// Nothing relevant found in the feature state
			return ids, nil
		}

		// and check the parent directory still exists
		fi, err := os.Stat(parentDir)
		if err != nil {
			if os.IsNotExist(err) {
				// if not, we remove the indexed module
				f.removeIndexedModule(rawPath)
				return ids, nil
			}
			f.logger.Printf("error checking existence (%q deleted): %s", parentDir, err)
			return ids, nil
		}
		if !fi.IsDir() {
			// Should never happen
			f.logger.Printf("error: %q (deleted) is not a directory", parentDir)
			return ids, nil
		}

		// If the parent directory exists, we just need to
		// check if the there are open documents for the path and the
		// path is a module path. If so, we need to reparse the module.
		dir := document.DirHandleFromPath(parentDir)
		hasOpenDocs, err := f.stateStore.DocumentStore.HasOpenDocuments(dir)
		if err != nil {
			f.logger.Printf("error when checking for open documents in path (%q deleted): %s", rawPath, err)
		}
		if !hasOpenDocs {
			return ids, nil
		}

		f.decodeModule(ctx, dir, true, true)
	}

	if changeType == protocol.Changed || changeType == protocol.Created {
		var dir document.DirHandle
		if isDir {
			dir = document.DirHandleFromPath(rawPath)
		} else {
			docHandle := document.HandleFromPath(rawPath)
			dir = docHandle.Dir
		}

		// Check if the there are open documents for the path and the
		// path is a module path. If so, we need to reparse the module.
		hasOpenDocs, err := f.stateStore.DocumentStore.HasOpenDocuments(dir)
		if err != nil {
			f.logger.Printf("error when checking for open documents in path (%q changed): %s", rawPath, err)
		}
		if !hasOpenDocs {
			return ids, nil
		}

		hasModuleRecord := f.Store.Exists(dir.Path())
		if !hasModuleRecord {
			return ids, nil
		}

		f.decodeModule(ctx, dir, true, true)
	}

	return ids, nil
}

func (f *ModulesFeature) removeIndexedModule(rawPath string) {
	modHandle := document.DirHandleFromPath(rawPath)

	err := f.stateStore.JobStore.DequeueJobsForDir(modHandle)
	if err != nil {
		f.logger.Printf("failed to dequeue jobs for module: %s", err)
		return
	}

	err = f.Store.Remove(rawPath)
	if err != nil {
		f.logger.Printf("failed to remove module from state: %s", err)
		return
	}
}

func (f *ModulesFeature) decodeDeclaredModuleCalls(ctx context.Context, dir document.DirHandle, ignoreState bool) (job.IDs, error) {
	jobIds := make(job.IDs, 0)

	declared, err := f.Store.DeclaredModuleCalls(dir.Path())
	if err != nil {
		return jobIds, err
	}

	var errs *multierror.Error

	for _, mc := range declared {
		var mcPath string
		switch source := mc.SourceAddr.(type) {
		// For local module sources, we can construct the path directly from the configuration
		case tfmod.LocalSourceAddr:
			mcPath = filepath.Join(dir.Path(), filepath.FromSlash(source.String()))
		// For registry modules, we need to find the local installation path (if installed)
		case tfaddr.Module:
			installedDir, ok := f.rootFeature.InstalledModulePath(dir.Path(), source.String())
			if !ok {
				continue
			}
			mcPath = filepath.Join(dir.Path(), filepath.FromSlash(installedDir))
		// For other remote modules, we need to find the local installation path (if installed)
		case tfmod.RemoteSourceAddr:
			installedDir, ok := f.rootFeature.InstalledModulePath(dir.Path(), source.String())
			if !ok {
				continue
			}
			mcPath = filepath.Join(dir.Path(), filepath.FromSlash(installedDir))
		default:
			// Unknown source address, we can't resolve the path
			continue
		}

		fi, err := os.Stat(mcPath)
		if err != nil || !fi.IsDir() {
			multierror.Append(errs, err)
			continue
		}

		mcIgnoreState := ignoreState
		err = f.Store.Add(mcPath)
		if err != nil {
			alreadyExistsErr := &globalState.AlreadyExistsError{}
			if errors.As(err, &alreadyExistsErr) {
				mcIgnoreState = false
			} else {
				multierror.Append(errs, err)
				continue
			}
		}

		mcHandle := document.DirHandleFromPath(mcPath)
		mcJobIds, mcErr := f.decodeModule(ctx, mcHandle, mcIgnoreState, false)
		jobIds = append(jobIds, mcJobIds...)
		multierror.Append(errs, mcErr)
	}

	return jobIds, errs.ErrorOrNil()
}

func (f *ModulesFeature) decodeModule(ctx context.Context, dir document.DirHandle, ignoreState bool, isFirstLevel bool) (job.IDs, error) {
	ids := make(job.IDs, 0)
	path := dir.Path()

	parseId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
		Dir: dir,
		Func: func(ctx context.Context) error {
			return jobs.ParseModuleConfiguration(ctx, f.fs, f.Store, path)
		},
		Type:        op.OpTypeParseModuleConfiguration.String(),
		IgnoreState: ignoreState,
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, parseId)

	// Changes to a setting currently requires a LS restart, so the LS
	// setting context cannot change during the execution of a job. That's
	// why we can extract it here and use it in Defer.
	// See https://github.com/hashicorp/terraform-ls/issues/1008
	// We can safely ignore the error here. If we can't get the options from
	// the context, validationOptions.EnableEnhancedValidation will be false
	// by default. So we don't run the validation jobs.
	validationOptions, _ := lsctx.ValidationOptions(ctx)

	metaId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
		Dir: dir,
		Func: func(ctx context.Context) error {
			return jobs.LoadModuleMetadata(ctx, f.Store, path)
		},
		Type:        op.OpTypeLoadModuleMetadata.String(),
		DependsOn:   job.IDs{parseId},
		IgnoreState: ignoreState,
		Defer: func(ctx context.Context, jobErr error) (job.IDs, error) {
			deferIds := make(job.IDs, 0)
			if jobErr != nil {
				f.logger.Printf("loading module metadata returned error: %s", jobErr)
			}

			modCalls := job.IDs{}
			if isFirstLevel {
				var mcErr error
				modCalls, mcErr = f.decodeDeclaredModuleCalls(ctx, dir, ignoreState)
				if mcErr != nil {
					f.logger.Printf("decoding declared module calls for %q failed: %s", dir.URI, mcErr)
					// We log the error but still continue scheduling other jobs
					// which are still valuable for the rest of the configuration
					// even if they may not have the data for module calls.
				}
				deferIds = append(deferIds, modCalls...)
			}

			eSchemaId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
				Dir: dir,
				Func: func(ctx context.Context) error {
					return jobs.PreloadEmbeddedSchema(ctx, f.logger, schemas.FS,
						f.Store, f.stateStore.ProviderSchemas, path)
				},
				Type:        op.OpTypePreloadEmbeddedSchema.String(),
				IgnoreState: ignoreState,
			})
			if err != nil {
				return deferIds, err
			}
			deferIds = append(deferIds, eSchemaId)

			refTargetsId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
				Dir: dir,
				Func: func(ctx context.Context) error {
					return jobs.DecodeReferenceTargets(ctx, f.Store, f.rootFeature, path)
				},
				Type:        op.OpTypeDecodeReferenceTargets.String(),
				DependsOn:   append(modCalls, eSchemaId),
				IgnoreState: ignoreState,
			})
			if err != nil {
				return deferIds, err
			}
			deferIds = append(deferIds, refTargetsId)

			refOriginsId, err := f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
				Dir: dir,
				Func: func(ctx context.Context) error {
					return jobs.DecodeReferenceOrigins(ctx, f.Store, f.rootFeature, path)
				},
				Type:        op.OpTypeDecodeReferenceOrigins.String(),
				DependsOn:   append(modCalls, eSchemaId),
				IgnoreState: ignoreState,
			})
			if err != nil {
				return deferIds, err
			}
			deferIds = append(deferIds, refOriginsId)

			// We don't want to validate nested modules
			if isFirstLevel && validationOptions.EnableEnhancedValidation {
				_, err = f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
					Dir: dir,
					Func: func(ctx context.Context) error {
						return jobs.SchemaModuleValidation(ctx, f.Store, f.rootFeature, dir.Path())
					},
					Type:        op.OpTypeSchemaModuleValidation.String(),
					DependsOn:   append(modCalls, eSchemaId),
					IgnoreState: ignoreState,
				})
				if err != nil {
					return deferIds, err
				}

				_, err = f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
					Dir: dir,
					Func: func(ctx context.Context) error {
						return jobs.ReferenceValidation(ctx, f.Store, f.rootFeature, dir.Path())
					},
					Type:        op.OpTypeReferenceValidation.String(),
					DependsOn:   job.IDs{refOriginsId, refTargetsId},
					IgnoreState: ignoreState,
				})
				if err != nil {
					return deferIds, err
				}
			}

			return deferIds, nil
		},
	})
	if err != nil {
		return ids, err
	}
	ids = append(ids, metaId)

	// We don't want to fetch module data from the registry for nested modules,
	// so we return early.
	if !isFirstLevel {
		return ids, nil
	}

	// This job may make an HTTP request, and we schedule it in
	// the low-priority queue, so we don't want to wait for it.
	_, err = f.stateStore.JobStore.EnqueueJob(ctx, job.Job{
		Dir: dir,
		Func: func(ctx context.Context) error {
			return jobs.GetModuleDataFromRegistry(ctx, f.registryClient,
				f.Store, f.stateStore.RegistryModules, path)
		},
		Priority:  job.LowPriority,
		DependsOn: job.IDs{metaId},
		Type:      op.OpTypeGetModuleDataFromRegistry.String(),
	})
	if err != nil {
		return ids, err
	}

	return ids, nil
}
