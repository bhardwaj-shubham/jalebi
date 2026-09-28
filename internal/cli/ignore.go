package cli

import (
	"io/fs"
	"path/filepath"
	"strings"
)

type IgnoreMatcher struct {
	names map[string]struct{}
	paths map[string]struct{}
}

func NewIgnoreMatcher(ignoredPaths []string) IgnoreMatcher {
	matcher := IgnoreMatcher{
		names: make(map[string]struct{}),
		paths: make(map[string]struct{}),
	}

	for _, ignoredPath := range ignoredPaths {
		ignoredPath = strings.TrimSpace(ignoredPath)

		if ignoredPath == "" {
			continue
		}

		isPath := strings.HasPrefix(ignoredPath, "./") ||
			strings.HasSuffix(ignoredPath, "/") ||
			strings.Contains(ignoredPath, "/")

		ignoredPath = strings.TrimPrefix(ignoredPath, "./")
		ignoredPath = strings.TrimSuffix(ignoredPath, "/")

		if isPath {
			matcher.paths[ignoredPath] = struct{}{}
		} else {
			matcher.names[ignoredPath] = struct{}{}
		}
	}

	return matcher
}

func (m IgnoreMatcher) ShouldIgnore(root, path string, d fs.DirEntry) bool {
	name := d.Name()

	if _, exists := m.names[name]; exists {
		return true
	}

	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}

	if _, exists := m.paths[rel]; exists {
		return true
	}

	return false
}
