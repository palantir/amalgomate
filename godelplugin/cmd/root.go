// Copyright (c) 2016 Palantir Technologies Inc. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Copyright 2016 Palantir Technologies, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"github.com/palantir/godel/framework/pluginapi"
	"github.com/spf13/cobra"
)

const VerifyFlagName = "verify"

var (
	DebugFlag bool

	projectDirFlag string
	configFileFlag string
)

var RootCmd = &cobra.Command{
	Use:   "amalgomate-plugin",
	Short: "Run amalgomate based on project configuration",
}

func init() {
	pluginapi.AddDebugPFlagPtr(RootCmd.PersistentFlags(), &DebugFlag)
	pluginapi.AddProjectDirPFlagPtr(RootCmd.PersistentFlags(), &projectDirFlag)
	if err := RootCmd.MarkPersistentFlagRequired(pluginapi.ProjectDirFlagName); err != nil {
		panic(err)
	}
	pluginapi.AddConfigPFlagPtr(RootCmd.PersistentFlags(), &configFileFlag)
	if err := RootCmd.MarkPersistentFlagRequired(pluginapi.ConfigFlagName); err != nil {
		panic(err)
	}
}
