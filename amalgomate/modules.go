// Copyright (c) 2021 Palantir Technologies Inc. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.

package amalgomate

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nmiyake/pkg/dirs"
	"github.com/pkg/errors"
	"github.com/termie/go-shutil"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
)

// rewriteImports rewrites all of the imports in the repackagedModuleRootDir that resolve to a package that belongs to
// the module specified by moduleImportPath so that they have "importPathToRepackagedModule" added as a path prefix.
// For example, if the moduleImportPath is "github.com/repackage" and the importPathToRepackagedModule is
// "github.com/mainmodule/generated_src/repackage/internal" and an import to "github.com/repackage/innerpkg" is found
// (and that import is part of the "github.com/repackage" module), then the import is rewritten to be
// "github.com/mainmodule/generated_src/repackage/internal/github.com/repackage/innerpkg". The moduleImportPath is
// required to verify that references to other modules are not renamed even if they share a prefix: for example, it is
// possible that "github.com/repackage/nested-module" is defined as a separate module, and in that case any imports to
// that path or subpath would not be rewritten, even though from a "path" perspective it would seem that this might be
// part of the "github.com/repackage" module. The import operation that determines whether a package is part of a module
// is performed relative to "repackagedModuleRootDir".
func rewriteImports(repackagedModuleRootDir, moduleImportPath, importPathToRepackagedModule string) error {
	fileSet := token.NewFileSet()
	foundMain := false
	flagPkgImported := false

	if err := filepath.WalkDir(repackagedModuleRootDir, func(fpath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// skip non-Go files
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}

		updated := false

		fileNode, err := parser.ParseFile(fileSet, fpath, nil, parser.ParseComments)
		if err != nil {
			return errors.Wrapf(err, "failed to parse file %s", fpath)
		}

		for _, currImport := range fileNode.Imports {
			currImportPathUnquoted, err := strconv.Unquote(currImport.Path.Value)
			if err != nil {
				return errors.Wrapf(err, "unable to unquote import %s", currImport.Path.Value)
			}

			if currImportPathUnquoted != "flag" {
				// no need to repackage standard library packages that are not "flag"
				if inStandardLibrary(currImportPathUnquoted) {
					continue
				}

				goModInfo, err := moduleInfoForPackage(currImportPathUnquoted, repackagedModuleRootDir)
				if err != nil {
					return err
				}

				// import belongs to module other than one being repackaged: nothing to do
				if goModInfo.Path != moduleImportPath {
					continue
				}
			}
			updated = true

			var updatedImport string
			if currImportPathUnquoted == "flag" {
				flagPkgImported = true
				updatedImport = filepath.Join(importPathToRepackagedModule, "amalgomated_flag")
			} else {
				updatedImport = path.Join(importPathToRepackagedModule, currImportPathUnquoted)
			}

			if !astutil.RewriteImport(fileSet, fileNode, currImportPathUnquoted, updatedImport) {
				return errors.Errorf("failed to rewrite import from %s to %s", currImportPathUnquoted, updatedImport)
			}

			removeImportPathChecking(fileNode)
		}

		// change package name for main packages
		if fileNode.Name.Name == "main" {
			updated = true

			fileNode.Name = ast.NewIdent(amalgomatedPackage)

			// find the main function
			mainFunc := findFunction(fileNode, "main")
			if mainFunc != nil {
				err = renameFunction(fileNode, "main", amalgomatedMain)
				if err != nil {
					return errors.Wrapf(err, "failed to rename function in file %s", fpath)
				}
				foundMain = true
			}
		}

		if !updated {
			return nil
		}

		if err = writeAstToFile(fpath, fileNode, fileSet); err != nil {
			return errors.Wrapf(err, "failed to write rewritten file %s", fpath)
		}
		return nil
	}); err != nil {
		return errors.Wrapf(err, "failed to walk directory %s", repackagedModuleRootDir)
	}

	if !foundMain {
		return errors.Errorf("main method not found in repackaged module directory tree %s", repackagedModuleRootDir)
	}

	if flagPkgImported {
		// if "flag" package is imported, add "flag" as a rewritten dependency. This is done because flag.CommandLine is
		// a global variable that is often used by programs and problems can arise if multiple amalgomated programs use
		// it.
		goRoot, err := dirs.GoRoot()
		if err != nil {
			return errors.WithStack(err)
		}
		fmtSrcDir := path.Join(goRoot, "src", "flag")
		fmtDstDir := path.Join(repackagedModuleRootDir, "amalgomated_flag")
		if err := shutil.CopyTree(fmtSrcDir, fmtDstDir, nil); err != nil {
			return errors.Wrapf(err, "failed to copy directory %s to %s", fmtSrcDir, fmtDstDir)
		}
		if _, err := removeEmptyDirs(fmtDstDir); err != nil {
			return errors.Wrapf(err, "failed to remove empty directories in destination %s", fmtDstDir)
		}
	}
	return nil
}

