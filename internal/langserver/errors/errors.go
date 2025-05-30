// Copyright (c) The OpenTofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2024 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package errors

import (
	e "errors"

	"github.com/opentofu/tofu-ls/internal/tofu/module"
)

func EnrichTfExecError(err error) error {
	if module.IsTofuNotFound(err) {
		return e.New("Tofu (CLI) is required. " +
			"Please install Tofu or make it available in $PATH")
	}
	return err
}
