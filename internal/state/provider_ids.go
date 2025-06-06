// Copyright (c) The OpenTofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2024 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"github.com/hashicorp/go-uuid"
	tfaddr "github.com/opentofu/registry-address"
)

type ProviderIds struct {
	Address tfaddr.Provider
	ID      string
}

func (s *ProviderSchemaStore) GetProviderID(addr tfaddr.Provider) (string, error) {
	txn := s.db.Txn(true)
	defer txn.Abort()

	obj, err := txn.First(providerIdsTableName, "id", addr)
	if err != nil {
		return "", err
	}

	if obj != nil {
		return obj.(ProviderIds).ID, nil
	}

	newId, err := uuid.GenerateUUID()
	if err != nil {
		return "", err
	}

	err = txn.Insert(providerIdsTableName, ProviderIds{
		ID:      newId,
		Address: addr,
	})
	if err != nil {
		return "", err
	}

	txn.Commit()
	return newId, nil
}
