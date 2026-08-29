package frontend

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractWritesEmbeddedApplication(t *testing.T) {
	dir := t.TempDir()
	if err := Extract(dir); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if len(content) == 0 {
		t.Fatal("embedded index.html is empty")
	}
}
