package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The serverless rules, as a test. A function may hold no state between calls, because the instance
// can vanish at any time. And a function may not call another directly: they only share services.
func TestFunctionsAreStatelessAndIndependent(t *testing.T) {
	err := filepath.WalkDir("functions", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		for _, decl := range file.Decls {
			if general, ok := decl.(*ast.GenDecl); ok && general.Tok == token.VAR {
				t.Errorf("%s: a package-level variable is state that would be lost with the instance", path)
			}
		}
		for _, spec := range file.Imports {
			imported, _ := strconv.Unquote(spec.Path.Value)
			if strings.HasPrefix(imported, "serverless/functions/") && imported != "serverless/functions/shared" {
				t.Errorf("%s must not import %s", path, imported)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
