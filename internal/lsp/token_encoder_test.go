// Copyright (c) The OpenTofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2024 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl/v2"
	"github.com/opentofu/tofu-ls/internal/protocol"
	"github.com/opentofu/tofu-ls/internal/source"
)

func TestTokenEncoder_singleLineTokens(t *testing.T) {
	bytes := []byte(`myblock "mytype" {
  str_attr = "something"
  num_attr = 42
  bool_attr = true
}`)
	te := &TokenEncoder{
		Lines: source.MakeSourceLines("test.tf", bytes),
		Tokens: []lang.SemanticToken{
			{
				Type: lang.TokenBlockType,
				Range: hcl.Range{
					Filename: "test.tf",
					Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
					End:      hcl.Pos{Line: 1, Column: 8, Byte: 7},
				},
			},
			{
				Type: lang.TokenBlockLabel,
				Range: hcl.Range{
					Filename: "test.tf",
					Start:    hcl.Pos{Line: 1, Column: 9, Byte: 8},
					End:      hcl.Pos{Line: 1, Column: 8, Byte: 16},
				},
			},
			{
				Type: lang.TokenAttrName,
				Range: hcl.Range{
					Filename: "test.tf",
					Start:    hcl.Pos{Line: 2, Column: 3, Byte: 21},
					End:      hcl.Pos{Line: 2, Column: 11, Byte: 29},
				},
			},
			{
				Type: lang.TokenAttrName,
				Range: hcl.Range{
					Filename: "test.tf",
					Start:    hcl.Pos{Line: 3, Column: 3, Byte: 46},
					End:      hcl.Pos{Line: 3, Column: 11, Byte: 54},
				},
			},
			{
				Type: lang.TokenAttrName,
				Range: hcl.Range{
					Filename: "test.tf",
					Start:    hcl.Pos{Line: 4, Column: 3, Byte: 62},
					End:      hcl.Pos{Line: 4, Column: 12, Byte: 71},
				},
			},
		},
		ClientCaps: protocol.SemanticTokensClientCapabilities{
			TokenTypes:     serverTokenTypes.AsStrings(),
			TokenModifiers: serverTokenModifiers.AsStrings(),
		},
	}
	data := te.Encode()
	expectedData := []uint32{
		0, 0, 7, 10, 0,
		0, 8, 8, 11, 0,
		1, 2, 8, 9, 0,
		1, 2, 8, 9, 0,
		1, 2, 9, 9, 0,
	}

	if diff := cmp.Diff(expectedData, data); diff != "" {
		t.Fatalf("unexpected encoded data.\nexpected: %#v\ngiven:    %#v",
			expectedData, data)
	}
}

func TestTokenEncoder_unknownTokenType(t *testing.T) {
	bytes := []byte(`variable "test" {
  type = string
  default = "foo"
}
`)
	te := &TokenEncoder{
		Lines: source.MakeSourceLines("test.tf", bytes),
		Tokens: []lang.SemanticToken{
			{
				Type:      lang.SemanticTokenType("unknown"),
				Modifiers: []lang.SemanticTokenModifier{},
				Range: hcl.Range{
					Filename: "main.tf",
					Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
					End:      hcl.Pos{Line: 1, Column: 9, Byte: 8},
				},
			},
			{
				Type:      lang.SemanticTokenType("another-unknown"),
				Modifiers: []lang.SemanticTokenModifier{},
				Range: hcl.Range{
					Filename: "main.tf",
					Start:    hcl.Pos{Line: 2, Column: 3, Byte: 20},
					End:      hcl.Pos{Line: 2, Column: 7, Byte: 24},
				},
			},
			{
				Type:      lang.TokenAttrName,
				Modifiers: []lang.SemanticTokenModifier{},
				Range: hcl.Range{
					Filename: "main.tf",
					Start:    hcl.Pos{Line: 3, Column: 3, Byte: 36},
					End:      hcl.Pos{Line: 3, Column: 10, Byte: 43},
				},
			},
		},
		ClientCaps: protocol.SemanticTokensClientCapabilities{
			TokenTypes:     serverTokenTypes.AsStrings(),
			TokenModifiers: serverTokenModifiers.AsStrings(),
		},
	}
	data := te.Encode()
	expectedData := []uint32{
		2, 2, 7, 9, 0,
	}

	if diff := cmp.Diff(expectedData, data); diff != "" {
		t.Fatalf("unexpected encoded data.\nexpected: %#v\ngiven:    %#v",
			expectedData, data)
	}
}

