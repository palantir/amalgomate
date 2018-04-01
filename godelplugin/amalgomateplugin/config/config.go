// Copyright (c) 2016 Palantir Technologies Inc. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.

package config

import (
	"sort"

	"github.com/palantir/amalgomate/godelplugin/amalgomateplugin"
	"github.com/palantir/amalgomate/godelplugin/amalgomateplugin/config/internal/v0"
)

type Config v0.Config

func (cfg *Config) ToParam() amalgomateplugin.Param {
	if len(cfg.Amalgomators) == 0 {
		return amalgomateplugin.Param{}
	}

	var orderedKeys []string
	for k := range cfg.Amalgomators {
		orderedKeys = append(orderedKeys, k)
	}
	sort.Slice(orderedKeys, func(i, j int) bool {
		o1 := cfg.Amalgomators[orderedKeys[i]].Order
		o2 := cfg.Amalgomators[orderedKeys[j]].Order
		if o1 != o2 {
			return o1 < o2
		}
		// if order is tied, fall back on string comparison
		return orderedKeys[i] < orderedKeys[j]
	})

	amalgomators := make(map[string]amalgomateplugin.ProductParam)
	for k, v := range cfg.Amalgomators {
		v := ProductConfig(v)
		amalgomators[k] = v.ToParam()
	}
	return amalgomateplugin.Param{
		OrderedKeys:  orderedKeys,
		Amalgomators: amalgomators,
	}
}

func ToAmalgomators(in map[string]ProductConfig) map[string]v0.ProductConfig {
	if in == nil {
		return nil
	}
	out := make(map[string]v0.ProductConfig, len(in))
	for k, v := range in {
		out[k] = ToProductConfig(v)
	}
	return out
}

type ProductConfig v0.ProductConfig

func ToProductConfig(in ProductConfig) v0.ProductConfig {
	return v0.ProductConfig(in)
}

func (cfg *ProductConfig) ToParam() amalgomateplugin.ProductParam {
	return amalgomateplugin.ProductParam{
		Config:    cfg.Config,
		OutputDir: cfg.OutputDir,
		Pkg:       cfg.Pkg,
	}
}
