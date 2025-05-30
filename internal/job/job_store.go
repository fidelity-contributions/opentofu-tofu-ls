// Copyright (c) The OpenTofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2024 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package job

import (
	"context"
)

type JobStore interface {
	EnqueueJob(ctx context.Context, newJob Job) (ID, error)
	WaitForJobs(ctx context.Context, ids ...ID) error
}
