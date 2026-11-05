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

// The microkernel rules, as a test: the kernel knows no plugin, and plugins know only the kernel.
func TestKernelDoesNotKnowPlugins(t *testing.T) {
	check := func(root string, banned func(string) bool) {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
				return err
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, spec := range file.Imports {
				imported, _ := strconv.Unquote(spec.Path.Value)
				if banned(imported) {
					t.Errorf("%s must not import %s", path, imported)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	check("kernel", func(imported string) bool { return strings.HasPrefix(imported, "microkernel/plugins") })
	check("plugins", func(imported string) bool { return strings.HasPrefix(imported, "microkernel/plugins") })
}
