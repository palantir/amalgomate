// Copyright (c) 2016 Palantir Technologies Inc. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.

package integration_test

import (
	"testing"

	"github.com/palantir/godel/framework/pluginapitester"
	"github.com/palantir/godel/pkg/products"
	"github.com/stretchr/testify/require"
)

const (
	godelYML = `exclude:
  names:
    - "\\..+"
    - "vendor"
  paths:
    - "godel"
`
)

func TestUpgradeConfig(t *testing.T) {
	pluginPath, err := products.Bin("amalgomate-plugin")
	require.NoError(t, err)
	pluginProvider := pluginapitester.NewPluginProvider(pluginPath)

	pluginapitester.RunUpgradeConfigTest(t,
		pluginProvider,
		nil,
		[]pluginapitester.UpgradeConfigTestCase{
			{
				Name: "legacy amalgomate config is upgraded",
				ConfigFiles: map[string]string{
					"godel/config/godel.yml": godelYML,
					"godel/config/amalgomate-plugin.yml": `legacy-config: true
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
`,
				},
				WantOutput: "Upgraded configuration for amalgomate-plugin.yml\n",
				WantFiles: map[string]string{
					"godel/config/amalgomate-plugin.yml": `ordered-keys:
- test-product
- next-product
- other-product
amalgomators:
  next-product:
    config: next.yml
    output-dir: next-output
    pkg: next-pkg
  other-product:
    config: other.yml
    output-dir: other-output
    pkg: other-pkg
  test-product:
    config: test.yml
    output-dir: test-output
    pkg: test-pkg
`,
				},
			},
			{
				Name: "current config is unmodified",
				ConfigFiles: map[string]string{
					"godel/config/godel.yml": godelYML,
					"godel/config/amalgomate-plugin.yml": `
ordered-keys:
  - test-product
  - next-product
  - other-product
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
`,
				},
				WantOutput: "",
				WantFiles: map[string]string{
					"godel/config/amalgomate-plugin.yml": `
ordered-keys:
  - test-product
  - next-product
  - other-product
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
`,
				},
			},
		},
	)
}
