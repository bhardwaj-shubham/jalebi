package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestAnalyze_TypeFiles(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	dirs := []string{"src", "empty", "bin", ".git", "node_modules"}

	for _, dir := range dirs {
		dirPath := filepath.Join(tmpDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
	}

	files := map[string]string{
		".env":                 "PORT=8080\n",
		"hello world.txt":      "Hello, World!",
		"src/main.go":          "package main\n\nfunc main()	{}\n",
		"bin/ignored":          "binary artifact",
		".git/ignored":         "git metadata",
		"node_modules/ignored": "module cache",
	}

	for relPath, content := range files {
		fullPath := filepath.Join(tmpDir, relPath)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write file %s: %v", relPath, err)
		}
	}

	matcher := NewIgnoreMatcher(nil)

	result, err := Analyze(ctx, tmpDir, matcher, TypeFiles)
	projectInfo := result.ProjectInfo

	if err != nil {
		t.Fatalf("failed to analyze project: %v", err)
	}

	if projectInfo.Directories != 2 || projectInfo.Files != 3 {
		t.Errorf("expected directories=2 & files=3, got directories=%d & files=%d", projectInfo.Directories, projectInfo.Files)
	}
}

func TestAnalyze_WithIgnore(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	dirs := []string{
		"src",
		"src/generated",
		"empty",
		"bin",
		".git",
		"node_modules",
		"vendor",
		"build",
	}

	for _, dir := range dirs {
		dirPath := filepath.Join(tmpDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
	}

	files := map[string]string{
		".env":                       "PORT=8080\n",
		"hello world.txt":            "Hello, World!",
		"main.go":                    "package main\n\nfunc main()	{}\n",
		"src/app.go":                 "package src\n",
		"src/generated/code.go":      "package generated",
		"bin/ignored":                "binary artifact",
		".git/ignored":               "git metadata",
		".git/config":                "TODO: should be skipped\n",
		"node_modules/ignored":       "module cache",
		"node_modules/dependency.js": "TODO: should be skipped\n",
		"vendor/lib.go":              "TODO: should be skipped\n",
		"build/output.go":            "TODO: should be skipped\n",
	}

	for relPath, content := range files {
		fullPath := filepath.Join(tmpDir, relPath)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write file %s: %v", relPath, err)
		}
	}

	matcher := NewIgnoreMatcher([]string{
		".env", "hello world.txt", "src/generated",
	})

	result, err := Analyze(ctx, tmpDir, matcher, TypeAll)
	if err != nil {
		t.Fatalf("failed to analyze project: %v", err)
	}

	projectInfo := result.ProjectInfo

	if projectInfo.Directories != 2 {
		t.Errorf("Directories = %d, want 2", projectInfo.Directories)
	}

	if projectInfo.Files != 2 {
		t.Errorf("Files: %d, want 2", projectInfo.Files)
	}

	if len(result.Tags) != 0 {
		t.Errorf("Tags = %d, want no tags", len(result.Tags))
	}
}

func TestAnalyze_TypeLanguages(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	targets := []string{
		"sample.go",
		"sample.py",
		"sample.js",
		"sample.ts",
		"Dockerfile",
		"Makefile",
		".env",
		".gitignore",
		"sample.xyz",
	}
	expectedOutput := map[string]int{
		"Go":         1,
		"Python":     1,
		"JavaScript": 1,
		"TypeScript": 1,
		"NO_EXT":     2,
		"Others":     3,
	}

	for _, name := range targets {
		filePath := filepath.Join(tmpDir, name)

		err := os.WriteFile(filePath, []byte{}, 0644)
		if err != nil {
			t.Fatalf("failed to create file %s: %v", name, err)
		}
	}

	matcher := NewIgnoreMatcher(nil)

	result, err := Analyze(ctx, tmpDir, matcher, TypeLanguages)
	filesCountByLanguage := result.Languages

	if err != nil {
		t.Errorf("failed to analyze languages: %v", err)
	}

	for language, expected := range expectedOutput {
		actual := filesCountByLanguage[language]

		if actual != expected {
			t.Errorf("AnalyzeLanguages: for %s expected %d, get %d", language, expected, actual)
		}
	}
}

