// Copyright 2016 Palantir Technologies, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"

	"github.com/palantir/amalgomate/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
