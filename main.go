// Copyright (c) The OpenTofu Authors
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2024 HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"os"

	"github.com/mitchellh/cli"

	"github.com/opentofu/tofu-ls/internal/cmd"
)

func main() {
	c := &cli.CLI{
		Name:       "tofu-ls",
		Version:    version,
		Args:       os.Args[1:],
		HelpWriter: os.Stdout,
	}

	ui := &cli.ColoredUi{
		ErrorColor: cli.UiColorRed,
		WarnColor:  cli.UiColorYellow,
		Ui: &cli.BasicUi{
			Writer:      os.Stdout,
			Reader:      os.Stdin,
			ErrorWriter: os.Stderr,
		},
	}

	c.Commands = map[string]cli.CommandFactory{
		"serve": func() (cli.Command, error) {
			return &cmd.ServeCommand{
				Ui:      ui,
				Version: version,
			}, nil
		},
		"version": func() (cli.Command, error) {
			return &cmd.VersionCommand{
				Ui:      ui,
				Version: version,
			}, nil
		},
	}

	exitStatus, err := c.Run()
	if err != nil {
		ui.Error("Error: " + err.Error())
	}

	os.Exit(exitStatus)
}
