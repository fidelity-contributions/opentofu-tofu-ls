// Copyright (c) The OpenTofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2024 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"path"

	"github.com/hashicorp/hcl-lang/decoder"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl/v2"
	lsctx "github.com/opentofu/tofu-ls/internal/context"
	idecoder "github.com/opentofu/tofu-ls/internal/decoder"
	"github.com/opentofu/tofu-ls/internal/document"
	"github.com/opentofu/tofu-ls/internal/features/modules/ast"
	fdecoder "github.com/opentofu/tofu-ls/internal/features/modules/decoder"
	"github.com/opentofu/tofu-ls/internal/features/modules/decoder/validations"
	"github.com/opentofu/tofu-ls/internal/features/modules/state"
	"github.com/opentofu/tofu-ls/internal/job"
	"github.com/opentofu/tofu-ls/internal/langserver/diagnostics"
	ilsp "github.com/opentofu/tofu-ls/internal/lsp"
	globalAst "github.com/opentofu/tofu-ls/internal/tofu/ast"
	"github.com/opentofu/tofu-ls/internal/tofu/module"
	op "github.com/opentofu/tofu-ls/internal/tofu/module/operation"
)

// SchemaModuleValidation does schema-based validation
// of module files (*.tf) and produces diagnostics
// associated with any "invalid" parts of code.
//
// It relies on previously parsed AST (via [ParseModuleConfiguration]),
// core schema of appropriate version (as obtained via [GetTofuVersion])
// and provider schemas ([PreloadEmbeddedSchema] or [ObtainSchema]).
func SchemaModuleValidation(ctx context.Context, modStore *state.ModuleStore, rootFeature fdecoder.RootReader, modPath string) error {
	mod, err := modStore.ModuleRecordByPath(modPath)
	if err != nil {
		return err
	}

	// Avoid validation if it is already in progress or already finished
	if mod.ModuleDiagnosticsState[globalAst.SchemaValidationSource] != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = modStore.SetModuleDiagnosticsState(modPath, globalAst.SchemaValidationSource, op.OpStateLoading)
	if err != nil {
		return err
	}

	d := decoder.NewDecoder(&fdecoder.PathReader{
		StateReader: modStore,
		RootReader:  rootFeature,
	})
	d.SetContext(idecoder.DecoderContext(ctx))

	moduleDecoder, err := d.Path(lang.Path{
		Path:       modPath,
		LanguageID: ilsp.OpenTofu.String(),
	})
	if err != nil {
		return err
	}

	var rErr error
	rpcContext := lsctx.DocumentContext(ctx)
	if rpcContext.Method == "textDocument/didChange" && ilsp.IsValidConfigLanguage(rpcContext.LanguageID) {
		filename := path.Base(rpcContext.URI)
		// We only revalidate a single file that changed
		var fileDiags hcl.Diagnostics
		fileDiags, rErr = moduleDecoder.ValidateFile(ctx, filename)

		modDiags, ok := mod.ModuleDiagnostics[globalAst.SchemaValidationSource]
		if !ok {
			modDiags = make(ast.ModDiags)
		}
		modDiags[ast.ModFilename(filename)] = fileDiags

		sErr := modStore.UpdateModuleDiagnostics(modPath, globalAst.SchemaValidationSource, modDiags)
		if sErr != nil {
			return sErr
		}
	} else {
		// We validate the whole module, e.g. on open
		var diags lang.DiagnosticsMap
		diags, rErr = moduleDecoder.Validate(ctx)

		sErr := modStore.UpdateModuleDiagnostics(modPath, globalAst.SchemaValidationSource, ast.ModDiagsFromMap(diags))
		if sErr != nil {
			return sErr
		}
	}

	return rErr
}

// ReferenceValidation does validation based on (mis)matched
// reference origins and targets, to flag up "orphaned" references.
//
// It relies on [DecodeReferenceTargets] and [DecodeReferenceOrigins]
// to supply both origins and targets to compare.
func ReferenceValidation(ctx context.Context, modStore *state.ModuleStore, rootFeature fdecoder.RootReader, modPath string) error {
	mod, err := modStore.ModuleRecordByPath(modPath)
	if err != nil {
		return err
	}

	// Avoid validation if it is already in progress or already finished
	if mod.ModuleDiagnosticsState[globalAst.ReferenceValidationSource] != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = modStore.SetModuleDiagnosticsState(modPath, globalAst.ReferenceValidationSource, op.OpStateLoading)
	if err != nil {
		return err
	}

	pathReader := &fdecoder.PathReader{
		StateReader: modStore,
		RootReader:  rootFeature,
	}
	pathCtx, err := pathReader.PathContext(lang.Path{
		Path:       modPath,
		LanguageID: ilsp.OpenTofu.String(),
	})
	if err != nil {
		return err
	}

	diags := validations.UnreferencedOrigins(ctx, pathCtx)
	return modStore.UpdateModuleDiagnostics(modPath, globalAst.ReferenceValidationSource, ast.ModDiagsFromMap(diags))
}

// TofuValidate uses Tofu CLI to run validate subcommand
// and turn the provided (JSON) output into diagnostics associated
// with "invalid" parts of code.
func TofuValidate(ctx context.Context, modStore *state.ModuleStore, modPath string) error {
	mod, err := modStore.ModuleRecordByPath(modPath)
	if err != nil {
		return err
	}

	// Avoid validation if it is already in progress or already finished
	if mod.ModuleDiagnosticsState[globalAst.TofuValidateSource] != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = modStore.SetModuleDiagnosticsState(modPath, globalAst.TofuValidateSource, op.OpStateLoading)
	if err != nil {
		return err
	}

	tfExec, err := module.TofuExecutorForModule(ctx, mod.Path())
	if err != nil {
		return err
	}

	jsonDiags, err := tfExec.Validate(ctx)
	if err != nil {
		return err
	}
	validateDiags := diagnostics.HCLDiagsFromJSON(jsonDiags)

	return modStore.UpdateModuleDiagnostics(modPath, globalAst.TofuValidateSource, ast.ModDiagsFromMap(validateDiags))
}
