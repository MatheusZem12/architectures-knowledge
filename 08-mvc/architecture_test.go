package main

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"testing"
)

// The MVC rule, as a test: the model knows neither the controller nor the view, nor the web.
func TestModelStaysIndependent(t *testing.T) {
	banned := map[string]bool{"mvc/controllers": true, "mvc/views": true, "net/http": true, "encoding/json": true, "html/template": true}
	files, err := filepath.Glob("models/*.go")
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
			if banned[imported] {
				t.Errorf("%s must not import %s", path, imported)
			}
		}
	}
}