func TestAnalyze_TypeLanguages_WithIgnore(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	targets := []string{
		"sample.go",
		"sample.py",
		"sample.js",
		"sample.ts",
		"Dockerfile",
		"Makefile",
		".env",
		".gitignore",
		"sample.xyz",
	}
	expectedOutput := map[string]int{
		"Go":         1,
		"Python":     1,
		"JavaScript": 1,
		"TypeScript": 1,
		"NO_EXT":     2,
		"Others":     1,
	}

	for _, name := range targets {
		filePath := filepath.Join(tmpDir, name)

		err := os.WriteFile(filePath, []byte{}, 0644)
		if err != nil {
			t.Fatalf("failed to create file %s: %v", name, err)
		}
	}

	matcher := NewIgnoreMatcher([]string{".env", ".gitignore"})

	result, err := Analyze(ctx, tmpDir, matcher, TypeLanguages)
	filesCountByLanguage := result.Languages

	if err != nil {
		t.Fatalf("failed to analyze languages: %v", err)
	}

	for language, expected := range expectedOutput {
		actual := filesCountByLanguage[language]

		if actual != expected {
			t.Errorf("AnalyzeLanguages: for %s expected %d, get %d", language, expected, actual)
		}
	}
}

func TestAnalyze_IsolateEachType(t *testing.T) {
	root := t.TempDir()

	createBenchmarkProject(t, root)

	matcher := NewIgnoreMatcher(nil)
	ctx := context.Background()

	wantProject, err := Analyze(ctx, root, matcher, TypeFiles)
	if err != nil {
		t.Fatal(err)
	}

	wantLanguages, err := Analyze(ctx, root, matcher, TypeLanguages)
	if err != nil {
		t.Fatal(err)
	}

	wantTags, err := Analyze(ctx, root, matcher, TypeTags)
	if err != nil {
		t.Fatal(err)
	}

	got, err := Analyze(ctx, root, matcher, TypeAll)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got.ProjectInfo, wantProject.ProjectInfo) {
		t.Fatalf("project mismatch:\n got: %#v\nwant: %#v", got.ProjectInfo, wantProject.ProjectInfo)
	}

	if !reflect.DeepEqual(got.Languages, wantLanguages.Languages) {
		t.Fatalf("languages mismatch:\n got: %#v\nwant: %#v", got.Languages, wantLanguages.Languages)
	}

	if !reflect.DeepEqual(got.Tags, wantTags.Tags) {
		t.Fatalf("tags mismatch:\n got: %#v\nwant: %#v", got.Tags, wantTags.Tags)
	}
}

func TestAnalyze_OversizedLines(t *testing.T) {
	tmpDir := t.TempDir()

	files := map[string]string{
		"before.go": "//TODO: before\n",
		"middle.go": strings.Join([]string{
			"//FIXME: before oversized line",
			strings.Repeat("x", tagScanMaxLineSize+1),
			"//BUG: after oversized line",
		}, "\n"),
		"after.go":     "//NOTE: after oversized file\n",
		"oversized.go": "TODO" + strings.Repeat("x", tagScanMaxLineSize) + "\n",
		"oversized-final.go": "// HACK: before\n" +
			strings.Repeat("x", tagScanMaxLineSize+1),
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)

		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	result, err := Analyze(context.Background(), tmpDir, NewIgnoreMatcher(nil), TypeTags)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if len(result.Tags) != 5 {
		t.Fatalf("got %d tags, want 5", len(result.Tags))
	}

	if len(result.SkippedLines) != 3 {
		t.Fatalf("got %d skipped lines, want 3", len(result.SkippedLines))
	}

	skipped := make(map[string]int)

	for _, line := range result.SkippedLines {
		skipped[filepath.Base(line.Path)] = line.LineNum
	}

	wantSkipped := map[string]int{
		"middle.go":          2,
		"oversized.go":       1,
		"oversized-final.go": 2,
	}

	for file, wantLine := range wantSkipped {
		gotLine, ok := skipped[file]
		if !ok {
			t.Errorf("missing skipped file %q", file)
			continue
		}

		if gotLine != wantLine {
			t.Errorf("%s skipped line = %d, want %d", file, gotLine, wantLine)
		}
	}
}

