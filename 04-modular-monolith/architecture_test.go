package main

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// allowed is the official map of who may depend on whom. Changing this map is an
// architecture decision, and it stays visible in code review.
var allowed = map[string][]string{
	"catalog":       {},
	"customers":     {},
	"payments":      {},
	"orders":        {"catalog", "customers", "payments"},
	"notifications": {"orders"},
}

// Go's "internal/" already blocks anyone reaching into another module. This test
// covers what the compiler cannot see: a module using the public API of one it should not.
func TestModulesOnlyDependOnAllowedModules(t *testing.T) {
	const prefix = "modular/modules/"
	err := filepath.WalkDir("modules", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		module := strings.Split(filepath.ToSlash(path), "/")[1]
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			imported, _ := strconv.Unquote(spec.Path.Value)
			if !strings.HasPrefix(imported, prefix) {
				continue
			}
			target := strings.Split(strings.TrimPrefix(imported, prefix), "/")[0]
			if target != module && !contains(allowed[module], target) {
				t.Errorf("%s: module %s must not depend on %s", path, module, target)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
