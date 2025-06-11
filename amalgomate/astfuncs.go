// Copyright (c) 2016 Palantir Technologies Inc. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.

package amalgomate

import (
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
)

const (
	amalgomatedPackage = "amalgomated"
	amalgomatedMain    = "AmalgomatedMain"
	internalDir        = "internal"
)

// repackage repackages the module for the main package specified in the provided configuration and writes the
// repackaged files into the provided output directory. The repackaged files are placed into a directory called
// "internal" that is created in the provided directory. This function assumes and verifies that the provided
// "outputDir" is a directory that exists. The provided configuration is processed based on the natural ordering of the
// name of the commands.
func repackage(config Config, outputDir string) error {
	if outputDirInfo, err := os.Stat(outputDir); err != nil {
		return errors.Wrapf(err, "failed to stat output directory: %s", outputDir)
	} else if !outputDirInfo.IsDir() {
		return errors.Wrapf(err, "not a directory: %s", outputDir)
	}

	internalDir := filepath.Join(outputDir, internalDir)
	// remove output directory if it already exists
	if err := os.RemoveAll(internalDir); err != nil {
		return errors.Wrapf(err, "failed to remove directory: %s", internalDir)
	}

	if err := os.Mkdir(internalDir, 0755); err != nil {
		return errors.Wrapf(err, "failed to create %s directory at %s", internalDir, internalDir)
	}

	projectModuleInfo, err := moduleInfoForDirectory(outputDir)
	if err != nil {
		return errors.Wrapf(err, "failed to determine module for directory %s", outputDir)
	}

	relPathFromModuleToOutputDir, err := relpathNormalizedPaths(projectModuleInfo.Dir, internalDir)
	if err != nil {
		return err
	}

	for _, currConfigKey := range sortedKeys(config.Pkgs) {
		currMainPkg := config.Pkgs[currConfigKey]

		currMainPkgModule, err := moduleInfoForPackage(currMainPkg.MainPkg, outputDir)
		if err != nil {
			return errors.Wrapf(err, "failed to determine module for main package")
		}

		if currMainPkgModule.Path == projectModuleInfo.Path {
			return errors.Errorf("module for package %s was reported as %s, which is the same as the project module: it is likely that this package is not part of a real module, and repackaging non-modules is not supported", currMainPkg.MainPkg, currMainPkgModule.Path)
		}

		if err := copyModuleRecursively(currMainPkgModule.Path, currMainPkgModule.Dir, filepath.Join(projectModuleInfo.Dir, relPathFromModuleToOutputDir)); err != nil {
			return errors.Wrapf(err, "failed to copy module")
		}
		if err := rewriteImports(internalDir, currMainPkgModule.Path, path.Join(projectModuleInfo.Path, relPathFromModuleToOutputDir), currMainPkg.DoNotRewriteFlagImport); err != nil {
			return errors.Wrapf(err, "failed to rewrite imports for module %+v", currMainPkgModule)
		}
	}
	return nil
}

// removeEmptyDirs removes all directories in rootDir (including the root directory itself) that are empty or contain
// only empty directories.
func removeEmptyDirs(rootDir string) (removed bool, rErr error) {
	dirContent, err := ioutil.ReadDir(rootDir)
	if err != nil {
		return false, errors.Wrapf(err, "failed to read directory")
	}

	removeCurrentDir := true
	for _, fi := range dirContent {
		if !fi.IsDir() {
			// if directory contains non-directory, it will not be removed
			removeCurrentDir = false
			continue
		}
		currChildRemoved, err := removeEmptyDirs(path.Join(rootDir, fi.Name()))
		if err != nil {
			return false, err
		}
		if !currChildRemoved {
			// if a child directory was not removed, it means it was non-empty, so this directory is non-empty
			removeCurrentDir = false
			continue
		}
	}

	if !removeCurrentDir {
		return false, nil
	}
	if err := os.Remove(rootDir); err != nil {
		return false, errors.Wrapf(err, "failed to remove directory")
	}
	return true, nil
}