func TestTokenEncoder_multiLineTokens(t *testing.T) {
	bytes := []byte(`myblock "mytype" {
  str_attr = "something"
  num_attr = 42
  bool_attr = true
}`)
	te := &TokenEncoder{
		Lines: source.MakeSourceLines("test.tf", bytes),
		Tokens: []lang.SemanticToken{
			{
				Type: lang.TokenAttrName,
				Range: hcl.Range{
					Filename: "test.tf",
					// Attribute name would actually never span
					// multiple lines, but we don't have any token
					// type that would *yet*
					Start: hcl.Pos{Line: 2, Column: 3, Byte: 21},
					End:   hcl.Pos{Line: 4, Column: 12, Byte: 71},
				},
			},
		},
		ClientCaps: protocol.SemanticTokensClientCapabilities{
			TokenTypes:     serverTokenTypes.AsStrings(),
			TokenModifiers: serverTokenModifiers.AsStrings(),
		},
	}
	data := te.Encode()
	expectedData := []uint32{
		1, 2, 24, 9, 0,
		1, 0, 15, 9, 0,
		1, 0, 11, 9, 0,
	}

	if diff := cmp.Diff(expectedData, data); diff != "" {
		t.Fatalf("unexpected encoded data.\nexpected: %#v\ngiven:    %#v",
			expectedData, data)
	}
}

func TestTokenEncoder_deltaStartCharBug(t *testing.T) {
	bytes := []byte(`resource "aws_iam_role_policy" "firehose_s3_access" {
}
`)
	te := &TokenEncoder{
		Lines: source.MakeSourceLines("test.tf", bytes),
		Tokens: []lang.SemanticToken{
			{
				Type:      lang.TokenBlockType,
				Modifiers: []lang.SemanticTokenModifier{},
				Range: hcl.Range{
					Filename: "main.tf",
					Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
					End:      hcl.Pos{Line: 1, Column: 9, Byte: 8},
				},
			},
			{
				Type:      lang.TokenBlockLabel,
				Modifiers: []lang.SemanticTokenModifier{lang.TokenModifierDependent},
				Range: hcl.Range{
					Filename: "main.tf",
					Start:    hcl.Pos{Line: 1, Column: 10, Byte: 9},
					End:      hcl.Pos{Line: 1, Column: 31, Byte: 30},
				},
			},
			{
				Type:      lang.TokenBlockLabel,
				Modifiers: []lang.SemanticTokenModifier{},
				Range: hcl.Range{
					Filename: "main.tf",
					Start:    hcl.Pos{Line: 1, Column: 32, Byte: 31},
					End:      hcl.Pos{Line: 1, Column: 52, Byte: 51},
				},
			},
		},
		ClientCaps: protocol.SemanticTokensClientCapabilities{
			TokenTypes:     serverTokenTypes.AsStrings(),
			TokenModifiers: serverTokenModifiers.AsStrings(),
		},
	}
	data := te.Encode()
	expectedData := []uint32{
		0, 0, 8, 10, 0,
		0, 9, 21, 11, 2,
		0, 22, 20, 11, 0,
	}

	if diff := cmp.Diff(expectedData, data); diff != "" {
		t.Fatalf("unexpected encoded data.\nexpected: %#v\ngiven:    %#v",
			expectedData, data)
	}
}

