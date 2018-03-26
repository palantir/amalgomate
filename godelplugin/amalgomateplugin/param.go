// Copyright (c) 2016 Palantir Technologies Inc. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Copyright 2016 Palantir Technologies, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package amalgomateplugin

type Param struct {
	OrderedKeys  []string
	Amalgomators map[string]ProductParam
}

type ProductParam struct {
	Config    string
	OutputDir string
	Pkg       string
}