func removeImportPathChecking(fileNode *ast.File) {
	var newCgList []*ast.CommentGroup
	for _, cg := range fileNode.Comments {
		var newCommentList []*ast.Comment
		for _, cc := range cg.List {
			// assume that any comment that starts with "// import" or "/* import" are import path checking
			// comments and don't add them to the new slice. This may omit some comments that are not
			// actually import checks, but downside is limited (it will just omit comment from repacked file).
			if !(strings.HasPrefix(cc.Text, "// import") || strings.HasPrefix(cc.Text, "/* import")) {
				newCommentList = append(newCommentList, cc)
			}
		}
		cg.List = newCommentList

		// CommentGroup assumes that len(List) > 0, so if logic above causes group to be empty, omit
		if len(cg.List) != 0 {
			newCgList = append(newCgList, cg)
		}
	}
	fileNode.Comments = newCgList
}

func addImports(file *ast.File, fileSet *token.FileSet, outputDir string, config Config) error {
	projectModule, err := moduleInfoForDirectory(outputDir)
	if err != nil {
		return errors.Wrapf(err, "failed to determine module for directory %s", outputDir)
	}

	pathFromProjectModuleToOutputDir, err := relpathNormalizedPaths(projectModule.Dir, outputDir)
	if err != nil {
		return err
	}

	processedPkgs := make(map[string]bool, len(config.Pkgs))
	for _, name := range sortedKeys(config.Pkgs) {
		progPkg := config.Pkgs[name]

		mainPkgInfo, err := packageForPatternInDirectory(progPkg.MainPkg, outputDir, packages.NeedName|packages.NeedFiles)
		if err != nil {
			return errors.Wrapf(err, "failed to get package information")
		}

		// repackaged import path is the project module import path + path to the output directory + internalDir + main package import path
		repackagedImportPath := path.Join(projectModule.Path, pathFromProjectModuleToOutputDir, internalDir, mainPkgInfo.PkgPath)
		added := astutil.AddNamedImport(fileSet, file, name, repackagedImportPath)
		if !added {
			return errors.Errorf("failed to add import %s", repackagedImportPath)
		}
		processedPkgs[progPkg.MainPkg] = true
	}
	return nil
}

// relpathNormalizedPaths makes targpath relative to basepath using filepath.Rel after normalizing both inputs using
// filepath.EvalSymLinks.
func relpathNormalizedPaths(basepath, targpath string) (string, error) {
	normalizedBase, err := filepath.EvalSymlinks(basepath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to normalize %s", basepath)
	}
	normalizedTarget, err := filepath.EvalSymlinks(targpath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to normalize %s", targpath)
	}

	relPath, err := filepath.Rel(normalizedBase, normalizedTarget)
	if err != nil {
		return "", errors.Wrapf(err, "failed to make %s relative to %s", normalizedTarget, normalizedBase)
	}
	return relPath, nil
}

func sortImports(file *ast.File) {
	for _, decl := range file.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
			sort.Sort(importSlice(gen.Specs))
			break
		}
	}
}

func getFirstToken(file *ast.File, t token.Token) *ast.GenDecl {
	for _, currDecl := range file.Decls {
		switch currDecl.(type) {
		case *ast.GenDecl:
			genDecl := currDecl.(*ast.GenDecl)
			if genDecl.Tok == t {
				return genDecl
			}
		}
	}
	return nil
}

