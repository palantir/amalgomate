// Copyright (c) 2016 Palantir Technologies Inc. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.

package amalgomate

import (
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Pkgs map[string]SrcPkg `yaml:"packages"`
}

type SrcPkg struct {
	MainPkg string `yaml:"main"`
}

func LoadConfig(configPath string) (Config, error) {
	file, err := ioutil.ReadFile(configPath)
	if err != nil {
		return Config{}, errors.Wrapf(err, "failed to read file %s", configPath)
	}

	var cfg Config
	if err := yaml.Unmarshal(file, &cfg); err != nil {
		return Config{}, errors.Wrapf(err, "failed to unmarshal file %s", configPath)
	}

	if len(cfg.Pkgs) == 0 {
		return Config{}, errors.Errorf("configuration read from file %s with content %q was empty", configPath, string(file))
	}

	for name, pkg := range cfg.Pkgs {
		if name == "" {
			return Config{}, errors.Errorf("config cannot contain a blank name: %v", cfg)
		}

		if pkg.MainPkg == "" {
			return Config{}, errors.Errorf("config for package %s had a blank main package directory: %v", name, cfg)
		}
	}
	return cfg, nil
}