// copyModuleRecursively recursively copies the module with the canonical name modulePath from srcDir into dstDir. Only
// copies files with the suffix ".go", omits files with the suffix "_test.go" and skips all directories named "vendor".
// The contents of srcDir are copied into the directory path that consists of the module path converted into a file
// path. That is, if the source module has the name "github.com/foo/bar", then the directory path to
// "dstDir/github.com/foo/bar" is created and made to contain all of the contents of srcDir that are part of the module
// modulePath except for the "go.mod" and "go.sum" files and any "vendor" directories. The destination path creation is
// done on the full module name, including version suffixes such as "/v2".
//
// The modulePath is relevant because it is possible for a module directory to contain content that is not part of the
// module and this operation explicitly does not copy such files. For example, the repository
// "github.com/nmiyake/minimal-module" contains a "go.mod" defining the module "github.com/nmiyake/minimal-module", but
// also contains "nested-module/go.mod" that defines the separate "github.com/nmiyake/minimal-module/nested-module"
// module.
//
// srcDir and dstDir must both be directories that exist and it must be possible to create directories that lead to the
// "dstDir/modulePath" path. The permissions for all created directories will be 0755 regardless of the source directory
// permissions. Does not follow symlinks.
func copyModuleRecursively(modulePath, srcDir, dstDir string) error {
	if !filepath.IsAbs(srcDir) {
		srcDirAbsPath, err := filepath.Abs(srcDir)
		if err != nil {
			return errors.Wrapf(err, "failed to convert %s into absolute path", srcDir)
		}
		srcDir = srcDirAbsPath
	}

	if fi, err := os.Stat(srcDir); err != nil {
		return errors.Wrapf(err, "failed to stat %s", srcDir)
	} else if !fi.IsDir() {
		return errors.Errorf("srcDir %s is not a directory", srcDir)
	}

	if fi, err := os.Stat(dstDir); err != nil {
		return errors.Wrapf(err, "failed to stat %s", dstDir)
	} else if !fi.IsDir() {
		return errors.Errorf("dstDir %s is not a directory", dstDir)
	}

	dstRootPath := filepath.Join(dstDir, modulePath)
	if err := os.MkdirAll(dstRootPath, 0755); err != nil {
		return errors.Wrapf(err, "failed to create directories to %s", dstRootPath)
	}

	if err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			// fully skip any directories named "vendor"
			if d.Name() == "vendor" {
				return fs.SkipDir
			}

			// if this is a directory, verify that it is part of the desired module. If not, do not process the
			// directory or any of its contents.
			currPathModulePath, err := modulePathForDirectory(path)
			if err != nil {
				return err
			}
			if currPathModulePath != modulePath {
				return fs.SkipDir
			}
		} else if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			// skip non-".go" files and "_test.go" files
			return nil
		}

		// translate path to be relative to the source directory
		relPathToSrc, err := filepath.Rel(srcDir, path)
		if err != nil {
			return errors.Wrapf(err, "failed to make %s relative to %s", path, srcDir)
		}
		if relPathToSrc == "" || relPathToSrc == "." {
			// skip root directory because it is already created
			return nil
		}

		dstPath := filepath.Join(dstRootPath, relPathToSrc)
		if d.IsDir() {
			if err := os.Mkdir(dstPath, 0755); err != nil {
				return errors.Wrapf(err, "failed to create directory at %s", dstPath)
			}
		} else {
			if err := shutil.CopyFile(path, dstPath, false); err != nil {
				return errors.Wrapf(err, "failed to copy %s to %s", path, dstPath)
			}
		}
		return nil
	}); err != nil {
		return errors.Wrapf(err, "failed to walk directory %s", srcDir)
	}
	return nil
}