func TestAnalyze_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tmpDir := t.TempDir()

	matcher := NewIgnoreMatcher(nil)

	_, err := Analyze(ctx, tmpDir, matcher, TypeFiles)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestAnalyzeWithLimit_Unlimited(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	content := "// TODO: first\n// FIXME: second\n// NOTE: third\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := AnalyzeWithLimit(ctx, tmpDir, NewIgnoreMatcher(nil), TypeTags, 0)
	if err != nil {
		t.Fatalf("AnalyzeWithLimit returned error: %v", err)
	}

	if got, want := len(result.Tags), 3; got != want {
		t.Fatalf("got %d tags, want %d", got, want)
	}
}

func TestAnalyzeWithLimit_TypeTags(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	content := "// TODO: first\n// FIXME: second\n// NOTE: third\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := AnalyzeWithLimit(ctx, tmpDir, NewIgnoreMatcher(nil), TypeTags, 2)
	if err != nil {
		t.Fatalf("AnalyzeWithLimit returned error: %v", err)
	}

	if got, want := len(result.Tags), 2; got != want {
		t.Fatalf("got %d tags, want %d", got, want)
	}

	if result.Tags[0].tag != "TODO" || result.Tags[1].tag != "FIXME" {
		t.Fatalf("unexpected tags: got %q and %q", result.Tags[0].tag, result.Tags[1].tag)
	}
}

func TestAnalyzeWithLimit_TypeAllPreservesProjectInfo(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	files := map[string]string{
		"main.go": "// TODO: first\n// FIXME: second\n",
		"app.js":  "// NOTE: third\n",
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	result, err := AnalyzeWithLimit(ctx, tmpDir, NewIgnoreMatcher(nil), TypeAll, 1)
	if err != nil {
		t.Fatalf("AnalyzeWithLimit returned error: %v", err)
	}

	if got, want := len(result.Tags), 1; got != want {
		t.Fatalf("got %d tags, want %d", got, want)
	}

	if got, want := result.ProjectInfo.Files, 2; got != want {
		t.Errorf("got %d files, want %d", got, want)
	}

	if len(result.Languages) == 0 {
		t.Error("expected language statistics to be populated")
	}
}

func TestAnalyzeWithFilter_SingleTag(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	content := "// TODO: first\n// FIXME: second\n// BUG: third\n// NOTE: fourth\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := AnalyzeWithFilter(
		ctx, tmpDir, NewIgnoreMatcher(nil), TypeTags, 0, []string{"TODO"},
	)
	if err != nil {
		t.Fatalf("AnalyzeWithFilter returned error: %v", err)
	}

	if got, want := len(result.Tags), 1; got != want {
		t.Fatalf("got %d tags, want %d", got, want)
	}

	if got, want := result.Tags[0].tag, "TODO"; got != want {
		t.Errorf("got tag %q, want %q", got, want)
	}
}

