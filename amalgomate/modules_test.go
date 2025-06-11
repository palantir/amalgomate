// Copyright (c) 2021 Palantir Technologies Inc. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.

package amalgomate

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/nmiyake/pkg/gofiles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_rewriteImports(t *testing.T) {
	origVal := os.Getenv("GOFLAGS")
	_ = os.Setenv("GOFLAGS", "")
	defer func() {
		_ = os.Setenv("GOFLAGS", origVal)
	}()

	for _, tc := range []struct {
		Name      string
		GoFiles   []gofiles.GoFileSpec
		WantFiles map[string]string
	}{
		{
			Name: "rewrites imports within the module",
			GoFiles: []gofiles.GoFileSpec{
				// primary module
				{
					RelPath: "go.mod",
					Src: `module github.com/test-project

require github.com/repackaged-module v1.0.0

replace github.com/repackaged-module => ./repackaged-module-src
`,
				},
				{
					RelPath: "tools.go",
					Src: `// +build tools
package main

import _ "github.com/repackaged-module"
`,
				},
				{
					RelPath: "internal/github.com/repackaged-module/main.go",
					Src: `package main

import _ "github.com/repackaged-module/foo"

func main() {}
`,
				},
				{
					RelPath: "internal/github.com/repackaged-module/foo/foo.go",
					Src:     "package foo",
				},
				// repackaged module
				{
					RelPath: `repackaged-module-src/go.mod`,
					Src:     `module github.com/repackaged-module`,
				},
				{
					RelPath: `repackaged-module-src/main.go`,
					Src: `package main

import _ "github.com/repackaged-module/foo"

func main() {}
`,
				},
				{
					RelPath: `repackaged-module-src/foo/foo.go`,
					Src:     `package foo`,
				},
			},
			WantFiles: map[string]string{
				"internal/github.com/repackaged-module/foo/foo.go": "package foo",
				"internal/github.com/repackaged-module/main.go": `package amalgomated

import _ "github.com/test-project/internal/github.com/repackaged-module/foo"

func AmalgomatedMain()	{}
`,
			},
		},
		{
			Name: "does not rewrite imports to other modules, even if path is within other module",
			GoFiles: []gofiles.GoFileSpec{
				// primary module
				{
					RelPath: "go.mod",
					Src: `module github.com/test-project

require github.com/repackaged-module v1.0.0
require github.com/repackaged-module/nested-module v1.0.0

replace github.com/repackaged-module => ./repackaged-module-src
replace github.com/repackaged-module/nested-module => ./repackaged-module-src/nested-module
`,
				},
				{
					RelPath: "tools.go",
					Src: `// +build tools
package main

import _ "github.com/repackaged-module"
`,
				},
				{
					RelPath: "internal/github.com/repackaged-module/main.go",
					Src: `package main

import _ "github.com/repackaged-module/foo"
import _ "github.com/repackaged-module/nested-module"

func main() {}
`,
				},
				{
					RelPath: "internal/github.com/repackaged-module/foo/foo.go",
					Src:     "package foo",
				},
				// repackaged module
				{
					RelPath: `repackaged-module-src/go.mod`,
					Src:     `module github.com/repackaged-module`,
				},
				{
					RelPath: `repackaged-module-src/main.go`,
					Src: `package main

import _ "github.com/repackaged-module/foo"

func main() {}
`,
				},
				{
					RelPath: `repackaged-module-src/foo/foo.go`,
					Src:     `package foo`,
				},
				{
					RelPath: `repackaged-module-src/nested-module/go.mod`,
					Src:     `module github.com/repackaged-module/nested-module`,
				},
				{
					RelPath: `repackaged-module-src/nested-module/nested.go`,
					Src:     `package nested`,
				},
			},
			WantFiles: map[string]string{
				"internal/github.com/repackaged-module/foo/foo.go": "package foo",
				"internal/github.com/repackaged-module/main.go": `package amalgomated

import _ "github.com/test-project/internal/github.com/repackaged-module/foo"
import _ "github.com/repackaged-module/nested-module"

func AmalgomatedMain()	{}
`,
			},
		},
		{
			Name: "rewrites flag imports",
			GoFiles: []gofiles.GoFileSpec{
				// primary module
				{
					RelPath: "go.mod",
					Src: `module github.com/test-project

require github.com/repackaged-module v1.0.0

replace github.com/repackaged-module => ./repackaged-module-src
`,
				},
				{
					RelPath: "tools.go",
					Src: `// +build tools
package main

import _ "github.com/repackaged-module"
`,
				},
				{
					RelPath: "internal/github.com/repackaged-module/main.go",
					Src: `package main

import _ "flag"

func main() {}
`,
				},
				// repackaged module
				{
					RelPath: `repackaged-module-src/go.mod`,
					Src:     `module github.com/repackaged-module`,
				},
				{
					RelPath: `repackaged-module-src/main.go`,
					Src: `package main

import _ "flag"

func main() {}
`,
				},
			},
			WantFiles: map[string]string{
				"internal/github.com/repackaged-module/main.go": `package amalgomated

import _ "github.com/test-project/internal/amalgomated_flag"

func AmalgomatedMain()	{}
`,
			},
		},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			tmpDir := t.TempDir()
			_, err := gofiles.Write(tmpDir, tc.GoFiles)
			require.NoError(t, err)

			err = rewriteImports(
				filepath.Join(tmpDir, "internal"),
				"github.com/repackaged-module",
				"github.com/test-project/internal",
				nil,
			)
			require.NoError(t, err)

			var sortedKeys []string
			for k := range tc.WantFiles {
				sortedKeys = append(sortedKeys, k)
			}
			sort.Strings(sortedKeys)

			for _, k := range sortedKeys {
				gotContent, err := os.ReadFile(filepath.Join(tmpDir, k))
				require.NoError(t, err, "Failed to read file %s", k)
				assert.Equal(t, tc.WantFiles[k], string(gotContent), "Unexpected file content for file %s", k)
			}
		})
	}
}

