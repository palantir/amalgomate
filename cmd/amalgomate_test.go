// Copyright (c) 2016 Palantir Technologies Inc. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/nmiyake/pkg/dirs"
	"github.com/nmiyake/pkg/gofiles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/palantir/amalgomate/cmd/amalgomate"
)

const internalDir = "internal"

func TestAmalgomate(t *testing.T) {
	tmpDir, cleanup, err := dirs.TempDir(".", "")
	defer cleanup()
	require.NoError(t, err)

	for i, currCase := range []struct {
		input          string
		outputPkgName  string
		expectedOutput string
	}{
		{
			input: `package main

			import "fmt"

			func main() {
				fmt.Println("testMain 0")
			}
			`,
			outputPkgName: "main",
			expectedOutput: `package amalgomated

			import "fmt"

			func AmalgomatedMain() {
				fmt.Println("testMain 0")
			}
			`,
		},
		{
			input: `
			// comment

			    package    main

			import (
				"fmt"
			)

			func main() {
				helper()
			}

			func helper() {
						fmt.Println("testMain 1")
			   }
			`,
			outputPkgName: "lib",
			expectedOutput: `// comment

			package amalgomated

			import (
				"fmt"
			)

			func AmalgomatedMain() {
				helper()
			}

			func helper() {
				fmt.Println("testMain 1")
			}
			`,
		},
	} {
		currCase.input = unindent(currCase.input)
		currCase.expectedOutput = unindent(currCase.expectedOutput)

		currTestDir := path.Join(tmpDir, fmt.Sprintf("test%d", i))
		err := os.Mkdir(currTestDir, 0755)
		require.NoError(t, err)

		testMainDir := path.Join(currTestDir, "testmain")
		files, err := gofiles.Write(testMainDir, []gofiles.GoFileSpec{
			{
				RelPath: "testmain.go",
				Src:     currCase.input,
			},
		})
		require.NoError(t, err, "Case %d", i)
		importPath := files["testmain.go"].ImportPath

		confPath := path.Join(currTestDir, "test.conf")
		err = ioutil.WriteFile(confPath, []byte(getConfigString(t, amalgomate.Config{
			Pkgs: map[string]amalgomate.SrcPkg{
				"testmain": {
					MainPkg: importPath,
				},
			},
		})), 0644)
		require.NoError(t, err, "Case %d", i)

		testOutputPath := path.Join(currTestDir, "testout")

		outputStr, err := runMain(confPath, testOutputPath, currCase.outputPkgName)
		require.NoError(t, err, "main encountered error for case %d. Output:\n%s", i, outputStr)

		outputTestMainDir := path.Join(testOutputPath, internalDir, importPath)
		writtenFileBytes, err := ioutil.ReadFile(path.Join(outputTestMainDir, "testmain.go"))
		require.NoError(t, err)

		assert.Equal(t, currCase.expectedOutput, string(writtenFileBytes), "Case %d", i)

		if currCase.outputPkgName == "main" {
			goBuild := exec.Command("go", "build")
			goBuild.Dir = testOutputPath
			buildOutput, err := goBuild.CombinedOutput()
			require.NoError(t, err, "Case %d: %v", i, string(buildOutput))

			output, err := exec.Command(path.Join(testOutputPath, "testout"), "testmain").Output()
			require.NoError(t, err, "Case %d", i)
			assert.Equal(t, fmt.Sprintf("testMain %d\n", i), string(output), "Case %d", i)
		} else {
			goBuild := exec.Command("go", "build")
			goBuild.Dir = testOutputPath
			buildOutput, err := goBuild.CombinedOutput()
			require.NoError(t, err, "Case %d: %s", i, string(buildOutput))
		}
	}
}

func TestAmalgomateFlag(t *testing.T) {
	tmpDir, cleanup, err := dirs.TempDir(".", "")
	defer cleanup()
	require.NoError(t, err)

	for i, currCase := range []struct {
		files []gofiles.GoFileSpec
	}{
		{
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo/foo.go",
					Src: `package main

			import (
				"flag"
				"fmt"
			)

			var list = flag.Bool("l", false, "")

			func main() {
				fmt.Println("testMain foo")
			}
			`,
				},
				{
					RelPath: "bar/bar.go",
					Src: `package main

			import (
				"flag"
				"fmt"
			)

			var list = flag.Bool("l", false, "")

			func main() {
				fmt.Println("testMain bar")
			}
			`,
				},
			},
		},
	} {
		currTestDir := path.Join(tmpDir, fmt.Sprintf("test%d", i))
		err := os.Mkdir(currTestDir, 0755)
		require.NoError(t, err)

		for i := range currCase.files {
			currCase.files[i].Src = unindent(currCase.files[i].Src)
		}

		files, err := gofiles.Write(currTestDir, currCase.files)
		require.NoError(t, err, "Case %d", i)

		confPath := path.Join(currTestDir, "test.conf")
		err = ioutil.WriteFile(confPath, []byte(getConfigString(t, amalgomate.Config{
			Pkgs: map[string]amalgomate.SrcPkg{
				"foo": {
					MainPkg: files["foo/foo.go"].ImportPath,
				},
				"bar": {
					MainPkg: files["bar/bar.go"].ImportPath,
				},
			},
		})), 0644)
		require.NoError(t, err, "Case %d", i)

		testOutputPath := path.Join(currTestDir, "testout")

		outputStr, err := runMain(confPath, testOutputPath, "main")
		require.NoError(t, err, "Case %d\nOutput:\n%s", i, outputStr)

		goBuild := exec.Command("go", "build")
		goBuild.Dir = testOutputPath
		_, err = goBuild.Output()
		require.NoError(t, err, "Case %d", i)

		output, err := exec.Command(path.Join(testOutputPath, "testout"), "foo").CombinedOutput()
		require.NoError(t, err, "Case %d\nOutput:\n%s", i, string(output))

		require.Equal(t, "testMain foo\n", string(output), "Case %d", i)

		output, err = exec.Command(path.Join(testOutputPath, "testout"), "bar").CombinedOutput()
		require.NoError(t, err, "Case %d", i)

		require.Equal(t, "testMain bar\n", string(output), "Case %d", i)
	}
}

