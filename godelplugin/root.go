// Copyright (c) 2016 Palantir Technologies Inc. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Copyright 2016 Palantir Technologies, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/palantir/godel/framework/pluginapi"
	"github.com/palantir/godel/pkg/dirchecksum"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/palantir/amalgomate/cmd/amalgomate"
)

const verifyFlagName = "verify"

var (
	debugFlag      bool
	projectDirFlag string
	configFileFlag string
	verifyFlag     bool
)

var rootCmd = &cobra.Command{
	Use:   "amalgomate-plugin",
	Short: "Run amalgomate based on project configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := readConfig(configFileFlag)
		if err != nil {
			return err
		}
		if err := os.Chdir(projectDirFlag); err != nil {
			return errors.Wrapf(err, "failed to set working directory")
		}
		return runAmalgomate(cfg.ToParam(), verifyFlag, cmd.OutOrStdout())
	},
}

const indentLen = 2

func runAmalgomate(param Param, verify bool, stdout io.Writer) error {
	var verifyFailedKeys []string
	verifyFailedErrors := make(map[string]string)
	verifyFailedFn := func(name, errStr string) {
		verifyFailedKeys = append(verifyFailedKeys, name)
		verifyFailedErrors[name] = errStr
	}

	for _, k := range param.OrderedKeys {
		val := param.Amalgomators[k]
		cfg, err := amalgomate.LoadConfig(val.Config)
		if err != nil {
			return errors.Wrapf(err, "failed to read amalgomate configuration for %s", k)
		}

		if verify {
			if _, err := os.Stat(val.OutputDir); os.IsNotExist(err) {
				verifyFailedFn(k, fmt.Sprintf("output directory %s does not exist", val.OutputDir))
				continue
			}

			originalChecksums, err := dirchecksum.ChecksumsForMatchingPaths(val.OutputDir, nil)
			if err != nil {
				return errors.Wrapf(err, "failed to compute original checksums")
			}

			newChecksums, err := dirchecksum.ChecksumsForDirAfterAction(val.OutputDir, func(dir string) error {
				if err := amalgomate.Run(cfg, dir, val.Pkg); err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				return errors.Wrapf(err, "amalgomate verify failed for %s", k)
			}

			if diff := originalChecksums.Diff(newChecksums); len(diff.Diffs) > 0 {
				verifyFailedFn(k, diff.String())
				continue
			}
			continue
		}
		if err := amalgomate.Run(cfg, val.OutputDir, val.Pkg); err != nil {
			return errors.Wrapf(err, "amalgomate failed for %s", k)
		}
	}
	if verify && len(verifyFailedKeys) > 0 {
		fmt.Fprintf(stdout, "amalgomator output differs from what currently exists: %v\n", verifyFailedKeys)
		for _, currKey := range verifyFailedKeys {
			fmt.Fprintf(stdout, "%s%s:\n", strings.Repeat(" ", indentLen), currKey)
			for _, currErrLine := range strings.Split(verifyFailedErrors[currKey], "\n") {
				fmt.Fprintf(stdout, "%s%s\n", strings.Repeat(" ", indentLen*2), currErrLine)
			}
		}
		return fmt.Errorf("")
	}
	return nil
}

func init() {
	pluginapi.AddDebugPFlagPtr(rootCmd.PersistentFlags(), &debugFlag)
	pluginapi.AddProjectDirPFlagPtr(rootCmd.PersistentFlags(), &projectDirFlag)
	if err := rootCmd.MarkPersistentFlagRequired(pluginapi.ProjectDirFlagName); err != nil {
		panic(err)
	}
	pluginapi.AddConfigPFlagPtr(rootCmd.PersistentFlags(), &configFileFlag)
	if err := rootCmd.MarkPersistentFlagRequired(pluginapi.ConfigFlagName); err != nil {
		panic(err)
	}
	rootCmd.PersistentFlags().BoolVar(&verifyFlag, verifyFlagName, false, "verify that current project matches output of amalgomate")
}
