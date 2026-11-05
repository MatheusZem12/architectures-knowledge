package main

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The CQRS rule, as a test: neither side can see the other. Events are the only bridge.
func TestSidesDoNotKnowEachOther(t *testing.T) {
	banned := map[string][]string{
		"write":  {"cqrs/read", "cqrs/api", "net/http", "encoding/json"},
		"read":   {"cqrs/write", "cqrs/api", "net/http"},
		"events": {"cqrs/read", "cqrs/write", "cqrs/api"},
	}
	for dir, imports := range banned {
		files, err := filepath.Glob(dir + "/*.go")
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range files {
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatal(err)
			}
			for _, spec := range file.Imports {
				imported, _ := strconv.Unquote(spec.Path.Value)
				for _, ban := range imports {
					if strings.HasPrefix(imported, ban) {
						t.Errorf("%s must not import %s", path, imported)
					}
				}
			}
		}
	}
}
