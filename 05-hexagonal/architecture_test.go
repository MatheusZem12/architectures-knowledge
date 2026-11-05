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

// The whole hexagonal rule fits here: dependencies point inward.
// If the domain imports net/http tomorrow, this test breaks the build.
func TestDependenciesPointInward(t *testing.T) {
	forbidden := map[string][]string{
		"internal/core/domain": {"hexagonal/internal", "net/http", "encoding/json", "database/sql", "os"},
		"internal/core/ports":  {"hexagonal/internal/adapters", "hexagonal/internal/core/app", "net/http"},
		"internal/core/app":    {"hexagonal/internal/adapters", "net/http", "encoding/json", "database/sql"},
	}
	for dir, banned := range forbidden {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			// The app test uses the memory adapter on purpose: it is test code.
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
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
