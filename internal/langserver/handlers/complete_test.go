// Copyright (c) The OpenTofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2024 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-version"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/opentofu/tofu-ls/internal/langserver"
	"github.com/opentofu/tofu-ls/internal/langserver/session"
	"github.com/opentofu/tofu-ls/internal/state"
	"github.com/opentofu/tofu-ls/internal/testutils"
	"github.com/opentofu/tofu-ls/internal/tofu/exec"
	"github.com/opentofu/tofu-ls/internal/walker"
	"github.com/stretchr/testify/mock"
)

func TestModuleCompletion_withoutInitialization(t *testing.T) {
	ls := langserver.NewLangServerMock(t, NewMockSession(nil))
	stop := ls.Start(t)
	defer stop()

	ls.CallAndExpectError(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/main.tf"
			},
			"position": {
				"character": 0,
				"line": 1
			}
		}`, TempDir(t).URI)}, session.SessionNotInitialized.Err())
}

func TestModuleCompletion_withValidData_basic(t *testing.T) {
	test := func(t *testing.T, languageID string) {
		tmpDir := TempDir(t)
		InitPluginCache(t, tmpDir.Path())

		err := os.WriteFile(filepath.Join(tmpDir.Path(), "main.tf"), []byte("provider \"test\" {\n\n}\n"), 0o755)
		if err != nil {
			t.Fatal(err)
		}

		var testSchema tfjson.ProviderSchemas
		err = json.Unmarshal([]byte(testModuleSchemaOutput), &testSchema)
		if err != nil {
			t.Fatal(err)
		}

		ss, err := state.NewStateStore()
		if err != nil {
			t.Fatal(err)
		}

		wc := walker.NewWalkerCollector()

		ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
			StateStore: ss,
			TofuCalls: &exec.TofuMockCalls{
				PerWorkDir: map[string][]*mock.Call{
					tmpDir.Path(): {
						{
							Method:        "Version",
							Repeatability: 1,
							Arguments: []interface{}{
								mock.AnythingOfType(""),
							},
							ReturnArguments: []interface{}{
								version.Must(version.NewVersion("1.6.0")),
								nil,
								nil,
							},
						},
						{
							Method:        "GetExecPath",
							Repeatability: 1,
							ReturnArguments: []interface{}{
								"",
							},
						},
						{
							Method:        "ProviderSchemas",
							Repeatability: 1,
							Arguments: []interface{}{
								mock.AnythingOfType(""),
							},
							ReturnArguments: []interface{}{
								&testSchema,
								nil,
							},
						},
					},
				},
			},
			WalkerCollector: wc,
		}))
		stop := ls.Start(t)
		defer stop()

		ls.Call(t, &langserver.CallRequest{
			Method: "initialize",
			ReqParams: fmt.Sprintf(`{
		"capabilities": {},
		"rootUri": %q,
		"processId": 12345
	}`, tmpDir.URI)})
		waitForWalkerPath(t, ss, wc, tmpDir)
		ls.Notify(t, &langserver.CallRequest{
			Method:    "initialized",
			ReqParams: "{}",
		})
		ls.Call(t, &langserver.CallRequest{
			Method: "textDocument/didOpen",
			ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "opentofu",
			"text": "provider \"test\" {\n\n}\n",
			"uri": "%s/main.tf"
		}
	}`, tmpDir.URI)})
		waitForAllJobs(t, ss)

		ls.CallAndExpectResponse(t, &langserver.CallRequest{
			Method: "textDocument/completion",
			ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/main.tf"
			},
			"position": {
				"character": 0,
				"line": 1
			}
		}`, tmpDir.URI)}, `{
			"jsonrpc": "2.0",
			"id": 3,
			"result": {
				"isIncomplete": false,
				"items": [
					{
						"label": "alias",
						"kind": 10,
						"detail": "optional, string",
						"documentation": "Alias for using the same provider with different configurations for different resources, e.g. eu-west",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 1,
									"character": 0
								},
								"end": {
									"line": 1,
									"character": 0
								}
							},
							"newText": "alias"
						}
					},
					{
						"label": "anonymous",
						"kind": 10,
						"detail": "optional, number",
						"documentation": "Desc 1",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 1,
									"character": 0
								},
								"end": {
									"line": 1,
									"character": 0
								}
							},
							"newText": "anonymous"
						}
					},
					{
						"label": "base_url",
						"kind": 10,
						"detail": "optional, string",
						"documentation": "Desc 2",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 1,
									"character": 0
								},
								"end": {
									"line": 1,
									"character": 0
								}
							},
							"newText": "base_url"
						}
					},
					{
						"label": "individual",
						"kind": 10,
						"detail": "optional, bool",
						"documentation": "Desc 3",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 1,
									"character": 0
								},
								"end": {
									"line": 1,
									"character": 0
								}
							},
							"newText": "individual"
						}
					},
					{
						"label": "version",
						"kind": 10,
						"detail": "optional, string",
						"documentation": "Specifies a version constraint for the provider. e.g. ~\u003e 1.0.\nDEPRECATED: Use required_providers block to manage provider version instead.",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 1,
									"character": 0
								},
								"end": {
									"line": 1,
									"character": 0
								}
							},
							"newText": "version"
						}
					}
				]
			}
		}`)
	}
	t.Run("opentofu", func(t *testing.T) {
		test(t, "opentofu")
	})
	//Checking that we can also complete in terraform language ID
	t.Run("terraform", func(t *testing.T) {
		test(t, "terraform")
	})
}

// verify that for unknown new versions we serve latest available schema
func TestModuleCompletion_withValidData_tooNewVersion(t *testing.T) {
	tmpDir := TempDir(t)
	InitPluginCache(t, tmpDir.Path())

	err := os.WriteFile(filepath.Join(tmpDir.Path(), "main.tf"), []byte("variable \"test\" {\n\n}\n"), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	var testSchema tfjson.ProviderSchemas
	err = json.Unmarshal([]byte(testModuleSchemaOutput), &testSchema)
	if err != nil {
		t.Fatal(err)
	}

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}

	wc := walker.NewWalkerCollector()

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		StateStore: ss,
		TofuCalls: &exec.TofuMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Path(): {
					{
						Method:        "Version",
						Repeatability: 1,
						Arguments: []interface{}{
							mock.AnythingOfType(""),
						},
						ReturnArguments: []interface{}{
							version.Must(version.NewVersion("999.999.999")),
							nil,
							nil,
						},
					},
					{
						Method:        "ProviderSchemas",
						Repeatability: 1,
						Arguments: []interface{}{
							mock.AnythingOfType(""),
						},
						ReturnArguments: []interface{}{
							&testSchema,
							nil,
						},
					},
				},
			},
		},
		WalkerCollector: wc,
	}))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
		"capabilities": {},
		"rootUri": %q,
		"processId": 12345
	}`, tmpDir.URI)})
	waitForWalkerPath(t, ss, wc, tmpDir)
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "opentofu",
			"text": "variable \"test\" {\n\n}\n",
			"uri": "%s/main.tf"
		}
	}`, tmpDir.URI)})
	waitForAllJobs(t, ss)

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/main.tf"
			},
			"position": {
				"character": 0,
				"line": 1
			}
		}`, tmpDir.URI)}, `{
			"jsonrpc": "2.0",
			"id": 3,
			"result": {
				"isIncomplete": false,
				"items": [
					{
						"label": "default",
						"kind": 10,
						"detail": "optional, any type",
						"documentation": "Default value to use when variable is not explicitly set",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 1,
									"character": 0
								},
								"end": {
									"line": 1,
									"character": 0
								}
							},
							"newText": "default"
						}
					},
					{
						"label": "description",
						"kind": 10,
						"detail": "optional, string",
						"documentation": "Description to document the purpose of the variable and what value is expected",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 1,
									"character": 0
								},
								"end": {
									"line": 1,
									"character": 0
								}
							},
							"newText": "description"
						}
					},
					{
						"label": "nullable",
						"kind": 10,
						"detail": "optional, bool",
						"documentation": "Specifies whether null is a valid value for this variable",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 1,
									"character": 0
								},
								"end": {
									"line": 1,
									"character": 0
								}
							},
							"newText": "nullable"
						}
					},
					{
						"label": "sensitive",
						"kind": 10,
						"detail": "optional, bool",
						"documentation": "Whether the variable contains sensitive material and should be hidden in the UI",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 1,
									"character": 0
								},
								"end": {
									"line": 1,
									"character": 0
								}
							},
							"newText": "sensitive"
						}
					},
					{
						"label": "type",
						"kind": 10,
						"detail": "optional, type",
						"documentation": "Type constraint restricting the type of value to accept, e.g. string or list(string)",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 1,
									"character": 0
								},
								"end": {
									"line": 1,
									"character": 0
								}
							},
							"newText": "type"
						}
					},
					{
						"label": "validation",
						"kind": 7,
						"detail": "Block",
						"documentation": "Custom validation rule to restrict what value is expected for the variable",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 1,
									"character": 0
								},
								"end": {
									"line": 1,
									"character": 0
								}
							},
							"newText": "validation"
						}
					}
				]
			}
		}`)
}

