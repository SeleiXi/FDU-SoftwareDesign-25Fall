package fs_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"softwaredesign/src/fs"
)

func TestDirTree(t *testing.T) {
	dir := t.TempDir()

	// Create test structure
	os.MkdirAll(filepath.Join(dir, "subdir"), 0o755)
	os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("content"), 0o644)
	os.WriteFile(filepath.Join(dir, "subdir", "file2.txt"), []byte("content"), 0o644)

	tree, err := fs.Tree(dir)
	if err != nil {
		t.Fatalf("dir tree failed: %v", err)
	}

	if !strings.Contains(tree, "file1.txt") {
		t.Fatalf("tree should contain file1.txt")
	}
	if !strings.Contains(tree, "subdir") {
		t.Fatalf("tree should contain subdir")
	}
	if !strings.Contains(tree, "file2.txt") {
		t.Fatalf("tree should contain file2.txt")
	}
}

func TestDirTreeEmpty(t *testing.T) {
	dir := t.TempDir()
	tree, err := fs.Tree(dir)
	if err != nil {
		t.Fatalf("dir tree failed: %v", err)
	}
	// Empty directory returns empty string, which is valid
	if tree != "" {
		t.Fatalf("empty dir should return empty string, got: %s", tree)
	}
}

