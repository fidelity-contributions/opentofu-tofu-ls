// Copyright (c) The OpenTofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2024 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jobs

import (
	"context"
	"fmt"

	"github.com/opentofu/tofu-ls/internal/document"
	"github.com/opentofu/tofu-ls/internal/features/rootmodules/state"
	"github.com/opentofu/tofu-ls/internal/job"
	"github.com/opentofu/tofu-ls/internal/tofu/datadir"
	op "github.com/opentofu/tofu-ls/internal/tofu/module/operation"
)

// ParseModuleManifest parses the "module manifest" which
// contains records of installed modules, e.g. where they're
// installed on the filesystem.
// This is useful for processing any modules which are not local
// nor hosted in the Registry (which would be handled by
// [GetModuleDataFromRegistry]).
func ParseModuleManifest(ctx context.Context, fs ReadOnlyFS, rootStore *state.RootStore, modPath string) error {
	mod, err := rootStore.RootRecordByPath(modPath)
	if err != nil {
		return err
	}

	// Avoid parsing if it is already in progress or already known
	if mod.ModManifestState != op.OpStateUnknown && !job.IgnoreState(ctx) {
		return job.StateNotChangedErr{Dir: document.DirHandleFromPath(modPath)}
	}

	err = rootStore.SetModManifestState(modPath, op.OpStateLoading)
	if err != nil {
		return err
	}

	manifestPath, ok := datadir.ModuleManifestFilePath(fs, modPath)
	if !ok {
		err := fmt.Errorf("%s: manifest file does not exist", modPath)
		sErr := rootStore.UpdateModManifest(modPath, nil, err)
		if sErr != nil {
			return sErr
		}
		return err
	}

	mm, err := datadir.ParseModuleManifestFromFile(manifestPath)
	if err != nil {
		err := fmt.Errorf("failed to parse manifest: %w", err)
		sErr := rootStore.UpdateModManifest(modPath, nil, err)
		if sErr != nil {
			return sErr
		}
		return err
	}

	sErr := rootStore.UpdateModManifest(modPath, mm, err)

	if sErr != nil {
		return sErr
	}
	return err
}