func TestModuleCompletion_withValidDataAndSnippets(t *testing.T) {
	tmpDir := TempDir(t)
	InitPluginCache(t, tmpDir.Path())
	err := os.WriteFile(filepath.Join(tmpDir.Path(), "main.tf"), []byte("provider \"test\" {\n\n}\n"), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	var testSchema tfjson.ProviderSchemas
	err = json.Unmarshal([]byte(testModuleSchemaOutput), &testSchema)
	if err != nil {
		t.Fatal(err)
	}

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := walker.NewWalkerCollector()

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TofuCalls: &exec.TofuMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Path(): {
					{
						Method:        "Version",
						Repeatability: 1,
						Arguments: []interface{}{
							mock.AnythingOfType(""),
						},
						ReturnArguments: []interface{}{
							version.Must(version.NewVersion("1.6.0")),
							nil,
							nil,
						},
					},
					{
						Method:        "GetExecPath",
						Repeatability: 1,
						ReturnArguments: []interface{}{
							"",
						},
					},
					{
						Method:        "ProviderSchemas",
						Repeatability: 1,
						Arguments: []interface{}{
							mock.AnythingOfType(""),
						},
						ReturnArguments: []interface{}{
							&testSchema,
							nil,
						},
					},
				},
			},
		},
		StateStore:      ss,
		WalkerCollector: wc,
	}))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
		"capabilities": {
			"textDocument": {
        "completion": {
          "completionItem": {
            "snippetSupport": true
          }
        }
      }
		},
		"rootUri": %q,
		"processId": 12345
	}`, tmpDir.URI)})
	waitForWalkerPath(t, ss, wc, tmpDir)
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "opentofu",
			"text": "provider \"test\" {\n\n}\n",
			"uri": "%s/main.tf"
		}
	}`, tmpDir.URI)})
	waitForAllJobs(t, ss)

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/main.tf"
			},
			"position": {
				"character": 0,
				"line": 1
			}
		}`, tmpDir.URI)}, `{
			"jsonrpc": "2.0",
			"id": 3,
			"result": {
				"isIncomplete": false,
				"items": [
					{
						"label": "alias",
						"kind": 10,
						"detail": "optional, string",
						"documentation": "Alias for using the same provider with different configurations for different resources, e.g. eu-west",
						"insertTextFormat": 2,
						"textEdit": {
							"range": {
								"start": {
									"line": 1,
									"character": 0
								},
								"end": {
									"line": 1,
									"character": 0
								}
							},
							"newText": "alias = \"${1:value}\""
						}
					},
					{
						"label": "anonymous",
						"kind": 10,
						"detail": "optional, number",
						"documentation": "Desc 1",
						"insertTextFormat": 2,
						"textEdit": {
							"range": {
								"start": {
									"line": 1,
									"character": 0
								},
								"end": {
									"line": 1,
									"character": 0
								}
							},
							"newText": "anonymous = "
						},
						"command": {
							"title": "Suggest",
							"command": "editor.action.triggerSuggest"
						}
					},
					{
						"label": "base_url",
						"kind": 10,
						"detail": "optional, string",
						"documentation": "Desc 2",
						"insertTextFormat": 2,
						"textEdit": {
							"range": {
								"start": {
									"line": 1,
									"character": 0
								},
								"end": {
									"line": 1,
									"character": 0
								}
							},
							"newText": "base_url = "
						},
						"command": {
							"title": "Suggest",
							"command": "editor.action.triggerSuggest"
						}
					},
					{
						"label": "individual",
						"kind": 10,
						"detail": "optional, bool",
						"documentation": "Desc 3",
						"insertTextFormat": 2,
						"textEdit": {
							"range": {
								"start": {
									"line": 1,
									"character": 0
								},
								"end": {
									"line": 1,
									"character": 0
								}
							},
							"newText": "individual = "
						},
						"command": {
							"title": "Suggest",
							"command": "editor.action.triggerSuggest"
						}
					},
					{
						"label": "version",
						"kind": 10,
						"detail": "optional, string",
						"documentation": "Specifies a version constraint for the provider. e.g. ~\u003e 1.0.\nDEPRECATED: Use required_providers block to manage provider version instead.",
						"insertTextFormat": 2,
						"textEdit": {
							"range": {
								"start": {
									"line": 1,
									"character": 0
								},
								"end": {
									"line": 1,
									"character": 0
								}
							},
							"newText": "version = \"${1:value}\""
						}
					}
				]
			}
		}`)
}

var testModuleSchemaOutput = `{
	"format_version": "0.1",
	"provider_schemas": {
		"test/test": {
			"provider": {
				"version": 0,
				"block": {
					"attributes": {
						"anonymous": {
							"type": "number",
							"description": "Desc 1",
							"description_kind": "plaintext",
							"optional": true
						},
						"base_url": {
							"type": "string",
							"description": "Desc **2**",
							"description_kind": "markdown",
							"optional": true
						},
						"individual": {
							"type": "bool",
							"description": "Desc _3_",
							"description_kind": "markdown",
							"optional": true
						}
					}
				}
			},
			"resource_schemas": {
				"test_resource_1": {
					"version": 0,
					"block": {
						"description": "Resource 1 description",
						"description_kind": "markdown",
						"attributes": {
							"deprecated_attr": {
								"type": "string",
								"deprecated": true
							}
						}
					}
				},
				"test_resource_2": {
					"version": 0,
					"block": {
						"description_kind": "markdown",
						"attributes": {
							"optional_attr": {
								"type": "string",
								"description_kind": "plain",
								"optional": true
							}
						},
						"block_types": {
							"setting": {
								"nesting_mode": "set",
								"block": {
									"attributes": {
										"name": {
											"type": "string",
											"description_kind": "plain",
											"required": true
										},
										"value": {
											"type": "string",
											"description_kind": "plain",
											"required": true
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}`

func TestVarsCompletion_withValidData(t *testing.T) {
	tmpDir := TempDir(t)
	InitPluginCache(t, tmpDir.Path())

	var testSchema tfjson.ProviderSchemas
	err := json.Unmarshal([]byte(testModuleSchemaOutput), &testSchema)
	if err != nil {
		t.Fatal(err)
	}

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := walker.NewWalkerCollector()

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TofuCalls: &exec.TofuMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Path(): {
					{
						Method:        "Version",
						Repeatability: 1,
						Arguments: []interface{}{
							mock.AnythingOfType(""),
						},
						ReturnArguments: []interface{}{
							version.Must(version.NewVersion("0.12.0")),
							nil,
							nil,
						},
					},
					{
						Method:        "GetExecPath",
						Repeatability: 1,
						ReturnArguments: []interface{}{
							"",
						},
					},
					{
						Method:        "ProviderSchemas",
						Repeatability: 1,
						Arguments: []interface{}{
							mock.AnythingOfType(""),
						},
						ReturnArguments: []interface{}{
							&testSchema,
							nil,
						},
					},
				},
			},
		},
		StateStore:      ss,
		WalkerCollector: wc,
	}))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
		"capabilities": {},
		"rootUri": %q,
		"processId": 12345
	}`, tmpDir.URI)})
	waitForWalkerPath(t, ss, wc, tmpDir)
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "opentofu",
			"text": "variable \"test\" {\n type=string\n}\n",
			"uri": "%s/variables.tf"
		}
	}`, tmpDir.URI)})
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "opentofu-vars",
			"uri": "%s/terraform.tfvars"
		}
	}`, tmpDir.URI)})
	waitForAllJobs(t, ss)

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/terraform.tfvars"
			},
			"position": {
				"character": 0,
				"line": 0
			}
		}`, tmpDir.URI)}, `{
			"jsonrpc": "2.0",
			"id": 4,
			"result": {
				"isIncomplete": false,
				"items": [
					{
						"label": "test",
						"kind": 10,
						"detail": "required, string",
						"insertTextFormat":1,
						"textEdit": {
							"range": {"start":{"line":0,"character":0}, "end":{"line":0,"character":0}},
							"newText":"test"
						}
					}
				]
			}
		}`)
}

func TestCompletion_moduleWithValidData(t *testing.T) {
	tmpDir := TempDir(t)

	writeContentToFile(t, filepath.Join(tmpDir.Path(), "submodule", "main.tf"), `variable "testvar" {
	type = string
}