// Test_moduleInfoForPackage verifies the functionality that returns the module information for a specified package
// import path resolved from a particular directory. Verifies behavior of cases where module directory is in the module
// cache, resolved locally using a replace directive, and vendored in the project, as well as cases where the module
// import path is not equivalent to the path on disk due to major versions and cases where there are nested modules.
// Also verifies that specifying a package that is not in the top level of the module still returns the proper module
// directory.
func Test_moduleInfoForPackage(t *testing.T) {
	origVal := os.Getenv("GOFLAGS")
	_ = os.Setenv("GOFLAGS", "")
	defer func() {
		_ = os.Setenv("GOFLAGS", origVal)
	}()

	for _, tc := range []struct {
		name            string
		inFiles         []gofiles.GoFileSpec
		pkgName         string
		wantGoModInfoFn func(projectDir string) (*GoModInfo, error)
	}{
		{
			name: "regular module resolves to go module directory",
			inFiles: []gofiles.GoFileSpec{
				// primary project
				{
					RelPath: "go.mod",
					Src: `module github.com/project

go 1.16

require github.com/nmiyake/minimal-module v1.0.0
`,
				},
				{
					RelPath: "go.sum",
					Src: `github.com/nmiyake/minimal-module v1.0.0 h1:Yrx9Iw7/TfaoNuzrDwgPQjEFHKnSCb52Fa696ZhJtgg=
github.com/nmiyake/minimal-module v1.0.0/go.mod h1:efWYh7hk5Cuvur2RY7ykPwDsyNbBzbXZsdWaZTmHxJ8=
`,
				},
				{
					RelPath: "main.go",
					Src: `package main

func main() {}
`,
				},
				{
					RelPath: "tools.go",
					Src: `// +build tools

package main

import _ "github.com/nmiyake/minimal-module"
`,
				},
			},
			pkgName: "github.com/nmiyake/minimal-module",
			wantGoModInfoFn: func(projectDir string) (*GoModInfo, error) {
				goModCacheLocationCmd := exec.Command("go", "env", "GOMODCACHE")
				output, err := goModCacheLocationCmd.CombinedOutput()
				if err != nil {
					return nil, err
				}
				return &GoModInfo{
					Path: "github.com/nmiyake/minimal-module",
					Dir:  filepath.Join(strings.TrimSpace(string(output)), "github.com", "nmiyake", "minimal-module@v1.0.0"),
				}, nil
			},
		},
		{
			name: "regular module with major version v2 resolves to go module directory",
			inFiles: []gofiles.GoFileSpec{
				// primary project
				{
					RelPath: "go.mod",
					Src: `module github.com/project

go 1.16

require github.com/nmiyake/minimal-module/v2 v2.0.1
`,
				},
				{
					RelPath: "go.sum",
					Src: `github.com/nmiyake/minimal-module/v2 v2.0.1 h1:tUx65Ixr3ZhDQPORfwOYCZAO/nzJrj440mZQwOXagZE=
github.com/nmiyake/minimal-module/v2 v2.0.1/go.mod h1:pGJ090vgk9MaRJtLKePp86CP0l1i+2veSgiY9/WWZn8=
`,
				},
				{
					RelPath: "main.go",
					Src: `package main

func main() {}
`,
				},
				{
					RelPath: "tools.go",
					Src: `// +build tools

package main

import _ "github.com/nmiyake/minimal-module/v2"
`,
				},
			},
			pkgName: "github.com/nmiyake/minimal-module/v2",
			wantGoModInfoFn: func(projectDir string) (*GoModInfo, error) {
				goModCacheLocationCmd := exec.Command("go", "env", "GOMODCACHE")
				output, err := goModCacheLocationCmd.CombinedOutput()
				if err != nil {
					return nil, err
				}
				return &GoModInfo{
					Path: "github.com/nmiyake/minimal-module/v2",
					Dir:  filepath.Join(strings.TrimSpace(string(output)), "github.com", "nmiyake", "minimal-module", "v2@v2.0.1"),
				}, nil
			},
		},
		{
			name: "regular module that is a nested module resolves to go module directory",
			inFiles: []gofiles.GoFileSpec{
				// primary project
				{
					RelPath: "go.mod",
					Src: `module github.com/project

go 1.16

require github.com/nmiyake/minimal-module/nested-module v1.0.0
`,
				},
				{
					RelPath: "go.sum",
					Src: `github.com/nmiyake/minimal-module/nested-module v1.0.0 h1:Q6dP2CQkcu7ONt36JGfGI/4tbTtSnOhHfTosmAmdKjE=
github.com/nmiyake/minimal-module/nested-module v1.0.0/go.mod h1:xCJMP9eJIyOyFRNRjOCBOGXCfGwQdsFDL0q84SeMcQo=
`,
				},
				{
					RelPath: "main.go",
					Src: `package main

func main() {}
`,
				},
				{
					RelPath: "tools.go",
					Src: `// +build tools

package main

import _ "github.com/nmiyake/minimal-module/nested-module"
`,
				},
			},
			pkgName: "github.com/nmiyake/minimal-module/nested-module",
			wantGoModInfoFn: func(projectDir string) (*GoModInfo, error) {
				goModCacheLocationCmd := exec.Command("go", "env", "GOMODCACHE")
				output, err := goModCacheLocationCmd.CombinedOutput()
				if err != nil {
					return nil, err
				}
				return &GoModInfo{
					Path: "github.com/nmiyake/minimal-module/nested-module",
					Dir:  filepath.Join(strings.TrimSpace(string(output)), "github.com", "nmiyake", "minimal-module", "nested-module@v1.0.0"),
				}, nil
			},
		},
		{
			name: "module imported using replace with top-level main is properly resolved",
			inFiles: []gofiles.GoFileSpec{
				// module being imported
				{
					RelPath: "helloworld/go.mod",
					Src:     `module github.com/helloworld`,
				},
				{
					RelPath: "helloworld/main.go",
					Src: `package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}
`,
				},
				// primary project
				{
					RelPath: "go.mod",
					Src: `module github.com/project

go 1.16

require github.com/helloworld v1.0.0

replace github.com/helloworld => ./helloworld
`,
				},
				{
					RelPath: "main.go",
					Src: `package main

func main() {}
`,
				},
				{
					RelPath: "tools.go",
					Src: `// +build tools

package main

import _ "github.com/helloworld"
`,
				},
			},
			pkgName: "github.com/helloworld",
			wantGoModInfoFn: func(projectDir string) (*GoModInfo, error) {
				return &GoModInfo{
					Path: "github.com/helloworld",
					Dir:  filepath.Join(projectDir, "helloworld"),
				}, nil
			},
		},
		{
			name: "module with major version v2 imported using replace with top-level main is properly resolved",
			inFiles: []gofiles.GoFileSpec{
				// module being imported
				{
					RelPath: "helloworld/go.mod",
					Src:     `module github.com/helloworld/v2`,
				},
				{
					RelPath: "helloworld/main.go",
					Src: `package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}
`,
				},
				// primary project
				{
					RelPath: "go.mod",
					Src: `module github.com/project

go 1.16

require github.com/helloworld/v2 v2.0.0

replace github.com/helloworld/v2 => ./helloworld
`,
				},
				{
					RelPath: "main.go",
					Src: `package main

func main() {}
`,
				},
				{
					RelPath: "tools.go",
					Src: `// +build tools

package main

import _ "github.com/helloworld/v2"
`,
				},
			},
			pkgName: "github.com/helloworld/v2",
			wantGoModInfoFn: func(projectDir string) (*GoModInfo, error) {
				return &GoModInfo{
					Path: "github.com/helloworld/v2",
					Dir:  filepath.Join(projectDir, "helloworld"),
				}, nil
			},
		},
		{
			name: "module imported using replace with nested main is properly resolved",
			inFiles: []gofiles.GoFileSpec{
				// module being imported
				{
					RelPath: "helloworld/go.mod",
					Src:     `module github.com/helloworld`,
				},
				{
					RelPath: "helloworld/hello/main.go",
					Src: `package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}
`,
				},
				// primary project
				{
					RelPath: "go.mod",
					Src: `module github.com/project

go 1.16

require github.com/helloworld v1.0.0

replace github.com/helloworld => ./helloworld
`,
				},
				{
					RelPath: "main.go",
					Src: `package main

func main() {}
`,
				},
				{
					RelPath: "tools.go",
					Src: `// +build tools

package main

import _ "github.com/helloworld/hello"
`,
				},
			},
			pkgName: "github.com/helloworld/hello",
			wantGoModInfoFn: func(projectDir string) (*GoModInfo, error) {
				return &GoModInfo{
					Path: "github.com/helloworld",
					Dir:  filepath.Join(projectDir, "helloworld"),
				}, nil
			},
		},
		{
			name: "vendored module properly resolves to version in vendor directory",
			inFiles: []gofiles.GoFileSpec{
				// primary project
				{
					RelPath: "go.mod",
					Src: `module github.com/project

go 1.16

require github.com/helloworld v1.0.0

replace github.com/helloworld => ./helloworld
`,
				},
				{
					RelPath: "main.go",
					Src: `package main

func main() {}
`,
				},
				{
					RelPath: "tools.go",
					Src: `// +build tools

package main

import _ "github.com/helloworld"
`,
				},
				{
					RelPath: "vendor/modules.txt",
					Src: `# github.com/helloworld v1.0.0 => ./helloworld
## explicit
github.com/helloworld
# github.com/helloworld => ./helloworld
	`,
				},
				{
					RelPath: "vendor/github.com/helloworld/go.mod",
					Src:     `module github.com/helloworld`,
				},
				{
					RelPath: "vendor/github.com/helloworld/main.go",
					Src: `package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}
`,
				},
			},
			pkgName: "github.com/helloworld",
			wantGoModInfoFn: func(projectDir string) (*GoModInfo, error) {
				return &GoModInfo{
					Path: "github.com/helloworld",
					Dir:  filepath.Join(projectDir, "vendor", "github.com", "helloworld"),
				}, nil
			},
		},
		{
			name: "vendored module with major version v2 properly resolves to version in vendor directory",
			inFiles: []gofiles.GoFileSpec{
				// primary project
				{
					RelPath: "go.mod",
					Src: `module github.com/project

go 1.16

require github.com/helloworld/v2 v2.0.0

replace github.com/helloworld/v2 => ./helloworld
`,
				},
				{
					RelPath: "main.go",
					Src: `package main

func main() {}
`,
				},
				{
					RelPath: "tools.go",
					Src: `// +build tools

package main

import _ "github.com/helloworld/v2"
`,
				},
				{
					RelPath: "vendor/modules.txt",
					Src: `# github.com/helloworld/v2 v2.0.0 => ./helloworld
## explicit
github.com/helloworld/v2
# github.com/helloworld/v2 => ./helloworld
	`,
				},
				{
					RelPath: "vendor/github.com/helloworld/v2/go.mod",
					Src:     `module github.com/helloworld/v2`,
				},
				{
					RelPath: "vendor/github.com/helloworld/v2/main.go",
					Src: `package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}
`,
				},
			},
			pkgName: "github.com/helloworld/v2",
			wantGoModInfoFn: func(projectDir string) (*GoModInfo, error) {
				return &GoModInfo{
					Path: "github.com/helloworld/v2",
					Dir:  filepath.Join(projectDir, "vendor", "github.com", "helloworld", "v2"),
				}, nil
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			projectDir := t.TempDir()
			_, err := gofiles.Write(projectDir, tc.inFiles)
			require.NoError(t, err)

			gotModuleInfo, err := moduleInfoForPackage(tc.pkgName, projectDir)
			require.NoError(t, err)

			gotModuleDirNormalized, err := filepath.EvalSymlinks(gotModuleInfo.Dir)
			require.NoError(t, err)
			gotModuleInfo.Dir = gotModuleDirNormalized

			require.NotNil(t, tc.wantGoModInfoFn, "tc.wantGoModInfoFn must not be nil: test case needs to specify this function")
			wantModuleInfo, err := tc.wantGoModInfoFn(projectDir)
			require.NoError(t, err)
			wantModuleDirNormalized, err := filepath.EvalSymlinks(wantModuleInfo.Dir)
			require.NoError(t, err)
			wantModuleInfo.Dir = wantModuleDirNormalized

			assert.Equal(t, wantModuleInfo, gotModuleInfo)
		})
	}
}

