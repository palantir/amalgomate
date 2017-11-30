// Copyright (c) 2016 Palantir Technologies Inc. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"github.com/palantir/pkg/cobracli"
	"github.com/spf13/cobra"

	"github.com/palantir/amalgomate/cmd/amalgomate"
)

const (
	debugFlagName     = "debug"
	configFlagName    = "config"
	outputDirFlagName = "output-dir"
	pkgFlagName       = "pkg"
)

var (
	debug     bool
	config    string
	outputDir string
	pkgFlag   string
)

// AmalgomateCmd represents the base command when called without any subcommands
var AmalgomateCmd = &cobra.Command{
	Use:   "amalgomate",
	Short: "Re-package main packages into a library package",
	Long: `amalgomate is used to re-package Go programs with
a "main" package into a library package that can be called
directly in-process.

amalgomate requires a configuration YML file that specifies
the packages that should be converted from "main" packages
into library packages. An output directory and the name of
the package for the generated source files should also be
specified.

Here is an example configuration file:

packages:
  gofmt:
    main: cmd/gofmt
  ptimports:
    main: github.com/palantir/checks/ptimports

An example invocation is of the form:

  amalgomate --config config.yml --output-dir generated_src --pkg amalgomated

This invocation would amalgomate the inputs specified in "config.yml" and would
write the generated source into the "generated_src" directory with the package
name "amalgomated".`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := amalgomate.LoadConfig(config)
		if err != nil {
			return err
		}
		return amalgomate.Run(cfg, outputDir, pkgFlag)
	},
}

func Execute() int {
	return cobracli.ExecuteWithDefaultParams(AmalgomateCmd, &debug)
}

func init() {
	AmalgomateCmd.Flags().BoolVar(&debug, debugFlagName, false, "run in debug mode")

	AmalgomateCmd.Flags().StringVar(&config, configFlagName, "", "configuration file that specifies packages to be amalgomated")
	if err := AmalgomateCmd.MarkFlagRequired(configFlagName); err != nil {
		panic(err)
	}

	AmalgomateCmd.Flags().StringVar(&outputDir, outputDirFlagName, "", "directory in which amalgomated output is written")
	if err := AmalgomateCmd.MarkFlagRequired(outputDirFlagName); err != nil {
		panic(err)
	}

	AmalgomateCmd.Flags().StringVar(&pkgFlag, pkgFlagName, "", "package name of the amalgomated source that is generated")
	if err := AmalgomateCmd.MarkFlagRequired(pkgFlagName); err != nil {
		panic(err)
	}
}
