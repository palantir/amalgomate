// Copyright (c) 2016 Palantir Technologies Inc. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Copyright 2016 Palantir Technologies, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Param struct {
	OrderedKeys  []string
	Amalgomators map[string]ProductConfig
}

type Config struct {
	Amalgomators amalgomators `yaml:"amalgomators"`
}

func (c *Config) ToParam() Param {
	if len(c.Amalgomators) == 0 {
		return Param{}
	}
	p := Param{
		Amalgomators: make(map[string]ProductConfig),
	}
	for _, item := range c.Amalgomators {
		key := item.Key.(string)
		p.OrderedKeys = append(p.OrderedKeys, key)
		p.Amalgomators[key] = item.Value.(ProductConfig)
	}
	return p
}

type amalgomators yaml.MapSlice

func (a *amalgomators) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var mapSlice yaml.MapSlice
	if err := unmarshal(&mapSlice); err != nil {
		return err
	}
	// values of MapSlice are known to be ProductConfig, so read them out as such
	for i, v := range mapSlice {
		bytes, err := yaml.Marshal(v.Value)
		if err != nil {
			return err
		}
		var currCfg ProductConfig
		if err := yaml.Unmarshal(bytes, &currCfg); err != nil {
			return err
		}
		mapSlice[i].Value = currCfg
	}
	*a = amalgomators(mapSlice)
	return nil
}

type ProductConfig struct {
	Config    string `yaml:"config"`
	OutputDir string `yaml:"output-dir"`
	Pkg       string `yaml:"pkg"`
}

func readConfig(cfgFile string) (Config, error) {
	bytes, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		return Config{}, errors.Wrapf(err, "failed to read file")
	}
	var cfg Config
	if err := yaml.Unmarshal(bytes, &cfg); err != nil {
		return Config{}, errors.Wrapf(err, "failed to unmarshal")
	}
	return cfg, nil
}
