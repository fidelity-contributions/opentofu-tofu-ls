// Copyright (c) The OpenTofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2024 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"path/filepath"

	"github.com/hashicorp/hcl-lang/decoder"
	lsp "github.com/opentofu/tofu-ls/internal/protocol"
	"github.com/opentofu/tofu-ls/internal/uri"
)

func RefOriginsToLocations(origins decoder.ReferenceOrigins) []lsp.Location {
	locations := make([]lsp.Location, len(origins))

	for i, origin := range origins {
		originUri := uri.FromPath(filepath.Join(origin.Path.Path, origin.Range.Filename))
		locations[i] = lsp.Location{
			URI:   lsp.DocumentURI(originUri),
			Range: HCLRangeToLSP(origin.Range),
		}
	}

	return locations
}