func TestTokenEncoder_tokenModifiers(t *testing.T) {
	bytes := []byte(`myblock "mytype" {
  str_attr = "something"
  num_attr = 42
  bool_attr = true
}`)
	te := &TokenEncoder{
		Lines: source.MakeSourceLines("test.tf", bytes),
		Tokens: []lang.SemanticToken{
			{
				Type: lang.TokenBlockType,
				Range: hcl.Range{
					Filename: "test.tf",
					Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
					End:      hcl.Pos{Line: 1, Column: 8, Byte: 7},
				},
			},
			{
				Type:      lang.TokenBlockLabel,
				Modifiers: []lang.SemanticTokenModifier{},
				Range: hcl.Range{
					Filename: "test.tf",
					Start:    hcl.Pos{Line: 1, Column: 9, Byte: 8},
					End:      hcl.Pos{Line: 1, Column: 8, Byte: 16},
				},
			},
			{
				Type:      lang.TokenAttrName,
				Modifiers: []lang.SemanticTokenModifier{},
				Range: hcl.Range{
					Filename: "test.tf",
					Start:    hcl.Pos{Line: 2, Column: 3, Byte: 21},
					End:      hcl.Pos{Line: 2, Column: 11, Byte: 29},
				},
			},
			{
				Type: lang.TokenAttrName,
				Modifiers: []lang.SemanticTokenModifier{
					lang.TokenModifierDependent,
				},
				Range: hcl.Range{
					Filename: "test.tf",
					Start:    hcl.Pos{Line: 3, Column: 3, Byte: 46},
					End:      hcl.Pos{Line: 3, Column: 11, Byte: 54},
				},
			},
			{
				Type: lang.TokenAttrName,
				Modifiers: []lang.SemanticTokenModifier{
					lang.TokenModifierDependent,
				},
				Range: hcl.Range{
					Filename: "test.tf",
					Start:    hcl.Pos{Line: 4, Column: 3, Byte: 62},
					End:      hcl.Pos{Line: 4, Column: 12, Byte: 71},
				},
			},
		},
		ClientCaps: protocol.SemanticTokensClientCapabilities{
			TokenTypes:     serverTokenTypes.AsStrings(),
			TokenModifiers: serverTokenModifiers.AsStrings(),
		},
	}
	data := te.Encode()
	expectedData := []uint32{
		0, 0, 7, 10, 0,
		0, 8, 8, 11, 0,
		1, 2, 8, 9, 0,
		1, 2, 8, 9, 2,
		1, 2, 9, 9, 2,
	}

	if diff := cmp.Diff(expectedData, data); diff != "" {
		t.Fatalf("unexpected encoded data.\nexpected: %#v\ngiven:    %#v",
			expectedData, data)
	}
}

func TestTokenEncoder_unsupported(t *testing.T) {
	bytes := []byte(`myblock "mytype" {
  str_attr = "something"
  num_attr = 42
  bool_attr = true
}`)
	te := &TokenEncoder{
		Lines: source.MakeSourceLines("test.tf", bytes),
		Tokens: []lang.SemanticToken{
			{
				Type: lang.TokenBlockType,
				Range: hcl.Range{
					Filename: "test.tf",
					Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
					End:      hcl.Pos{Line: 1, Column: 8, Byte: 7},
				},
			},
			{
				Type:      lang.TokenBlockLabel,
				Modifiers: []lang.SemanticTokenModifier{},
				Range: hcl.Range{
					Filename: "test.tf",
					Start:    hcl.Pos{Line: 1, Column: 9, Byte: 8},
					End:      hcl.Pos{Line: 1, Column: 8, Byte: 16},
				},
			},
			{
				Type:      lang.TokenAttrName,
				Modifiers: []lang.SemanticTokenModifier{},
				Range: hcl.Range{
					Filename: "test.tf",
					Start:    hcl.Pos{Line: 2, Column: 3, Byte: 21},
					End:      hcl.Pos{Line: 2, Column: 11, Byte: 29},
				},
			},
			{
				Type: lang.TokenAttrName,
				Modifiers: []lang.SemanticTokenModifier{
					lang.TokenModifierDependent,
				},
				Range: hcl.Range{
					Filename: "test.tf",
					Start:    hcl.Pos{Line: 3, Column: 3, Byte: 46},
					End:      hcl.Pos{Line: 3, Column: 11, Byte: 54},
				},
			},
			{
				Type: lang.TokenAttrName,
				Modifiers: []lang.SemanticTokenModifier{
					lang.TokenModifierDependent,
				},
				Range: hcl.Range{
					Filename: "test.tf",
					Start:    hcl.Pos{Line: 4, Column: 3, Byte: 62},
					End:      hcl.Pos{Line: 4, Column: 12, Byte: 71},
				},
			},
		},
		ClientCaps: protocol.SemanticTokensClientCapabilities{
			TokenTypes:     []string{"hcl-blockType", "hcl-attrName"},
			TokenModifiers: []string{},
		},
	}
	data := te.Encode()
	expectedData := []uint32{
		0, 0, 7, 1, 0,
		1, 2, 8, 0, 0,
		1, 2, 8, 0, 0,
		1, 2, 9, 0, 0,
	}

	if diff := cmp.Diff(expectedData, data); diff != "" {
		t.Fatalf("unexpected encoded data.\nexpected: %#v\ngiven:    %#v",
			expectedData, data)
	}
}
