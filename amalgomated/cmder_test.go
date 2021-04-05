// Copyright (c) 2016 Palantir Technologies Inc. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.

package amalgomated_test

import (
	"fmt"
	"os/exec"
	"testing"

	"github.com/palantir/amalgomate/amalgomated"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testCmder exec.Cmd

func (c *testCmder) Cmd(args []string, cmdWd string) *exec.Cmd {
	c.Args = args
	c.Dir = cmdWd
	return (*exec.Cmd)(c)
}

func (c *testCmder) Run() ([]byte, error) {
	if len(c.Args) > 0 && c.Args[0] == "error" {
		return nil, fmt.Errorf("testCmder returned error based on provided arguments: %v", c.Args)
	}
	return []byte(fmt.Sprintf("Cmd: %v, args: %v, cmdWd: %v", c.Path, c.Args, c.Dir)), nil
}

func TestRunnerWithPrependedArgs(t *testing.T) {
	prepended := []string{"prepended 1", "prepended 2"}
	c := amalgomated.CmderWithPrependedArgs((*testCmder)(exec.Command("/bin/test")), prepended...)

	provided := []string{"provided 1", "provided 2"}
	wd := "wd"

	output, err := (*testCmder)(c.Cmd(provided, wd)).Run()
	require.NoError(t, err)

	expected := "Cmd: /bin/test, args: [prepended 1 prepended 2 provided 1 provided 2], cmdWd: wd"
	assert.Equal(t, expected, string(output))
}
