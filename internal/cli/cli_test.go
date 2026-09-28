package cli

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCmdArgs_NoArgs(t *testing.T) {
	ctx := context.Background()

	err := CmdArgs(ctx, []string{})
	if err == nil || err.Error() != "no project directory provided" {
		t.Errorf("expected 'no project directory provided' error, got %v", err)
	}
}

func TestCmdArgs_Flags(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	testCases := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "version shorthand",
			args:    []string{"-v"},
			wantErr: false,
		},
		{
			name:    "version",
			args:    []string{"--version"},
			wantErr: false,
		},
		{
			name:    "help shorthand",
			args:    []string{"-h"},
			wantErr: false,
		},
		{
			name:    "help",
			args:    []string{"-help"},
			wantErr: false,
		},
		{
			name:    "invalid flag",
			args:    []string{"--invalid-flag"},
			wantErr: true,
		},
		{
			name:    "help before path",
			args:    []string{"-help", tmpDir},
			wantErr: false,
		},
		{
			name:    "help after path",
			args:    []string{tmpDir, "-help"},
			wantErr: true,
		}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := CmdArgs(ctx, tc.args)

			if (err != nil) != tc.wantErr {
				t.Errorf("CmdArgs(%q) error = %v, wantErr %v", tc.args, err, tc.wantErr)
			}
		})
	}
}

func TestParseFlags(t *testing.T) {
	testCases := []struct {
		name       string
		args       []string
		wantType   AnalysisType
		wantPath   string
		wantIgnore IgnoreList
		wantErr    bool
	}{
		{
			name:     "defaults",
			args:     []string{"."},
			wantType: TypeAll,
			wantPath: ".",
		},
		{
			name:     "files",
			args:     []string{"--type", "files", "."},
			wantType: TypeFiles,
			wantPath: ".",
		},
		{
			name:     "languages",
			args:     []string{"--type", "languages", "."},
			wantType: TypeLanguages,
			wantPath: ".",
		},
		{
			name:     "tags",
			args:     []string{"--type", "tags", "."},
			wantType: TypeTags,
			wantPath: ".",
		},
		{
			name:    "invalid type",
			args:    []string{"--type", "invalid", "."},
			wantErr: true,
		},
		{
			name:       "ignore",
			args:       []string{"--ignore", "generated|prisma|integrations", "."},
			wantType:   TypeAll,
			wantPath:   ".",
			wantIgnore: []string{"generated", "prisma", "integrations"},
		},
		{
			name:    "missing path",
			args:    []string{"--type", "files"},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := ParseFlags(tc.args, io.Discard)

			if (err != nil) != tc.wantErr {
				t.Fatalf("ParseFlags() error = %v, wantErr %v", err, tc.wantErr)
			}

			if tc.wantErr {
				return
			}

			if cfg.Type != tc.wantType {
				t.Errorf("Type = %q, want %q", cfg.Type, tc.wantType)
			}

			if cfg.Path != tc.wantPath {
				t.Errorf("Path = %q, want %q", cfg.Path, tc.wantPath)
			}

			if tc.wantIgnore != nil {
				if !reflect.DeepEqual(cfg.IgnoreList, tc.wantIgnore) {
					t.Errorf("IgnoreList = %v, want %v", cfg.IgnoreList, tc.wantIgnore)
				}
			}
		})
	}
}

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

func TestCmdArgs_NonExistentPath(t *testing.T) {
	ctx := context.Background()
	err := CmdArgs(ctx, []string{"/path/does/not/exist/1234"})

	if err == nil {
		t.Fatal("expected error for non-existent path, got nil")
	}
}

func TestCmdArgs_PathIsFile(t *testing.T) {
	ctx := context.Background()
	tmpFile, err := os.CreateTemp("", "testFile")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	err = CmdArgs(ctx, []string{tmpFile.Name()})
	if err == nil {
		t.Error("expected error when path is a file instead of a directory, got nil")
	}
}

func TestCmdArgs_ValidDirectory(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	err := CmdArgs(ctx, []string{tmpDir})
	if err != nil {
		t.Errorf("expected no error for valid directory, got %v", err)
	}
}

