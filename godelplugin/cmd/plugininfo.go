// Copyright (c) 2016 Palantir Technologies Inc. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Copyright 2016 Palantir Technologies, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"github.com/palantir/godel/framework/pluginapi/v2/pluginapi"
	"github.com/palantir/godel/framework/verifyorder"
	"github.com/palantir/pkg/cobracli"
)

var PluginInfo = pluginapi.MustNewPluginInfo(
	"com.palantir.amalgomate",
	"amalgomate-plugin",
	cobracli.Version,
	pluginapi.PluginInfoUsesConfigFile(),
	pluginapi.PluginInfoGlobalFlagOptions(
		pluginapi.GlobalFlagOptionsParamDebugFlag("--"+pluginapi.DebugFlagName),
		pluginapi.GlobalFlagOptionsParamProjectDirFlag("--"+pluginapi.ProjectDirFlagName),
		pluginapi.GlobalFlagOptionsParamConfigFlag("--"+pluginapi.ConfigFlagName),
	),
	pluginapi.PluginInfoTaskInfo(
		"amalgomate",
		"Amalgomate programs",
		pluginapi.TaskInfoCommand("run"),
		pluginapi.TaskInfoVerifyOptions(
			// by default, run after "generate" but before next built-in task
			pluginapi.VerifyOptionsOrdering(intVar(verifyorder.Generate+50)),
			pluginapi.VerifyOptionsApplyFalseArgs("--"+VerifyFlagName),
		),
	),
	pluginapi.PluginInfoUpgradeConfigTaskInfo(
		pluginapi.UpgradeConfigTaskInfoCommand("upgrade-config"),
		pluginapi.LegacyConfigFile("amalgomate.yml"),
	),
)

func intVar(val int) *int {
	return &val
}
