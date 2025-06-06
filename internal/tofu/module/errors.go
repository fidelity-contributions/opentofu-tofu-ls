// Copyright (c) The OpenTofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2024 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package module

import (
	"fmt"
)

type ModuleNotFoundErr struct {
	Dir string
}

func (e *ModuleNotFoundErr) Error() string {
	if e.Dir != "" {
		return fmt.Sprintf("module not found for %s", e.Dir)
	}
	return "module not found"
}

func IsModuleNotFound(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*ModuleNotFoundErr)
	return ok
}

type NoTofuExecPathErr struct{}

func (NoTofuExecPathErr) Error() string {
	return "No exec path provided for tofu"
}

func IsTofuNotFound(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(NoTofuExecPathErr)
	return ok
}