func TestAmalgomateVendor(t *testing.T) {
	tmpDir, cleanup, err := dirs.TempDir(".", "")
	defer cleanup()
	require.NoError(t, err)

	for i, currCase := range []struct {
		input gofiles.GoFileSpec
		want  string
	}{
		{
			input: gofiles.GoFileSpec{
				RelPath: "testmain/vendor/testmain.go",
				Src: `package main

			import "fmt"

			func main() {
				fmt.Println("testMain 0")
			}
			`,
			},
			want: `package amalgomated

			import "fmt"

			func AmalgomatedMain() {
				fmt.Println("testMain 0")
			}
			`,
		},
	} {
		currCase.input.Src = unindent(currCase.input.Src)
		currCase.want = unindent(currCase.want)

		currTestDir := path.Join(tmpDir, fmt.Sprintf("test%d", i))
		err := os.Mkdir(currTestDir, 0755)
		require.NoError(t, err, "Case %d", i)

		files, err := gofiles.Write(currTestDir, []gofiles.GoFileSpec{currCase.input})
		require.NoError(t, err, "Case %d", i)

		testMainImportPath := files["testmain/vendor/testmain.go"].ImportPath
		confPath := path.Join(currTestDir, "test.conf")
		err = ioutil.WriteFile(confPath, []byte(getConfigString(t, amalgomate.Config{
			Pkgs: map[string]amalgomate.SrcPkg{
				"testmain": {
					MainPkg: testMainImportPath,
				},
			},
		})), 0644)
		require.NoError(t, err, "Case %d", i)

		testOutputPath := path.Join(currTestDir, "testout")

		runMainOutput, err := runMain(confPath, testOutputPath, "main")
		require.NoError(t, err, "Case %d\nOutput:\n%s", i, runMainOutput)

		outputTestMainDir := path.Join(testOutputPath, internalDir, testMainImportPath)
		writtenFileBytes, err := ioutil.ReadFile(path.Join(outputTestMainDir, "testmain.go"))
		require.NoError(t, err, "Case %d", i)

		require.Equal(t, currCase.want, string(writtenFileBytes), "Case %d", i)

		goBuild := exec.Command("go", "build")
		goBuild.Dir = testOutputPath
		output, err := goBuild.Output()
		require.NoError(t, err, "Case %d\nOutput:\n%s", i, string(output))

		output, err = exec.Command(path.Join(testOutputPath, "testout"), "testmain").Output()
		require.NoError(t, err, "Case %d\nOutput:\n%s", i, string(output))
		require.Equal(t, fmt.Sprintf("testMain %d\n", i), string(output), "Case %d", i)
	}
}