func TestAnalyzeWithFilter_MultipleTags(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	content := "// TODO: first\n// FIXME: second\n// BUG: third\n// NOTE: fourth\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := AnalyzeWithFilter(
		ctx, tmpDir, NewIgnoreMatcher(nil), TypeTags, 0, []string{"TODO", "BUG"},
	)
	if err != nil {
		t.Fatalf("AnalyzeWithFilter returned error: %v", err)
	}

	if got, want := len(result.Tags), 2; got != want {
		t.Fatalf("got %d tags, want %d", got, want)
	}

	found := make(map[string]int)
	for _, tag := range result.Tags {
		found[tag.tag]++
	}

	for _, tag := range []string{"TODO", "BUG"} {
		if found[tag] != 1 {
			t.Errorf("tag %q occurred %d times, want 1", tag, found[tag])
		}
	}

	for _, tag := range []string{"FIXME", "NOTE"} {
		if found[tag] != 0 {
			t.Errorf("unexpected tag %q in filtered results", tag)
		}
	}
}

func TestAnalyzeWithFilter_NilAndEmptyDefaultToAllTags(t *testing.T) {
	for _, tags := range [][]string{nil, {}} {
		t.Run(fmt.Sprintf("tags=%v", tags), func(t *testing.T) {
			ctx := context.Background()
			tmpDir := t.TempDir()

			content := "// TODO: first\n// FIXME: second\n// BUG: third\n// HACK: fourth\n// REFACTOR: fifth\n// NOTE: sixth\n"
			if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(content), 0644); err != nil {
				t.Fatal(err)
			}

			result, err := AnalyzeWithFilter(
				ctx, tmpDir, NewIgnoreMatcher(nil), TypeTags, 0, tags,
			)
			if err != nil {
				t.Fatalf("AnalyzeWithFilter returned error: %v", err)
			}

			if got, want := len(result.Tags), 6; got != want {
				t.Fatalf("got %d tags, want %d", got, want)
			}
		})
	}
}

func TestAnalyzeWithFilter_FilterBeforeLimit(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	content := "// FIXME: first\n// TODO: first\n// BUG: third\n// TODO: second\n// TODO: third\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := AnalyzeWithFilter(
		ctx, tmpDir, NewIgnoreMatcher(nil), TypeTags, 2, []string{"TODO"},
	)
	if err != nil {
		t.Fatalf("AnalyzeWithFilter returned error: %v", err)
	}

	if got, want := len(result.Tags), 2; got != want {
		t.Fatalf("got %d tags, want %d", got, want)
	}

	for _, tag := range result.Tags {
		if tag.tag != "TODO" {
			t.Errorf("got tag %q, want only TODO", tag.tag)
		}
	}
}