func TestAnalyzeProject(t *testing.T) {
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

func TestAnalyzeProject_WithIgnore(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	dirs := []string{"src", "src/generated", "empty", "bin", ".git", "node_modules"}

	for _, dir := range dirs {
		dirPath := filepath.Join(tmpDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
	}

	files := map[string]string{
		".env":                  "PORT=8080\n",
		"hello world.txt":       "Hello, World!",
		"src/main.go":           "package main\n\nfunc main()	{}\n",
		"src/generated/code.go": "package generated",
		"bin/ignored":           "binary artifact",
		".git/ignored":          "git metadata",
		"node_modules/ignored":  "module cache",
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

	result, err := Analyze(ctx, tmpDir, matcher, TypeFiles)
	projectInfo := result.ProjectInfo

	if err != nil {
		t.Fatalf("failed to analyze project: %v", err)
	}

	if projectInfo.Directories != 2 || projectInfo.Files != 1 {
		t.Errorf("expected directories=2 & files=1, got directories=%d & files=%d", projectInfo.Directories, projectInfo.Files)
	}
}

func TestAnalyzeLanguages(t *testing.T) {
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

func TestAnalyzeLanguages_WithIgnore(t *testing.T) {
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

func TestAnalyze(t *testing.T) {
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

func TestTopLanguages(t *testing.T) {
	input1 := FilesCountByLanguage{
		"Go":         10,
		"Python":     7,
		"JavaScript": 7,
		"TypeScript": 5,
		"Markdown":   2,
		"Others":     20,
		"NO_EXT":     15,
	}

	input2 := FilesCountByLanguage{
		"Go":     5,
		"Python": 3,
	}

	expected1 := []LanguageCount{
		{Language: "Go", Count: 10},
		{Language: "JavaScript", Count: 7},
		{Language: "Python", Count: 7},
	}

	expected2 := []LanguageCount{
		{Language: "Go", Count: 5},
		{Language: "Python", Count: 3},
	}

	actual1 := TopLanguages(input1)
	actual2 := TopLanguages(input2)

	if !reflect.DeepEqual(actual1, expected1) {
		t.Errorf("for input-1, expected %v, get %v", expected1, actual1)
	}

	if !reflect.DeepEqual(actual2, expected2) {
		t.Errorf("for input-2, expected %v, get %v", expected2, actual2)
	}
}

func TestFindTagsInDirectory(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	dirs := []string{"src", "bin", ".git", "node_modules"}

	for _, dir := range dirs {
		dirPath := filepath.Join(tmpDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
	}

	files := map[string]string{
		".env":                 "//TODO: change port to 3000\nPORT=8080\n",
		"hello-world.txt":      "//REFACTOR: format file \nHello, World!",
		"src/main.go":          "package main\n\n//TODO: add code NOTE: use snippets\nfunc main()	{}\n",
		"work.txt":             "TODOING work",
		"Dockerfile":           "//TODO: add postgres container\n",
		"Makefile":             "//FIXME: fix makefile build command\n",
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

	result, err := Analyze(ctx, tmpDir, matcher, TypeTags)
	searchedTags := result.Tags

	if err != nil {
		t.Errorf("failed to search notes: %v", err)
	}

	expectedSearchedTags := []SearchedTags{
		{
			path:    filepath.Join(tmpDir, "Dockerfile"),
			lineNum: 1,
			colNum:  3,
			tag:     "TODO",
			context: "//TODO: add postgres container",
		},
		{
			path:    filepath.Join(tmpDir, "Makefile"),
			lineNum: 1,
			colNum:  3,
			tag:     "FIXME",
			context: "//FIXME: fix makefile build command",
		},
		{
			path:    filepath.Join(tmpDir, "src", "main.go"),
			lineNum: 3,
			colNum:  3,
			tag:     "TODO",
			context: "//TODO: add code NOTE: use snippets",
		},
		{
			path:    filepath.Join(tmpDir, "src", "main.go"),
			lineNum: 3,
			colNum:  18,
			tag:     "NOTE",
			context: "//TODO: add code NOTE: use snippets",
		},
	}

	if len(searchedTags) != len(expectedSearchedTags) {
		t.Fatalf("expected %d searched notes, got %d",
			len(expectedSearchedTags), len(searchedTags))
	}

	for idx, expectedResult := range expectedSearchedTags {
		actualResult := searchedTags[idx]

		if !reflect.DeepEqual(expectedResult, actualResult) {
			t.Errorf("searched notes: expected: %v, get: %v", expectedResult, actualResult)
		}
	}
}

func TestFindTagsInDirectory_WithIgnore(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	dirs := []string{"src", "test", "bin", ".git", "node_modules"}

	for _, dir := range dirs {
		dirPath := filepath.Join(tmpDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
	}

	files := map[string]string{
		".env":                 "//TODO: change port to 3000\nPORT=8080\n",
		"hello-world.txt":      "//REFACTOR: format file \nHello, World!",
		"src/main.go":          "package main\n\n//TODO: add code NOTE: use snippets\nfunc main()	{}\n",
		"test/main_test.go":    "//TODO: add the test for main function\n",
		"src/secret.txt":       "//FIXME: please add secret here\n",
		"work.txt":             "TODOING work",
		"Dockerfile":           "//TODO: add postgres container\n",
		"Makefile":             "//FIXME: fix makefile build command\n",
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

	matcher := NewIgnoreMatcher([]string{
		"test/", "secret.txt", "Dockerfile", "Makefile",
	})

	result, err := Analyze(ctx, tmpDir, matcher, TypeTags)
	searchedTags := result.Tags

	if err != nil {
		t.Fatalf("failed to search notes: %v", err)
	}

	expectedSearchedTags := []SearchedTags{
		{
			path:    filepath.Join(tmpDir, "src", "main.go"),
			lineNum: 3,
			colNum:  3,
			tag:     "TODO",
			context: "//TODO: add code NOTE: use snippets",
		},
		{
			path:    filepath.Join(tmpDir, "src", "main.go"),
			lineNum: 3,
			colNum:  18,
			tag:     "NOTE",
			context: "//TODO: add code NOTE: use snippets",
		},
	}

	if len(searchedTags) != len(expectedSearchedTags) {
		t.Fatalf("expected %d searched notes, got %d",
			len(expectedSearchedTags), len(searchedTags))
	}

	for idx, expectedResult := range expectedSearchedTags {
		actualResult := searchedTags[idx]

		if !reflect.DeepEqual(expectedResult, actualResult) {
			t.Errorf("searched notes: expected: %v, get: %v", expectedResult, actualResult)
		}
	}
}

func TestAnalyzeProjectCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tmpDir := t.TempDir()

	matcher := NewIgnoreMatcher(nil)

	_, err := Analyze(ctx, tmpDir, matcher, TypeFiles)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
