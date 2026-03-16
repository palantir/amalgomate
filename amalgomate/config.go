// Copyright (c) 2016 Palantir Technologies Inc. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.

package amalgomate

import (
	"go/token"
	"maps"
	"os"
	"slices"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Pkgs map[string]SrcPkg `yaml:"packages"`
	// AmalgomateDir specifies the directory in the output path in which amalgomated packages are written.
	// If blank, defaults to "internal". This directory should be considered fully managed by amalgomate, as it is
	// removed before amalgomate is run. If non-empty, must be a valid Go identifier (token.IsIdentifier must return
	// true for the value). This restriction is imposed to ensure that the value can be used as a valid import path and
	// is safe to use as part of a path (it is not an absolute path, does not contain subdirectories or directives like
	// "..", etc.).
	AmalgomateDir string `yaml:"amalgomate-dir"`
	// RepackageOnly specifies whether the amalgomate operation should only repackage target code. If true, does not
	// write the top-level file that provides entrypoints to the amalgomated code.
	RepackageOnly bool `yaml:"repackage-only"`
}

type SrcPkg struct {
	MainPkg                string   `yaml:"main"`
	DoNotRewriteFlagImport []string `yaml:"do-not-rewrite-flag-import"`
	RenameInternal         bool     `yaml:"rename-internal"`
}

func (cfg Config) Validate() error {
	if cfg.AmalgomateDir != "" && !token.IsIdentifier(cfg.AmalgomateDir) {
		return errors.Errorf("AmalgomateDir %s must be a valid Go identifier if it is non-empty", cfg.AmalgomateDir)
	}

	for _, name := range slices.Sorted(maps.Keys(cfg.Pkgs)) {
		if name == "" {
			return errors.Errorf("Pkgs cannot have an entry with an empty key")
		}
		pkg := cfg.Pkgs[name]
		if pkg.MainPkg == "" {
			return errors.Errorf("package %s in Pkgs cannot have an empty main package directory", name)
		}
	}
	return nil
}

func LoadConfig(configPath string) (Config, error) {
	file, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, errors.Wrapf(err, "failed to read file %s", configPath)
	}

	var cfg Config
	if err := yaml.Unmarshal(file, &cfg); err != nil {
		return Config{}, errors.Wrapf(err, "failed to unmarshal file %s", configPath)
	}

	// impose restriction that configuration must specify at least 1 package
	if len(cfg.Pkgs) == 0 {
		return Config{}, errors.Errorf("configuration read from file %s with content %q is empty", configPath, string(file))
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, errors.Wrapf(err, "configuration read from file %s is not valid", configPath)
	}

	return cfg, nil
}
