// Copyright (c) The OpenTofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2024 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package job

import (
	"context"
	"fmt"

	"github.com/opentofu/tofu-ls/internal/document"
)

type ignoreState struct{}

func IgnoreState(ctx context.Context) bool {
	v, ok := ctx.Value(ignoreState{}).(bool)
	if !ok {
		return false
	}
	return v
}

func WithIgnoreState(ctx context.Context, ignore bool) context.Context {
	return context.WithValue(ctx, ignoreState{}, ignore)
}

type StateNotChangedErr struct {
	Dir document.DirHandle
}

func (e StateNotChangedErr) Error() string {
	return fmt.Sprintf("%s: state not changed", e.Dir.URI)
}