output "testout" {
	value = 42
}
`)
	mainCfg := `module "refname" {
  source = "./submodule"

}

output "test" {
  value = 42
}
`
	writeContentToFile(t, filepath.Join(tmpDir.Path(), "main.tf"), mainCfg)
	mainCfg = `module "refname" {
  source = "./submodule"

}

output "test" {
  value = module.refname.
}
`

	tfExec := testutils.NewTestingExecutor(t, tmpDir.Path())
	err := tfExec.Get(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	var testSchema tfjson.ProviderSchemas
	err = json.Unmarshal([]byte(testModuleSchemaOutput), &testSchema)
	if err != nil {
		t.Fatal(err)
	}

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := walker.NewWalkerCollector()
	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TofuCalls: &exec.TofuMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Path(): {
					{
						Method:        "Version",
						Repeatability: 1,
						Arguments: []interface{}{
							mock.AnythingOfType(""),
						},
						ReturnArguments: []interface{}{
							version.Must(version.NewVersion("1.6.0")),
							nil,
							nil,
						},
					},
					{
						Method:        "GetExecPath",
						Repeatability: 1,
						ReturnArguments: []interface{}{
							"",
						},
					},
					{
						Method:        "ProviderSchemas",
						Repeatability: 1,
						Arguments: []interface{}{
							mock.AnythingOfType(""),
						},
						ReturnArguments: []interface{}{
							&testSchema,
							nil,
						},
					},
				},
			},
		},
		StateStore:      ss,
		WalkerCollector: wc,
	}))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
		"capabilities": {},
		"rootUri": %q,
		"processId": 12345
	}`, tmpDir.URI)})
	waitForWalkerPath(t, ss, wc, tmpDir)
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "opentofu",
			"text": %q,
			"uri": "%s/main.tf"
		}
	}`, mainCfg, tmpDir.URI)})
	waitForAllJobs(t, ss)

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/main.tf"
			},
			"position": {
				"character": 0,
				"line": 2
			}
		}`, tmpDir.URI)}, `{
			"jsonrpc": "2.0",
			"id": 3,
			"result": {
				"isIncomplete": false,
				"items": [
					{
					  "label": "count",
					  "kind": 10,
					  "detail": "optional, number",
					  "documentation": "Total number of instances of this block.\n\nNote: A given block cannot use both count and for_each.",
					  "insertTextFormat": 1,
					  "textEdit": {
						"range": {
						  "start": { "line": 2, "character": 0 },
						  "end": { "line": 2, "character": 0 }
						},
						"newText": "count"
					  }
					},
					{
					  "label": "depends_on",
					  "kind": 10,
					  "detail": "optional, set of reference",
					  "documentation": "Set of references to hidden dependencies, e.g. other resources or data sources",
					  "insertTextFormat": 1,
					  "textEdit": {
						"range": {
						  "start": { "line": 2, "character": 0 },
						  "end": { "line": 2, "character": 0 }
						},
						"newText": "depends_on"
					  }
					},
					{
					  "label": "for_each",
					  "kind": 10,
					  "detail": "optional, map of any single type or set of string or object",
					  "documentation": "A meta-argument that accepts a map or a set of strings, and creates an instance for each item in that map or set.\n\nNote: A given block cannot use both count and for_each.",
					  "insertTextFormat": 1,
					  "textEdit": {
						"range": {
						  "start": { "line": 2, "character": 0 },
						  "end": { "line": 2, "character": 0 }
						},
						"newText": "for_each"
					  }
					},
					{
					  "label": "providers",
					  "kind": 10,
					  "detail": "optional, map of provider references",
					  "documentation": "Explicit mapping of providers which the module uses",
					  "insertTextFormat": 1,
					  "textEdit": {
						"range": {
						  "start": { "line": 2, "character": 0 },
						  "end": { "line": 2, "character": 0 }
						},
						"newText": "providers"
					  }
					},
					{
					  "label": "testvar",
					  "kind": 10,
					  "detail": "required, string",
					  "insertTextFormat": 1,
					  "textEdit": {
						"range": {
						  "start": { "line": 2, "character": 0 },
						  "end": { "line": 2, "character": 0 }
						},
						"newText": "testvar"
					  }
					},
					{
					  "label": "version",
					  "kind": 10,
					  "detail": "optional, string",
					  "documentation": "Constraint to set the version of the module, e.g. ~\u003e 1.0. Only applicable to modules in a module registry.",
					  "insertTextFormat": 1,
					  "textEdit": {
						"range": {
						  "start": { "line": 2, "character": 0 },
						  "end": { "line": 2, "character": 0 }
						},
						"newText": "version"
					  }
					}
				  ]
			}
		}`)

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/main.tf"
			},
			"position": {
				"character": 25,
				"line": 6
			}
		}`, tmpDir.URI)}, `{
			"jsonrpc": "2.0",
			"id": 4,
			"result": {
				"isIncomplete": false,
				"items": [
					{
						"label": "module.refname.testout",
						"kind": 6,
						"detail": "number",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 6,
									"character": 10
								},
								"end": {
									"line": 6,
									"character": 25
								}
							},
							"newText": "module.refname.testout"
						}
					}
				]
			}
		}`)
}

func TestCompletion_multipleModulesWithValidData(t *testing.T) {
	tmpDir := TempDir(t)

	writeContentToFile(t, filepath.Join(tmpDir.Path(), "submodule-alpha", "main.tf"), `
