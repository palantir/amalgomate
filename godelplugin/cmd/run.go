// Copyright (c) 2016 Palantir Technologies Inc. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Copyright 2016 Palantir Technologies, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/palantir/amalgomate/godelplugin/amalgomateplugin"
	"github.com/palantir/amalgomate/godelplugin/amalgomateplugin/config"
)

var (
	verifyFlag bool
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run amalgomate based on project configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := readConfig(configFileFlag)
		if err != nil {
			return err
		}
		if err := os.Chdir(projectDirFlag); err != nil {
			return errors.Wrapf(err, "failed to set working directory")
		}
		param, err := cfg.ToParam()
		if err != nil {
			return err
		}
		return amalgomateplugin.Run(param, verifyFlag, cmd.OutOrStdout())
	},
}

func init() {
	runCmd.Flags().BoolVar(&verifyFlag, VerifyFlagName, false, "verify that current project matches output of amalgomate")
	RootCmd.AddCommand(runCmd)
}

func readConfig(cfgFile string) (config.Config, error) {
	cfgBytes, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		return config.Config{}, errors.Wrapf(err, "failed to read file")
	}
	upgradedBytes, err := config.UpgradeConfig(cfgBytes)
	if err != nil {
		return config.Config{}, errors.Wrapf(err, "failed to upgrade amalgomate-plugin configuration")
	}
	var cfg config.Config
	if err := yaml.UnmarshalStrict(upgradedBytes, &cfg); err != nil {
		return config.Config{}, errors.Wrapf(err, "failed to unmarshal amalgomate-plugin configuration")
	}
	return cfg, nil
}