func TestAnalyzeWithFilter_TypeAllPreservesProjectInfo(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	files := map[string]string{
		"main.go":   "// TODO: first\n// FIXME: second\n",
		"app.js":    "// BUG: third\n// TODO: fourth\n",
		"readme.md": "# Project\n",
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	result, err := AnalyzeWithFilter(
		ctx, tmpDir, NewIgnoreMatcher(nil), TypeAll, 1, []string{"TODO"},
	)
	if err != nil {
		t.Fatalf("AnalyzeWithFilter returned error: %v", err)
	}

	if got, want := len(result.Tags), 1; got != want {
		t.Fatalf("got %d tags, want %d", got, want)
	}

	if result.Tags[0].tag != "TODO" {
		t.Errorf("got tag %q, want TODO", result.Tags[0].tag)
	}

	if got, want := result.ProjectInfo.Files, 3; got != want {
		t.Errorf("got %d files, want %d", got, want)
	}

	if result.Languages["Go"] != 1 {
		t.Errorf("Go files = %d, want 1", result.Languages["Go"])
	}
	if result.Languages["JavaScript"] != 1 {
		t.Errorf("JavaScript files = %d, want 1", result.Languages["JavaScript"])
	}
	if result.Languages["Markdown"] != 1 {
		t.Errorf("Markdown files = %d, want 1", result.Languages["Markdown"])
	}
}

func TestAnalyzeWithFilter_Deduplication(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	content := "// TODO: first\n// BUG: second\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// TagList.Set is responsible for deduplication
	var tags TagList
	if err := tags.Set("TODO|TODO|BUG"); err != nil {
		t.Fatalf("TagList.Set returned error: %v", err)
	}

	result, err := AnalyzeWithFilter(
		ctx, tmpDir, NewIgnoreMatcher(nil), TypeTags, 0, tags,
	)
	if err != nil {
		t.Fatalf("AnalyzeWithFilter returned error: %v", err)
	}

	if got, want := len(result.Tags), 2; got != want {
		t.Fatalf("got %d tags, want %d: %#v", got, want, result.Tags)
	}

	found := make(map[string]int)
	for _, tag := range result.Tags {
		found[tag.tag]++
	}

	for _, tag := range []string{"TODO", "BUG"} {
		if found[tag] != 1 {
			t.Errorf("tag %q occurred %d times, want 1", tag, found[tag])
		}
	}
}

func TestAnalyzeWithFilters_LanguageFilter(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	files := map[string]string{
		"main.go":   "// TODO: Go task\n// FIXME: Go issue\n",
		"app.py":    "# TODO: Python task\n",
		"app.js":    "// BUG: JavaScript issue\n",
		"readme.md": "<!-- TODO: Markdown task -->\n",
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	result, err := AnalyzeWithFilters(
		ctx, tmpDir, NewIgnoreMatcher(nil), TypeAll, 0, nil, []string{"Go"},
	)
	if err != nil {
		t.Fatalf("AnalyzeWithFilters returned error: %v", err)
	}

	if got, want := len(result.Tags), 2; got != want {
		t.Fatalf("got %d tags, want %d", got, want)
	}

	for _, tag := range result.Tags {
		if filepath.Base(tag.path) != "main.go" {
			t.Errorf("unexpected tag from %q", tag.path)
		}
	}

	if got, want := result.ProjectInfo.Files, 4; got != want {
		t.Errorf("file count = %d, want %d", got, want)
	}

	if result.Languages["Go"] != 1 ||
		result.Languages["Python"] != 1 ||
		result.Languages["JavaScript"] != 1 ||
		result.Languages["Markdown"] != 1 {
		t.Errorf("language statistics changed unexpectedly: %#v", result.Languages)
	}
}

func TestAnalyzeWithFilters_TagAndLanguageFilters(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	files := map[string]string{
		"main.go": "// TODO: Go task\n// FIXME: Go issue\n",
		"app.py":  "# TODO: Python task\n",
		"app.js":  "// TODO: JavaScript task\n",
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	result, err := AnalyzeWithFilters(
		ctx, tmpDir, NewIgnoreMatcher(nil), TypeTags, 0,
		[]string{"TODO"}, []string{"Go", "Python"},
	)
	if err != nil {
		t.Fatalf("AnalyzeWithFilters returned error: %v", err)
	}

	if got, want := len(result.Tags), 2; got != want {
		t.Fatalf("got %d tags, want %d", got, want)
	}

	for _, tag := range result.Tags {
		if tag.tag != "TODO" {
			t.Errorf("got tag %q, want TODO", tag.tag)
		}
		if filepath.Base(tag.path) == "app.js" {
			t.Errorf("JavaScript tag should have been excluded")
		}
	}
}

func TestAnalyzeWithFilters_LimitAppliesAfterFilters(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	files := map[string]string{
		"main.go": "// FIXME: Go issue\n// TODO: Go task 1\n// TODO: Go task 2\n",
		"app.py":  "# TODO: Python task\n",
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	result, err := AnalyzeWithFilters(
		ctx, tmpDir, NewIgnoreMatcher(nil), TypeTags, 1,
		[]string{"TODO"}, []string{"Go"},
	)
	if err != nil {
		t.Fatalf("AnalyzeWithFilters returned error: %v", err)
	}

	if got, want := len(result.Tags), 1; got != want {
		t.Fatalf("got %d tags, want %d", got, want)
	}

	if result.Tags[0].tag != "TODO" {
		t.Errorf("got tag %q, want TODO", result.Tags[0].tag)
	}
}
