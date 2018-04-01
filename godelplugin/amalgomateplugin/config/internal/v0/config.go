// Copyright (c) 2016 Palantir Technologies Inc. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.

package v0

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Amalgomators map[string]ProductConfig `yaml:"amalgomators,omitempty"`
}

type ProductConfig struct {
	Order     int    `yaml:"order,omitempty"`
	Config    string `yaml:"config,omitempty"`
	OutputDir string `yaml:"output-dir,omitempty"`
	Pkg       string `yaml:"pkg,omitempty"`
}

func UpgradeConfig(cfgBytes []byte) ([]byte, error) {
	var cfg Config
	if err := yaml.UnmarshalStrict(cfgBytes, &cfg); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal amalgomate-plugin v0 configuration")
	}
	return cfgBytes, nil
}