// moduleInfoForPackage returns the GoModInfo for the package with the specified import path resolved in the provided
// directory. The returned GoModInfo contains the module path (the module name/import path) and the path to the
// directory on disk where the module is located. The module path on disk will reflect the location from which the
// source files will be obtained to build the package in the provided directory: for example, the local module path, a
// local directory specified in a "replace" directive in the "go.mod" file of the module in which "dir" is located, the
// vendor directory of the module in which "dir" is located, etc.
func moduleInfoForPackage(pkgName, dir string) (*GoModInfo, error) {
	// get package information from package name, which should include the module information
	outputDirPkg, err := packageForPatternInDirectory(pkgName, dir, packages.NeedName|packages.NeedFiles|packages.NeedModule)
	if err != nil {
		return nil, err
	}
	if len(outputDirPkg.GoFiles) == 0 {
		return nil, errors.Errorf("no Go files in package %s resolved from directory %s", pkgName, dir)
	}

	if outputDirPkg.Module == nil {
		return nil, errors.Errorf("unable to determine module for package %s resolved from directory %s", pkgName, dir)
	}

	// determine the module for the specified package (may differ from the package because the main package for the
	// module may not be in the root directory of the module)
	modulePath := outputDirPkg.Module.Path
	if modulePath == "" {
		return nil, errors.Errorf("could not determine module for package %q in directory %q", pkgName, dir)
	}

	// if resolved module has directory field, use it
	if outputDirPkg.Module.Dir != "" {
		return &GoModInfo{
			Path: modulePath,
			Dir:  outputDirPkg.Module.Dir,
		}, nil
	}

	// use "go list -e -json" for the module in the specified directory. Do this to determine the module directory
	// (which may be a resolved directory like the vendor directory). So far, this seems to be the only way to reliably
	// determine the on-disk path to the module for a package that may not have the same import path as its module.
	// Run with the "-e" flag because this command returns an error if the specified module path does not contain any
	// ".go" files (even if it is a valid module root). However, in this failure mode the "Path" field is still
	// populated correctly, which is the only information that is needed.
	goListCmd := exec.Command("go", "list", "-e", "-json", modulePath)
	goListCmd.Dir = dir
	output, err := goListCmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrapf(err, "faied to run command %v: %s", goListCmd.Args, string(output))
	}
	// only unmarshal "Dir" field
	dirStruct := struct {
		Dir string
	}{}
	if err := json.Unmarshal(output, &dirStruct); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal JSON: %s", string(output))
	}
	return &GoModInfo{
		Dir:  dirStruct.Dir,
		Path: modulePath,
	}, nil
}

// packageForPatternInDirectory returns the *package.Package loaded for the provided pattern resolved in the provided
// directory using the provided packages.LoadMode. Returns an error if there are errors loading the package information
// or if no packages are returned. If multiple packages are loaded, returns the first one.
func packageForPatternInDirectory(pattern, dir string, mode packages.LoadMode) (*packages.Package, error) {
	outputDirPkgs, err := packages.Load(&packages.Config{
		Dir:  dir,
		Mode: mode,
	}, pattern)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to determine package for directory %s", dir)
	}
	if len(outputDirPkgs) == 0 {
		return nil, errors.Errorf("no packages found in directory %s", dir)
	}
	return outputDirPkgs[0], nil
}

type GoModInfo struct {
	// Module path
	Path string
	// Path to the module directory
	Dir string
}

// modulePathForDirectory returns the module path for the specified directory. The implementation of this function
// differs from moduleInfoForDirectory because it uses package loading to determine the information rather than the
// "go list" command run in a directory. The latter does not work for vendor directories after Go 1.17 (after which
// go.mod files for vendored modules are no longer included). On the other hand, "moduleInfoForDirectory" cannot use
// this implementation because the module information returned by the package lookup does not include the directory for
// the module.
func modulePathForDirectory(dir string) (string, error) {
	dirPkg, err := packageForPatternInDirectory(dir, dir, packages.NeedModule)
	if err != nil {
		return "", errors.Wrapf(err, "failed to determine package for directory")
	}
	if dirPkg.Module == nil {
		// if package lookup didn't work, then fall back on moduleInfoForDirectory
		modInfo, err := moduleInfoForDirectory(dir)
		if err != nil {
			return "", errors.Wrapf(err, "failed to fall back on moduleInfoForDirectory")
		}
		return modInfo.Path, nil
	}
	return dirPkg.Module.Path, nil
}

// moduleInfoForDirectory returns the *GoModInfo for the specified directory. Returns the result of running
// "go list -mod=readonly -m -json" using the provided directory as the working directory.
func moduleInfoForDirectory(dir string) (*GoModInfo, error) {
	goListCmd := exec.Command("go", "list", "-mod=readonly", "-m", "-json")
	goListCmd.Dir = dir
	output, err := goListCmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to execute command %v in directory %s. Output: %s", goListCmd.Args, dir, string(output))
	}
	var modInfo GoModInfo
	if err := json.Unmarshal(output, &modInfo); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal JSON: %s", string(output))
	}
	if modInfo.Path == "command-line-arguments" {
		return nil, errors.Errorf("directory %s is not a valid module", dir)
	}
	return &modInfo, nil
}
