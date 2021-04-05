// Copyright (c) 2016 Palantir Technologies Inc. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.

package amalgomated_test

import (
	"bytes"
	"flag"
	"fmt"
	"testing"

	"github.com/palantir/amalgomate/amalgomated"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunApp(t *testing.T) {
	for i, currCase := range []struct {
		args             []string
		fset             func() *flag.FlagSet
		expectedExitCode int
		expectedOutput   string
	}{
		// running non-proxy command executes app function
		{
			args: []string{
				"arg0",
				"arg1",
			},
			expectedExitCode: 13,
			expectedOutput:   "[arg0 arg1]\n",
		},
		// flags are passed to non-proxy command
		{
			args: []string{
				"arg0",
				"--global-flag",
				"arg1",
				"--command-flag",
				"flag-value",
			},
			expectedExitCode: 13,
			expectedOutput:   "[arg0 --global-flag arg1 --command-flag flag-value]\n",
		},
		// running proxy command executes proxy
		{
			args: []string{
				"arg0",
				amalgomated.ProxyCmdPrefix + "foo",
			},
			expectedExitCode: 0,
			expectedOutput:   "foo\n",
		},
		// providing flag set ignores flag values
		{
			args: []string{
				"arg0",
				"--string-flag",
				"flag_value",
				amalgomated.ProxyCmdPrefix + "foo",
			},
			fset: func() *flag.FlagSet {
				fset := flag.NewFlagSet("fset", flag.ContinueOnError)
				_ = fset.String("string-flag", "", "")
				return fset
			},
			expectedExitCode: 0,
			expectedOutput:   "foo\n",
		},
	} {
		runAppOutput := &bytes.Buffer{}
		cmdWithRunner, err := amalgomated.NewCmdWithRunner("foo", func() {
			fmt.Fprintln(runAppOutput, "foo")
		})
		require.NoError(t, err, "Case %d", i)

		cmdSet, err := amalgomated.NewStringCmdSetForRunners(cmdWithRunner)
		require.NoError(t, err, "Case %d", i)
		cmdLibrary := amalgomated.NewCmdLibrary(cmdSet)
		appFunc := func(osArgs []string) int {
			fmt.Fprintln(runAppOutput, osArgs)
			return currCase.expectedExitCode
		}

		var fset *flag.FlagSet
		if currCase.fset != nil {
			fset = currCase.fset()
		}
		actualExitCode := amalgomated.RunApp(currCase.args, fset, cmdLibrary, appFunc)
		assert.Equal(t, currCase.expectedOutput, runAppOutput.String(), "Case %d", i)
		assert.Equal(t, currCase.expectedExitCode, actualExitCode, "Case %d", i)
	}
}
