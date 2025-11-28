package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Tree renders a directory tree rooted at path.
func Tree(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s 不是目录", path)
	}
	entries, err := readEntries(abs)
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return "", nil
	}
	var lines []string
	for i, entry := range entries {
		last := i == len(entries)-1
		lines = append(lines, formatEntry(entry, abs, "", last)...)
	}
	return strings.Join(lines, "\n"), nil
}

func formatEntry(entry os.DirEntry, parent, prefix string, last bool) []string {
	connector := "├── "
	nextPrefix := prefix + "│   "
	if last {
		connector = "└── "
		nextPrefix = prefix + "    "
	}
	line := fmt.Sprintf("%s%s%s", prefix, connector, entry.Name())
	lines := []string{line}
	if entry.IsDir() {
		childEntries, err := readEntries(filepath.Join(parent, entry.Name()))
		if err != nil {
			return append(lines, fmt.Sprintf("%s%s<error: %v>", nextPrefix, "├── ", err))
		}
		for i, child := range childEntries {
			childLast := i == len(childEntries)-1
			lines = append(lines, formatEntry(child, filepath.Join(parent, entry.Name()), nextPrefix, childLast)...)
		}
	}
	return lines
}

func readEntries(path string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() == entries[j].IsDir() {
			return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
		}
		return entries[i].IsDir()
	})
	return entries, nil
}