variable "alpha-var" {
	type = string
}

output "alpha-out" {
	value = 1
}
`)
	writeContentToFile(t, filepath.Join(tmpDir.Path(), "submodule-beta", "main.tf"), `
variable "beta-var" {
	type = number
}

output "beta-out" {
	value = 2
}
`)
	mainCfg := `module "alpha" {
  source = "./submodule-alpha"

}
module "beta" {
  source = "./submodule-beta"

}

output "test" {
	value = 2
}
`
	writeContentToFile(t, filepath.Join(tmpDir.Path(), "main.tf"), mainCfg)
	mainCfg = `module "alpha" {
  source = "./submodule-alpha"

}
module "beta" {
  source = "./submodule-beta"

}

output "test" {
  value = module.
}
`

	tfExec := testutils.NewTestingExecutor(t, tmpDir.Path())
	err := tfExec.Get(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	var testSchema tfjson.ProviderSchemas
	err = json.Unmarshal([]byte(testModuleSchemaOutput), &testSchema)
	if err != nil {
		t.Fatal(err)
	}

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := walker.NewWalkerCollector()

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TofuCalls: &exec.TofuMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Path(): {
					{
						Method:        "Version",
						Repeatability: 1,
						Arguments: []interface{}{
							mock.AnythingOfType(""),
						},
						ReturnArguments: []interface{}{
							version.Must(version.NewVersion("1.6.0")),
							nil,
							nil,
						},
					},
					{
						Method:        "GetExecPath",
						Repeatability: 1,
						ReturnArguments: []interface{}{
							"",
						},
					},
					{
						Method:        "ProviderSchemas",
						Repeatability: 1,
						Arguments: []interface{}{
							mock.AnythingOfType(""),
						},
						ReturnArguments: []interface{}{
							&testSchema,
							nil,
						},
					},
				},
			},
		},
		StateStore:      ss,
		WalkerCollector: wc,
	}))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
		"capabilities": {},
		"rootUri": %q,
		"processId": 12345
	}`, tmpDir.URI)})
	waitForWalkerPath(t, ss, wc, tmpDir)
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "opentofu",
			"text": %q,
			"uri": "%s/main.tf"
		}
	}`, mainCfg, tmpDir.URI)})
	waitForAllJobs(t, ss)

	// first module
	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/main.tf"
			},
			"position": {
				"character": 0,
				"line": 2
			}
		}`, tmpDir.URI)}, `{
			"jsonrpc": "2.0",
			"id": 3,
			"result": {
				"isIncomplete": false,
				"items": [
					{
					  "label": "alpha-var",
					  "kind": 10,
					  "detail": "required, string",
					  "insertTextFormat": 1,
					  "textEdit": {
						"range": {
						  "start": { "line": 2, "character": 0 },
						  "end": { "line": 2, "character": 0 }
						},
						"newText": "alpha-var"
					  }
					},
					{
					  "label": "count",
					  "kind": 10,
					  "detail": "optional, number",
					  "documentation": "Total number of instances of this block.\n\nNote: A given block cannot use both count and for_each.",
					  "insertTextFormat": 1,
					  "textEdit": {
						"range": {
						  "start": { "line": 2, "character": 0 },
						  "end": { "line": 2, "character": 0 }
						},
						"newText": "count"
					  }
					},
					{
					  "label": "depends_on",
					  "kind": 10,
					  "detail": "optional, set of reference",
					  "documentation": "Set of references to hidden dependencies, e.g. other resources or data sources",
					  "insertTextFormat": 1,
					  "textEdit": {
						"range": {
						  "start": { "line": 2, "character": 0 },
						  "end": { "line": 2, "character": 0 }
						},
						"newText": "depends_on"
					  }
					},
					{
					  "label": "for_each",
					  "kind": 10,
					  "detail": "optional, map of any single type or set of string or object",
					  "documentation": "A meta-argument that accepts a map or a set of strings, and creates an instance for each item in that map or set.\n\nNote: A given block cannot use both count and for_each.",
					  "insertTextFormat": 1,
					  "textEdit": {
						"range": {
						  "start": { "line": 2, "character": 0 },
						  "end": { "line": 2, "character": 0 }
						},
						"newText": "for_each"
					  }
					},
					{
					  "label": "providers",
					  "kind": 10,
					  "detail": "optional, map of provider references",
					  "documentation": "Explicit mapping of providers which the module uses",
					  "insertTextFormat": 1,
					  "textEdit": {
						"range": {
						  "start": { "line": 2, "character": 0 },
						  "end": { "line": 2, "character": 0 }
						},
						"newText": "providers"
					  }
					},
					{
					  "label": "version",
					  "kind": 10,
					  "detail": "optional, string",
					  "documentation": "Constraint to set the version of the module, e.g. ~\u003e 1.0. Only applicable to modules in a module registry.",
					  "insertTextFormat": 1,
					  "textEdit": {
						"range": {
						  "start": { "line": 2, "character": 0 },
						  "end": { "line": 2, "character": 0 }
						},
						"newText": "version"
					  }
					}
			  ]
			}
		}`)
	// second module
	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/main.tf"
			},
			"position": {
				"character": 0,
				"line": 6
			}
		}`, tmpDir.URI)}, `{
			"jsonrpc": "2.0",
			"id": 4,
			"result": {
				"isIncomplete": false,
				 "items": [
					{
					  "label": "beta-var",
					  "kind": 10,
					  "detail": "required, number",
					  "insertTextFormat": 1,
					  "textEdit": {
						"range": {
						  "start": { "line": 6, "character": 0 },
						  "end": { "line": 6, "character": 0 }
						},
						"newText": "beta-var"
					  }
					},
					{
					  "label": "count",
					  "kind": 10,
					  "detail": "optional, number",
					  "documentation": "Total number of instances of this block.\n\nNote: A given block cannot use both count and for_each.",
					  "insertTextFormat": 1,
					  "textEdit": {
						"range": {
						  "start": { "line": 6, "character": 0 },
						  "end": { "line": 6, "character": 0 }
						},
						"newText": "count"
					  }
					},
					{
					  "label": "depends_on",
					  "kind": 10,
					  "detail": "optional, set of reference",
					  "documentation": "Set of references to hidden dependencies, e.g. other resources or data sources",
					  "insertTextFormat": 1,
					  "textEdit": {
						"range": {
						  "start": { "line": 6, "character": 0 },
						  "end": { "line": 6, "character": 0 }
						},
						"newText": "depends_on"
					  }
					},
					{
					  "label": "for_each",
					  "kind": 10,
					  "detail": "optional, map of any single type or set of string or object",
					  "documentation": "A meta-argument that accepts a map or a set of strings, and creates an instance for each item in that map or set.\n\nNote: A given block cannot use both count and for_each.",
					  "insertTextFormat": 1,
					  "textEdit": {
						"range": {
						  "start": { "line": 6, "character": 0 },
						  "end": { "line": 6, "character": 0 }
						},
						"newText": "for_each"
					  }
					},
					{
					  "label": "providers",
					  "kind": 10,
					  "detail": "optional, map of provider references",
					  "documentation": "Explicit mapping of providers which the module uses",
					  "insertTextFormat": 1,
					  "textEdit": {
						"range": {
						  "start": { "line": 6, "character": 0 },
						  "end": { "line": 6, "character": 0 }
						},
						"newText": "providers"
					  }
					},
					{
					  "label": "version",
					  "kind": 10,
					  "detail": "optional, string",
					  "documentation": "Constraint to set the version of the module, e.g. ~\u003e 1.0. Only applicable to modules in a module registry.",
					  "insertTextFormat": 1,
					  "textEdit": {
						"range": {
						  "start": { "line": 6, "character": 0 },
						  "end": { "line": 6, "character": 0 }
						},
						"newText": "version"
					  }
					}
				  ]
			}
		}`)
	// outputs
	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/main.tf"
			},
			"position": {
				"character": 17,
				"line": 10
			}
		}`, tmpDir.URI)}, `{
			"jsonrpc": "2.0",
			"id": 5,
			"result": {
				"isIncomplete": false,
				"items": [
					{
						"label": "module.alpha",
						"kind": 6,
						"detail": "object",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 10,
									"character": 10
								},
								"end": {
									"line": 10,
									"character": 17
								}
							},
							"newText": "module.alpha"
						}
					},
					{
						"label": "module.beta",
						"kind": 6,
						"detail": "object",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 10,
									"character": 10
								},
								"end": {
									"line": 10,
									"character": 17
								}
							},
							"newText": "module.beta"
						}
					}
				]
			}
		}`)
}

