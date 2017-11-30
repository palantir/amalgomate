// Copyright (c) 2016 Palantir Technologies Inc. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Copyright 2016 Palantir Technologies, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/palantir/godel/framework/pluginapi"
	"github.com/palantir/godel/framework/verifyorder"
	"github.com/palantir/pkg/cobracli"
)

var pluginInfo = pluginapi.MustNewInfo(
	"com.palantir",
	"amalgomate-plugin",
	cobracli.Version,
	"amalgomate.yml",
	pluginapi.MustNewTaskInfo(
		"amalgomate",
		"Amalgomate programs",
		pluginapi.TaskInfoGlobalFlagOptions(pluginapi.NewGlobalFlagOptions(
			pluginapi.GlobalFlagOptionsParamDebugFlag("--"+pluginapi.DebugFlagName),
			pluginapi.GlobalFlagOptionsParamProjectDirFlag("--"+pluginapi.ProjectDirFlagName),
			pluginapi.GlobalFlagOptionsParamConfigFlag("--"+pluginapi.ConfigFlagName),
		)),
		pluginapi.TaskInfoVerifyOptions(pluginapi.NewVerifyOptions(
			// by default, run after "generate" but before next built-in task
			pluginapi.VerifyOptionsOrdering(intVar(verifyorder.Generate+50)),
			pluginapi.VerifyOptionsApplyFalseArgs("--"+verifyFlagName),
		)),
	),
)

func intVar(val int) *int {
	return &val
}
