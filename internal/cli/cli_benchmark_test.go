package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkThreeWalkDir(b *testing.B) {
	root := b.TempDir()

	createBenchmarkProject(b, root)

	matcher := NewIgnoreMatcher(nil)
	ctx := context.Background()

	for b.Loop() {
		if _, err := Analyze(ctx, root, matcher, TypeFiles); err != nil {
			b.Fatal(err)
		}

		if _, err := Analyze(ctx, root, matcher, TypeLanguages); err != nil {
			b.Fatal(err)
		}

		if _, err := Analyze(ctx, root, matcher, TypeTags); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkOneWalkDir(b *testing.B) {
	root := b.TempDir()
	createBenchmarkProject(b, root)

	matcher := NewIgnoreMatcher(nil)
	ctx := context.Background()

	for b.Loop() {
		if _, err := Analyze(ctx, root, matcher, TypeAll); err != nil {
			b.Fatal(err)
		}
	}
}

func createBenchmarkProject(t testing.TB, root string) {
	t.Helper()

	files := map[string]string{
		"README.md":            "# Project\n\nTODO: improve documentation\n",
		"Makefile":             "build:\n\tgo build\n",
		"Dockerfile":           "FROM alpine\n",
		".env":                 "PORT=8080\n",
		"cmd/main.go":          "package main\n\n// TODO: improve this\nfunc main() {}\n",
		"internal/app.go":      "package internal\n\n// FIXME: refactor\n",
		"internal/service.py":  "# BUG: temporary implementation\n",
		"frontend/app.ts":      "// NOTE: improve this\n",
		"frontend/app.js":      "// HACK: temporary\n",
		"docs/design.md":       "# Design\n",
		"generated/model.go":   "package generated\n",
		"generated/schema.ts":  "export type ID = string\n",
		"tests/app_test.go":    "package tests\n\n// TODO: add more tests\n",
		"config/config.json":   "{}\n",
		"scripts/build.sh":     "#!/bin/sh\n",
		"data/sample.xyz":      "unknown\n",
		"LICENSE":              "MIT\n",
		"vendor/dependency.go": "package dependency\n",
	}

	for name, content := range files {
		path := filepath.Join(root, name)

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}
