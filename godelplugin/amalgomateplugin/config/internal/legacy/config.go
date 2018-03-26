// Copyright (c) 2016 Palantir Technologies Inc. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.

package legacy

import (
	"github.com/palantir/godel/pkg/versionedconfig"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/palantir/amalgomate/godelplugin/amalgomateplugin/config/internal/v0"
)

type ConfigWithLegacy struct {
	versionedconfig.ConfigWithLegacy `yaml:",inline"`
	Config                           `yaml:",inline"`
}

type Config struct {
	Amalgomators map[string]ProductConfig `yaml:"amalgomators"`
}

type ProductConfig struct {
	Config    string `yaml:"config"`
	OutputDir string `yaml:"output-dir"`
	Pkg       string `yaml:"pkg"`
}

func UpgradeConfig(cfgBytes []byte) ([]byte, error) {
	var legacyCfg ConfigWithLegacy
	if err := yaml.UnmarshalStrict(cfgBytes, &legacyCfg); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal amalgomate-plugin legacy configuration")
	}

	// legacy configuration specified that ordering of YAML map keys was the ordering
	var configMapSlice yaml.MapSlice
	if err := yaml.Unmarshal(cfgBytes, &configMapSlice); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal amalgomate-plugin legacy configuration as yaml.MapSlice")
	}

	var orderedKeys []string
	amalgomators := configMapSlice[1].Value.(yaml.MapSlice)
	for _, mapItem := range amalgomators {
		orderedKeys = append(orderedKeys, mapItem.Key.(string))
	}

	v0Cfg := v0.Config{}
	v0Cfg.OrderedKeys = orderedKeys
	if len(legacyCfg.Amalgomators) > 0 {
		v0Cfg.Amalgomators = make(map[string]v0.ProductConfig)
		for k, v := range legacyCfg.Amalgomators {
			v0Cfg.Amalgomators[k] = v0.ProductConfig{
				Config:    v.Config,
				OutputDir: v.OutputDir,
				Pkg:       v.Pkg,
			}
		}
	}
	upgradedBytes, err := yaml.Marshal(v0Cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal amalgomate-plugin v0 configuration")
	}
	return upgradedBytes, nil
}