func TestVarReferenceCompletion_withValidData(t *testing.T) {
	tmpDir := TempDir(t)
	InitPluginCache(t, tmpDir.Path())

	variableDecls := `variable "aaa" {}
variable "bbb" {}
variable "ccc" {}
`
	f, err := os.Create(filepath.Join(tmpDir.Path(), "variables.tf"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString(variableDecls)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	var testSchema tfjson.ProviderSchemas
	err = json.Unmarshal([]byte(testModuleSchemaOutput), &testSchema)
	if err != nil {
		t.Fatal(err)
	}

	ss, err := state.NewStateStore()
	if err != nil {
		t.Fatal(err)
	}
	wc := walker.NewWalkerCollector()

	ls := langserver.NewLangServerMock(t, NewMockSession(&MockSessionInput{
		TofuCalls: &exec.TofuMockCalls{
			PerWorkDir: map[string][]*mock.Call{
				tmpDir.Path(): {
					{
						Method:        "Version",
						Repeatability: 1,
						Arguments: []interface{}{
							mock.AnythingOfType(""),
						},
						ReturnArguments: []interface{}{
							version.Must(version.NewVersion("0.12.0")),
							nil,
							nil,
						},
					},
					{
						Method:        "GetExecPath",
						Repeatability: 1,
						ReturnArguments: []interface{}{
							"",
						},
					},
					{
						Method:        "ProviderSchemas",
						Repeatability: 1,
						Arguments: []interface{}{
							mock.AnythingOfType(""),
						},
						ReturnArguments: []interface{}{
							&testSchema,
							nil,
						},
					},
				},
			},
		},
		StateStore:      ss,
		WalkerCollector: wc,
	}))
	stop := ls.Start(t)
	defer stop()

	ls.Call(t, &langserver.CallRequest{
		Method: "initialize",
		ReqParams: fmt.Sprintf(`{
		"capabilities": {},
		"rootUri": %q,
		"processId": 12345
	}`, tmpDir.URI)})
	waitForWalkerPath(t, ss, wc, tmpDir)
	ls.Notify(t, &langserver.CallRequest{
		Method:    "initialized",
		ReqParams: "{}",
	})
	ls.Call(t, &langserver.CallRequest{
		Method: "textDocument/didOpen",
		ReqParams: fmt.Sprintf(`{
		"textDocument": {
			"version": 0,
			"languageId": "opentofu",
			"text": "output \"test\" {\n  value = var.\n}\n",
			"uri": "%s/outputs.tf"
		}
	}`, tmpDir.URI)})
	waitForAllJobs(t, ss)

	ls.CallAndExpectResponse(t, &langserver.CallRequest{
		Method: "textDocument/completion",
		ReqParams: fmt.Sprintf(`{
			"textDocument": {
				"uri": "%s/outputs.tf"
			},
			"position": {
				"character": 14,
				"line": 1
			}
		}`, tmpDir.URI)}, `{
			"jsonrpc": "2.0",
			"id": 3,
			"result": {
				"isIncomplete": false,
				"items": [
					{
						"label": "var.aaa",
						"kind": 6,
						"detail": "dynamic",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 1,
									"character": 10
								},
								"end": {
									"line": 1,
									"character": 14
								}
							},
							"newText": "var.aaa"
						}
					},
					{
						"label": "var.bbb",
						"kind": 6,
						"detail": "dynamic",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 1,
									"character": 10
								},
								"end": {
									"line": 1,
									"character": 14
								}
							},
							"newText": "var.bbb"
						}
					},
					{
						"label": "var.ccc",
						"kind": 6,
						"detail": "dynamic",
						"insertTextFormat": 1,
						"textEdit": {
							"range": {
								"start": {
									"line": 1,
									"character": 10
								},
								"end": {
									"line": 1,
									"character": 14
								}
							},
							"newText": "var.ccc"
						}
					}
				]
			}
		}`)
}

func writeContentToFile(t *testing.T, path string, content string) {
	err := os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}

	_, err = f.WriteString(content)
	if err != nil {
		t.Fatal(err)
	}

	err = f.Close()
	if err != nil {
		t.Fatal(err)
	}
}
