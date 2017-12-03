// Copyright (c) 2016 Palantir Technologies Inc. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Copyright 2016 Palantir Technologies, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
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
	var got Config
	err := yaml.Unmarshal([]byte(content), &got)
	require.NoError(t, err)

	wantCfg := Config{
		Amalgomators: []yaml.MapItem{
			{
				Key: "test-product",
				Value: ProductConfig{
					Config:    "test.yml",
					OutputDir: "test-output",
					Pkg:       "test-pkg",
				},
			},
			{
				Key: "next-product",
				Value: ProductConfig{
					Config:    "next.yml",
					OutputDir: "next-output",
					Pkg:       "next-pkg",
				},
			},
			{
				Key: "other-product",
				Value: ProductConfig{
					Config:    "other.yml",
					OutputDir: "other-output",
					Pkg:       "other-pkg",
				},
			},
		},
	}
	assert.Equal(t, wantCfg, got)
}

func TestToParam(t *testing.T) {
	cfg := Config{
		Amalgomators: []yaml.MapItem{
			{
				Key: "test-product",
				Value: ProductConfig{
					Config:    "test.yml",
					OutputDir: "test-output",
					Pkg:       "test-pkg",
				},
			},
			{
				Key: "next-product",
				Value: ProductConfig{
					Config:    "next.yml",
					OutputDir: "next-output",
					Pkg:       "next-pkg",
				},
			},
			{
				Key: "other-product",
				Value: ProductConfig{
					Config:    "other.yml",
					OutputDir: "other-output",
					Pkg:       "other-pkg",
				},
			},
		},
	}

	wantParam := Param{
		OrderedKeys: []string{
			"test-product",
			"next-product",
			"other-product",
		},
		Amalgomators: map[string]ProductConfig{
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
	assert.Equal(t, wantParam, cfg.ToParam())
}
