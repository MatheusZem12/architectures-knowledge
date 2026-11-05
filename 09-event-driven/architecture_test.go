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

// The rule of this architecture, as a test: no module imports another. Events are the only link.
func TestModulesOnlyTalkThroughEvents(t *testing.T) {
	err := filepath.WalkDir("modules", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			imported, _ := strconv.Unquote(spec.Path.Value)
			if strings.HasPrefix(imported, "eventdriven/modules/") {
				t.Errorf("%s must not import %s: publish an event instead", path, imported)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