func TestAmalgomateInternal(t *testing.T) {
	tmpDir, cleanup, err := dirs.TempDir(".", "")
	defer cleanup()
	require.NoError(t, err)

	for i, currCase := range []struct {
		files []gofiles.GoFileSpec
		want  string
	}{
		{
			files: []gofiles.GoFileSpec{
				{
					RelPath: "testmain/testmain.go",
					Src:     `package main; import "{{index . "testmain/internal/helper/helper.go"}}"; func main() { helper.Call() }`,
				},
				{
					RelPath: "testmain/internal/helper/helper.go",
					Src:     `package helper; import "fmt"; func Call() { fmt.Println("testHelper 0") }`,
				},
			},
			want: `package amalgomated

			import "{{import}}"

			func AmalgomatedMain()	{ helper.Call() }
			`,
		},
		{
			files: []gofiles.GoFileSpec{
				{
					RelPath: "testmain/testmain.go",
					Src:     `package main; import renamed "{{index . "testmain/internal/helper/helper.go"}}"; func main() { renamed.Call() }`,
				},
				{
					RelPath: "testmain/internal/helper/helper.go",
					Src:     `package helper; import "fmt"; func Call() { fmt.Println("testHelper 1") }`,
				},
			},
			want: `package amalgomated

			import renamed "{{import}}"

			func AmalgomatedMain()	{ renamed.Call() }
			`,
		},
	} {
		currCase.want = unindent(currCase.want)

		currTestDir := path.Join(tmpDir, fmt.Sprintf("test%d", i))
		err := os.Mkdir(currTestDir, 0755)
		require.NoError(t, err, "Case %d", i)

		files, err := gofiles.Write(currTestDir, currCase.files)
		require.NoError(t, err, "Case %d", i)

		testMainImportPath := files["testmain/testmain.go"].ImportPath
		helperImportPath := files["testmain/internal/helper/helper.go"].ImportPath

		confPath := path.Join(currTestDir, "test.conf")
		err = ioutil.WriteFile(confPath, []byte(getConfigString(t, amalgomate.Config{
			Pkgs: map[string]amalgomate.SrcPkg{
				"testmain": {
					MainPkg: testMainImportPath,
				},
			},
		})), 0644)
		require.NoError(t, err, "Case %d", i)

		testOutputPath := path.Join(currTestDir, "testout")
		currCase.want = strings.Replace(currCase.want, "{{import}}", path.Join("github.com/palantir/amalgomate/cmd", testOutputPath, internalDir, helperImportPath), -1)

		outputStr, err := runMain(confPath, testOutputPath, "main")
		require.NoError(t, err, "main encountered error for case %d. Output:\n%s", i, outputStr)

		outputTestMainDir := path.Join(testOutputPath, internalDir, testMainImportPath)
		writtenFileBytes, err := ioutil.ReadFile(path.Join(outputTestMainDir, "testmain.go"))
		require.NoError(t, err, "Case %d", i)

		require.Equal(t, currCase.want, string(writtenFileBytes), "Case %d", i)

		goBuild := exec.Command("go", "build")
		goBuild.Dir = testOutputPath
		_, err = goBuild.Output()
		require.NoError(t, err)

		output, err := exec.Command(path.Join(testOutputPath, "testout"), "testmain").Output()
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("testHelper %d\n", i), string(output), "Case %d", i)
	}
}

func TestAmalgomateErrors(t *testing.T) {
	tmpDir, cleanup, err := dirs.TempDir(".", "")
	defer cleanup()
	require.NoError(t, err)

	for i, currCase := range []struct {
		input         gofiles.GoFileSpec
		expectedError string
	}{
		{
			input: gofiles.GoFileSpec{
				RelPath: "testmain/testmain.go",
				Src: `package main

			func foo() {}
			`,
			},
			expectedError: `(?s)^Error: failed to repackage files specified in configuration: main method not found in package .+/testmain.+`,
		},
		{
			input: gofiles.GoFileSpec{
				RelPath: "testmain/testmain.go",
				Src: `package main

			func main() {}

			func AmalgomatedMain() {}
			`,
			},
			expectedError: `(?s)^Error: failed to repackage files specified in configuration: failed to rename function in file .+/testmain/testmain.go: cannot rename function main to AmalgomatedMain because a function with the new name already exists.+`,
		},
	} {
		currCase.input.Src = unindent(currCase.input.Src)

		currTestDir := path.Join(tmpDir, fmt.Sprintf("test%d", i))
		err = os.Mkdir(currTestDir, 0755)
		require.NoError(t, err)

		files, err := gofiles.Write(currTestDir, []gofiles.GoFileSpec{currCase.input})
		require.NoError(t, err, "Case %d", i)

		testMainImportPath := files["testmain/testmain.go"].ImportPath
		confPath := path.Join(currTestDir, "test.conf")
		err = ioutil.WriteFile(confPath, []byte(getConfigString(t, amalgomate.Config{
			Pkgs: map[string]amalgomate.SrcPkg{
				"testmain": {
					MainPkg: testMainImportPath,
				},
			},
		})), 0644)
		require.NoError(t, err, "Case %d", i)

		testOutputPath := path.Join(currTestDir, "currTestOut")
		output, err := runMain(confPath, testOutputPath, "main")

		// expect error
		require.Error(t, err, fmt.Sprintf("Case %d\nOutput:\n%s", i, output))
		assert.Regexp(t, currCase.expectedError, output, "Case %d", i)
	}
}

func createArgs(args ...string) []string {
	var newArgs []string
	newArgs = append(newArgs, "--"+configFlagName, args[0])
	newArgs = append(newArgs, "--"+outputDirFlagName, args[1])
	newArgs = append(newArgs, "--"+pkgFlagName, args[2])
	return newArgs
}

func runMain(args ...string) (string, error) {
	args = createArgs(args...)
	output := &bytes.Buffer{}

	cmdCopy := *AmalgomateCmd
	cmdCopy.SetOutput(output)
	cmdCopy.SetArgs(args)
	err := cmdCopy.Execute()

	return output.String(), err
}

func getConfigString(t *testing.T, config amalgomate.Config) string {
	bytes, err := yaml.Marshal(config)
	require.NoError(t, err)
	return string(bytes)
}

func unindent(input string) string {
	return strings.Replace(input, "\n\t\t\t", "\n", -1)
}
