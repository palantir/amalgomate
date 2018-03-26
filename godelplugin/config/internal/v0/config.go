// Copyright (c) 2016 Palantir Technologies Inc. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Copyright 2016 Palantir Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v0

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Config struct {
	// OrderedKeys stores the order in which the amalgomators should be run. If unspecified, they are run in
	// alphabetical order based on the name of the key. If specified, every element must be present.
	OrderedKeys []string `yaml:"ordered-keys"`

	Amalgomators map[string]ProductConfig `yaml:"amalgomators"`
}

type ProductConfig struct {
	Config    string `yaml:"config"`
	OutputDir string `yaml:"output-dir"`
	Pkg       string `yaml:"pkg"`
}

func UpgradeConfig(cfgBytes []byte) ([]byte, error) {
	var cfg Config
	if err := yaml.UnmarshalStrict(cfgBytes, &cfg); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal generate-plugin v0 configuration")
	}
	return cfgBytes, nil
}
