// Copyright (c) The OpenTofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utm

import (
	"context"

	ilsp "github.com/opentofu/opentofu-ls/internal/lsp"
)

const UtmSource = "opentofu-ls"

func UtmMedium(ctx context.Context) string {
	clientName, ok := ilsp.ClientName(ctx)
	if ok {
		return clientName
	}

	return ""
}
