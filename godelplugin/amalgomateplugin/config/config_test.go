// Copyright (c) 2016 Palantir Technologies Inc. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.

package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/palantir/amalgomate/godelplugin/amalgomateplugin"
	"github.com/palantir/amalgomate/godelplugin/amalgomateplugin/config"
)

func TestReadConfig(t *testing.T) {
	content := `
amalgomators:
  test-product:
    config: test.yml
    output-dir: test-output
    pkg: test-pkg
  next-product:
    config: next.yml
    output-dir: next-output
    pkg: next-pkg
  other-product:
    config: other.yml
    output-dir: other-output
    pkg: other-pkg
`
	var got config.Config
	err := yaml.Unmarshal([]byte(content), &got)
	require.NoError(t, err)

	wantCfg := config.Config{
		Amalgomators: config.ToAmalgomators(map[string]config.ProductConfig{
			"test-product": {
				Config:    "test.yml",
				OutputDir: "test-output",
				Pkg:       "test-pkg",
			},
			"next-product": {
				Config:    "next.yml",
				OutputDir: "next-output",
				Pkg:       "next-pkg",
			},
			"other-product": {
				Config:    "other.yml",
				OutputDir: "other-output",
				Pkg:       "other-pkg",
			},
		}),
	}
	assert.Equal(t, wantCfg, got)
}

func TestToParam(t *testing.T) {
	cfg := config.Config{
		Amalgomators: config.ToAmalgomators(map[string]config.ProductConfig{
			"test-product": {
				Config:    "test.yml",
				OutputDir: "test-output",
				Pkg:       "test-pkg",
			},
			"next-product": {
				Config:    "next.yml",
				OutputDir: "next-output",
				Pkg:       "next-pkg",
			},
			"other-product": {
				Config:    "other.yml",
				OutputDir: "other-output",
				Pkg:       "other-pkg",
			},
		}),
	}

	wantParam := amalgomateplugin.Param{
		Amalgomators: map[string]amalgomateplugin.ProductParam{
			"test-product": {
				Config:    "test.yml",
				OutputDir: "test-output",
				Pkg:       "test-pkg",
			},
			"next-product": {
				Config:    "next.yml",
				OutputDir: "next-output",
				Pkg:       "next-pkg",
			},
			"other-product": {
				Config:    "other.yml",
				OutputDir: "other-output",
				Pkg:       "other-pkg",
			},
		},
	}
	param, err := cfg.ToParam()
	require.NoError(t, err)
	assert.Equal(t, wantParam, param)
}