func setVarCompositeLiteralElements(file *ast.File, constName string, elems []ast.Expr) error {
	decl := getFirstToken(file, token.VAR)
	if decl == nil {
		return errors.Errorf("could not find token of type VAR in %s", file.Name)
	}

	var constExpr ast.Expr
	for _, currSpec := range decl.Specs {
		// declaration is already known to be of type const, so all of the specs are ValueSpec
		valueSpec := currSpec.(*ast.ValueSpec)
		for i, valueSpecName := range valueSpec.Names {
			if valueSpecName.Name == constName {
				constExpr = valueSpec.Values[i]
				break
			}
		}
	}

	if constExpr == nil {
		return errors.Errorf("could not find variable with name %s in given declaration", constName)
	}

	var compLit *ast.CompositeLit
	var ok bool
	if compLit, ok = constExpr.(*ast.CompositeLit); !ok {
		return errors.Errorf("variable %s did not have a composite literal value", constName)
	}

	compLit.Elts = elems

	return nil
}

func createMapLiteralEntries(pkgs map[string]SrcPkg) []ast.Expr {
	// if multiple commands refer to the same package, the command that is lexicographically first is the one that
	// is used for the named import. Create a map that stores the mapping from the package to the import name.
	pkgToFirstCmdMap := make(map[string]string, len(pkgs))
	for _, name := range sortedKeys(pkgs) {
		if _, ok := pkgToFirstCmdMap[pkgs[name].MainPkg]; !ok {
			// add mapping only if it does not already exist
			pkgToFirstCmdMap[pkgs[name].MainPkg] = name
		}
	}

	var entries []ast.Expr
	for _, name := range sortedKeys(pkgs) {
		entries = append(entries, createMapKeyValueExpression(name, pkgToFirstCmdMap[pkgs[name].MainPkg]))
	}
	return entries
}

// createMapKeyValueExpression creates a new map key value function expression of the form "{{name}}": func() { {{namedImport}}.{{amalgomatedMain}}() }.
// In most cases "name" and "namedImport" will be the same, but if multiple commands refer to the same package, then the
// commands that are lexicographically later should refer to the named import of the first command.
func createMapKeyValueExpression(name, namedImport string) *ast.KeyValueExpr {
	return &ast.KeyValueExpr{
		Key: &ast.BasicLit{
			Kind:  token.STRING,
			Value: fmt.Sprintf(`"%v"`, name),
		},
		Value: &ast.FuncLit{
			Type: &ast.FuncType{
				Params: &ast.FieldList{},
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ast.NewIdent(namedImport),
								Sel: ast.NewIdent(amalgomatedMain),
							},
						},
					},
				},
			},
		},
	}
}

func renameFunction(fileNode *ast.File, originalName, newName string) error {
	originalFunc := findFunction(fileNode, originalName)
	if originalFunc == nil {
		return errors.Errorf("function %s does not exist", originalName)
	}

	if findFunction(fileNode, newName) != nil {
		return errors.Errorf("cannot rename function %s to %s because a function with the new name already exists", originalName, newName)
	}

	originalFunc.Name = ast.NewIdent(newName)
	return nil
}

func findFunction(fileNode *ast.File, funcName string) *ast.FuncDecl {
	for _, currDecl := range fileNode.Decls {
		switch t := currDecl.(type) {
		case *ast.FuncDecl:
			if t.Name.Name == funcName {
				return currDecl.(*ast.FuncDecl)
			}
		}
	}
	return nil
}

func writeAstToFile(path string, fileNode *ast.File, fileSet *token.FileSet) (writeErr error) {
	outputFile, err := os.Create(path)
	if err != nil {
		return errors.Wrapf(err, "failed to create file %s", path)
	}
	defer func() {
		if err := outputFile.Close(); err != nil {
			writeErr = errors.Errorf("failed to close file %s", path)
		}
	}()
	if err := printer.Fprint(outputFile, fileSet, fileNode); err != nil {
		return errors.Wrapf(err, "failed to write to file %s", path)
	}
	return nil
}

func sortedKeys(pkgs map[string]SrcPkg) []string {
	sortedKeys := make([]string, 0, len(pkgs))
	for currKey := range pkgs {
		sortedKeys = append(sortedKeys, currKey)
	}
	sort.Strings(sortedKeys)
	return sortedKeys
}
