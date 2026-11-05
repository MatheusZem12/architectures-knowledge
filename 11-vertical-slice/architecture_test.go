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

// The vertical slice rule, as a test: a feature must not depend on another feature.
func TestSlicesDoNotKnowEachOther(t *testing.T) {
	err := filepath.WalkDir("features", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			imported, _ := strconv.Unquote(spec.Path.Value)
			if strings.HasPrefix(imported, "verticalslice/features/") {
				t.Errorf("%s must not import %s", path, imported)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