// Test_copyModuleRecursively verifies that the copyModuleRecursively recursively copies a source module directory into
// a destination directory.
func Test_copyModuleRecursively(t *testing.T) {
	for _, tc := range []struct {
		Name       string
		ModuleName string
		SrcFiles   []gofiles.GoFileSpec
		WantFiles  []string
	}{
		{
			Name:       "Copies basic module",
			ModuleName: "github.com/test",
			SrcFiles: []gofiles.GoFileSpec{
				{
					RelPath: "go.mod",
					Src:     "module github.com/test",
				},
				{
					RelPath: "foo/foo.go",
					Src:     "package foo",
				},
			},
			WantFiles: []string{
				"github.com",
				"github.com/test",
				"github.com/test/foo",
				"github.com/test/foo/foo.go",
			},
		},
		{
			Name:       "Does not copy nested module",
			ModuleName: "github.com/test",
			SrcFiles: []gofiles.GoFileSpec{
				{
					RelPath: "go.mod",
					Src:     "module github.com/test",
				},
				{
					RelPath: "main.go",
					Src:     "package main",
				},
				{
					RelPath: "nested-module/go.mod",
					Src:     "module github.com/test/nested-module",
				},
				{
					RelPath: "nested-module/main.go",
					Src:     "package main",
				},
			},
			WantFiles: []string{
				"github.com",
				"github.com/test",
				"github.com/test/main.go",
			},
		},
		{
			Name:       "Does not copy vendor directory",
			ModuleName: "github.com/test",
			SrcFiles: []gofiles.GoFileSpec{
				{
					RelPath: "go.mod",
					Src:     "module github.com/test",
				},
				{
					RelPath: "main.go",
					Src:     "package main",
				},
				{
					RelPath: "vendor/github.com/foo/foo.go",
					Src:     "package foo",
				},
			},
			WantFiles: []string{
				"github.com",
				"github.com/test",
				"github.com/test/main.go",
			},
		},
		{
			Name:       "Includes module version suffix in copy",
			ModuleName: "github.com/test/v2",
			SrcFiles: []gofiles.GoFileSpec{
				{
					RelPath: "go.mod",
					Src:     "module github.com/test/v2",
				},
				{
					RelPath: "main.go",
					Src:     "package main",
				},
			},
			WantFiles: []string{
				"github.com",
				"github.com/test",
				"github.com/test/v2",
				"github.com/test/v2/main.go",
			},
		},
		{
			Name:       "Single-segment module name works",
			ModuleName: "test-module-name",
			SrcFiles: []gofiles.GoFileSpec{
				{
					RelPath: "go.mod",
					Src:     "module test-module-name",
				},
				{
					RelPath: "main.go",
					Src:     "package main",
				},
			},
			WantFiles: []string{
				"test-module-name",
				"test-module-name/main.go",
			},
		},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			tmpDir := t.TempDir()

			dstDir := filepath.Join(tmpDir, "dstDir")
			err := os.Mkdir(dstDir, 0755)
			require.NoError(t, err)

			srcDir := filepath.Join(tmpDir, "srcDir")
			_, err = gofiles.Write(srcDir, tc.SrcFiles)
			require.NoError(t, err)

			err = copyModuleRecursively(tc.ModuleName, srcDir, dstDir)
			require.NoError(t, err)

			gotFilePaths, err := allFilePaths(dstDir)
			require.NoError(t, err)

			assert.Equal(t, tc.WantFiles, gotFilePaths)
		})
	}
}

func allFilePaths(dir string) ([]string, error) {
	var paths []string
	if err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// skip root entry, but process contents
		if path == dir {
			return nil
		}
		gotRelPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		paths = append(paths, gotRelPath)
		return nil
	}); err != nil {
		return nil, err
	}
	return paths, nil
}
