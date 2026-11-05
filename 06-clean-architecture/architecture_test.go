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

// The dependency rule, written as a test: each ring only sees the rings inside it.
func TestDependencyRule(t *testing.T) {
	rings := map[string][]string{
		"entities":             {"cleanarch/usecases", "cleanarch/adapters", "cleanarch/frameworks", "net/http", "encoding/json", "database/sql"},
		"usecases":             {"cleanarch/adapters", "cleanarch/frameworks", "net/http", "encoding/json", "database/sql"},
		"adapters":             {"cleanarch/frameworks"},
		"adapters/controllers": {"net/http"},
		"adapters/presenters":  {"net/http"},
	}
	for dir, banned := range rings {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
				return err
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, spec := range file.Imports {
				imported, _ := strconv.Unquote(spec.Path.Value)
				for _, ban := range banned {
					if imported == ban || strings.HasPrefix(imported, ban+"/") {
						t.Errorf("%s must not import %s", path, imported)
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
