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

package config

import (
	"fmt"
	"sort"

	"github.com/pkg/errors"

	"github.com/palantir/amalgomate/godelplugin/amalgomateplugin"
	"github.com/palantir/amalgomate/godelplugin/config/internal/v0"
)

type Config v0.Config

func (cfg *Config) ToParam() (amalgomateplugin.Param, error) {
	if len(cfg.OrderedKeys) != 0 {
		// verify that ordered keys configuration is valid (return error if not)
		specified := make(map[string]struct{})
		var extra []string

		for _, key := range cfg.OrderedKeys {
			if _, ok := cfg.Amalgomators[key]; !ok {
				// key in OrderedKeys is not valid
				extra = append(extra, key)
				continue
			}
			specified[key] = struct{}{}
		}

		var missing []string
		for k := range cfg.Amalgomators {
			if _, ok := specified[k]; !ok {
				missing = append(missing, k)
			}
		}
		sort.Strings(missing)

		if len(extra) > 0 || len(missing) > 0 {
			msg := "OrderedKeys was specified in configuration but had issues:"
			if len(missing) > 0 {
				msg += fmt.Sprintf(" missing key(s) %v", missing)
				if len(extra) > 0 {
					msg += ","
				}
			}
			if len(extra) > 0 {
				msg += fmt.Sprintf(" invalid key(s) %v", extra)
			}
			return amalgomateplugin.Param{}, errors.Errorf(msg)
		}
	}

	if len(cfg.Amalgomators) == 0 {
		return amalgomateplugin.Param{}, nil
	}
	p := amalgomateplugin.Param{
		OrderedKeys:  cfg.OrderedKeys,
		Amalgomators: make(map[string]amalgomateplugin.ProductParam),
	}
	for k, v := range cfg.Amalgomators {
		v := ProductConfig(v)
		p.Amalgomators[k] = v.ToParam()
	}
	return p, nil
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
