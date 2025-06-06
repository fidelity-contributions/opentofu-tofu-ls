// Copyright (c) The OpenTofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2024 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"

	ilsp "github.com/opentofu/tofu-ls/internal/lsp"
	lsp "github.com/opentofu/tofu-ls/internal/protocol"
)

func (svc *service) TextDocumentLink(ctx context.Context, params lsp.DocumentLinkParams) ([]lsp.DocumentLink, error) {
	cc, err := ilsp.ClientCapabilities(ctx)
	if err != nil {
		return nil, err
	}

	dh := ilsp.HandleFromDocumentURI(params.TextDocument.URI)
	doc, err := svc.stateStore.DocumentStore.GetDocument(dh)
	if err != nil {
		return nil, err
	}

	if doc.LanguageID != ilsp.OpenTofu.String() {
		return nil, nil
	}

	jobIds, err := svc.stateStore.JobStore.ListIncompleteJobsForDir(dh.Dir)
	if err != nil {
		return nil, err
	}
	svc.stateStore.JobStore.WaitForJobs(ctx, jobIds...)

	d, err := svc.decoderForDocument(ctx, doc)
	if err != nil {
		return nil, err
	}

	links, err := d.LinksInFile(doc.Filename)
	if err != nil {
		return nil, err
	}

	return ilsp.Links(links, cc.TextDocument.DocumentLink), nil
}
