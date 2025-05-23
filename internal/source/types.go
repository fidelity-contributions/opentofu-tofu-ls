// Copyright (c) The OpenTofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2024 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package source

type Lines []Line

func (l Lines) Copy() Lines {
	newLines := make(Lines, len(l))

	for i, line := range l {
		newLines[i] = line.Copy()
	}

	return newLines
}
