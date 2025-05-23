// Copyright (c) The OpenTofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2024 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package walker

import (
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/opentofu/tofu-ls/internal/job"
)

type WalkerCollector struct {
	errors   *multierror.Error
	errorsMu *sync.RWMutex

	jobIds   job.IDs
	jobIdsMu *sync.RWMutex
}

func NewWalkerCollector() *WalkerCollector {
	return &WalkerCollector{
		errorsMu: &sync.RWMutex{},
		jobIds:   make(job.IDs, 0),
		jobIdsMu: &sync.RWMutex{},
	}
}

func (wc *WalkerCollector) CollectError(err error) {
	wc.errorsMu.Lock()
	defer wc.errorsMu.Unlock()
	wc.errors = multierror.Append(wc.errors, err)
}

func (wc *WalkerCollector) ErrorOrNil() error {
	wc.errorsMu.RLock()
	defer wc.errorsMu.RUnlock()
	return wc.errors.ErrorOrNil()
}

func (wc *WalkerCollector) CollectJobId(jobId job.ID) {
	wc.jobIdsMu.Lock()
	defer wc.jobIdsMu.Unlock()
	wc.jobIds = append(wc.jobIds, jobId)
}

func (wc *WalkerCollector) JobIds() job.IDs {
	wc.jobIdsMu.RLock()
	defer wc.jobIdsMu.RUnlock()
	return wc.jobIds
}
