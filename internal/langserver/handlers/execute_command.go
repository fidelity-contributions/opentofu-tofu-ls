// Copyright (c) The OpenTofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2024 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"
	"fmt"

	"github.com/creachadair/jrpc2"
	lsctx "github.com/opentofu/tofu-ls/internal/context"
	"github.com/opentofu/tofu-ls/internal/langserver/cmd"
	"github.com/opentofu/tofu-ls/internal/langserver/handlers/command"
	lsp "github.com/opentofu/tofu-ls/internal/protocol"
)

func cmdHandlers(svc *service) cmd.Handlers {
	cmdHandler := &command.CmdHandler{
		StateStore: svc.stateStore,
		Logger:     svc.logger,
	}
	if svc.features != nil {
		cmdHandler.ModulesFeature = svc.features.Modules
		cmdHandler.RootModulesFeature = svc.features.RootModules
	}
	return cmd.Handlers{
		cmd.Name("rootmodules"):      removedHandler("use module.callers instead"),
		cmd.Name("module.callers"):   cmdHandler.ModuleCallersHandler,
		cmd.Name("tofu.init"):        cmdHandler.TofuInitHandler,
		cmd.Name("tofu.validate"):    cmdHandler.TofuValidateHandler,
		cmd.Name("module.calls"):     cmdHandler.ModuleCallsHandler,
		cmd.Name("module.providers"): cmdHandler.ModuleProvidersHandler,
		cmd.Name("module.opentofu"):  cmdHandler.TofuVersionRequestHandler,
	}
}

func (svc *service) WorkspaceExecuteCommand(ctx context.Context, params lsp.ExecuteCommandParams) (interface{}, error) {
	if params.Command == "editor.action.triggerSuggest" {
		// If this was actually received by the server, it means the client
		// does not support explicit suggest triggering, so we fail silently
		// TODO: Revisit once https://github.com/microsoft/language-server-protocol/issues/1117 is addressed
		return nil, nil
	}

	commandPrefix, _ := lsctx.CommandPrefix(ctx)
	handler, ok := cmdHandlers(svc).Get(params.Command, commandPrefix)
	if !ok {
		return nil, fmt.Errorf("%w: command handler not found for %q", jrpc2.MethodNotFound.Err(), params.Command)
	}

	ctx = lsctx.WithProgressToken(ctx, params.WorkDoneToken)

	return handler(ctx, cmd.ParseCommandArgs(params.Arguments))
}

func removedHandler(errorMessage string) cmd.Handler {
	return func(context.Context, cmd.CommandArgs) (interface{}, error) {
		return nil, jrpc2.Errorf(jrpc2.MethodNotFound, "REMOVED: %s", errorMessage)
	}
}
