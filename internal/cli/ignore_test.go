package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestNewIgnoreMatcher(t *testing.T) {
	testCases := []struct {
		name         string
		ignoredPaths []string
		wantNames    []string
		wantPaths    []string
	}{
		{
			name:         "normalize multiple names and paths",
			ignoredPaths: []string{"prisma", ".github", "./test/", "src/config/", ".env"},
			wantNames:    []string{"prisma", ".github", ".env"},
			wantPaths:    []string{"test", "src/config"},
		},
		{
			name:         "ignore empty and whitespace patterns",
			ignoredPaths: []string{"", "   ", "prisma", "  ", "./test/"},
			wantNames:    []string{"prisma"},
			wantPaths:    []string{"test"},
		},
		{
			name:         "ignore duplicate patterns",
			ignoredPaths: []string{"prisma", "prisma", "./test/", "test/"},
			wantNames:    []string{"prisma"},
			wantPaths:    []string{"test"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matcher := NewIgnoreMatcher(tc.ignoredPaths)

			if len(matcher.names) != len(tc.wantNames) {
				t.Errorf("names count = %d, want %d", len(matcher.names), len(tc.wantNames))
			}

			if len(matcher.paths) != len(tc.wantPaths) {
				t.Errorf("paths count = %d, want %d", len(matcher.paths), len(tc.wantPaths))
			}

			for _, wantName := range tc.wantNames {
				if _, exists := matcher.names[wantName]; !exists {
					t.Errorf("NewIgnoreMatcher() missing name %q", wantName)
				}
			}

			for _, wantPath := range tc.wantPaths {
				if _, exists := matcher.paths[wantPath]; !exists {
					t.Errorf("NewIgnoreMatcher() missing path %q", wantPath)
				}
			}
		})
	}
}

func TestShouldIgnore(t *testing.T) {
	matcher := IgnoreMatcher{
		names: map[string]struct{}{
			".git":         {},
			"node_modules": {},
			"temp.log":     {},
		},
		paths: map[string]struct{}{
			"src/secret": {},
			"build/out":  {},
		},
	}

	root := t.TempDir()

	dirs := []string{
		"node_modules",
		"src",
		"src/logs",
		"src/secret",
		"src/secret-public",
		"build",
		"build/out",
	}

	for _, dir := range dirs {
		dirPath := filepath.Join(root, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("failed to create a directory %s: %v", dir, err)
		}
	}

	files := map[string]string{
		"src/logs/temp.log": "temporary log",
		"main.go":           "package main",
	}

	for relPath, content := range files {
		fullPath := filepath.Join(root, relPath)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write a file %s: %v", relPath, err)
		}
	}

	testCases := []struct {
		name     string
		subPath  string
		expected bool
	}{
		{
			name:     "ignore by name directory",
			subPath:  "node_modules",
			expected: true,
		},
		{
			name:     "ignore by name deep file",
			subPath:  "src/logs/temp.log",
			expected: true,
		},
		{
			name:     "ignore by exact path",
			subPath:  "src/secret",
			expected: true,
		},
		{
			name:     "do not ignore similar path",
			subPath:  "src/secret-public",
			expected: false,
		},
		{
			name:     "do not ignore normal file",
			subPath:  "main.go",
			expected: false,
		},
		{
			name:     "do not ignore normal directory",
			subPath:  "src",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(root, tc.subPath)

			entries, err := os.ReadDir(filepath.Dir(path))
			if err != nil {
				t.Fatalf("failed to read directory %s: %v", filepath.Dir(path), err)
			}

			var entry fs.DirEntry
			for _, candidate := range entries {
				if candidate.Name() == filepath.Base(path) {
					entry = candidate
					break
				}
			}

			if entry == nil {
				t.Fatalf("entry not found: %s", tc.subPath)
			}

			got := matcher.ShouldIgnore(root, path, entry)
			if got != tc.expected {
				t.Errorf("ShouldIgnore() = %v, want %v", got, tc.expected)
			}
		})
	}
}
