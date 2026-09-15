package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
		args    []string
		wantErr bool
	}{
		{[]string{"-v"}, false},
		{[]string{"--version"}, false},
		{[]string{"-h"}, false},
		{[]string{"--help"}, false},
		{[]string{"--invalid-flag"}, true},
		{[]string{"--help", tmpDir}, false},
		{[]string{tmpDir, "--help"}, false},
	}

	for _, tc := range testCases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			err := CmdArgs(ctx, tc.args)

			if (err != nil) != tc.wantErr {
				t.Errorf("CmdArgs(%q) error = %v, wantErr %v", tc.args, err, tc.wantErr)
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

	projectInfo, err := AnalyzeProject(ctx, tmpDir)
	if err != nil {
		t.Fatalf("failed to analyze project: %v", err)
	}

	if projectInfo.Directories != 2 || projectInfo.Files != 3 {
		t.Errorf("expected directories=2 & files=3, got directories=%d & files=%d", projectInfo.Directories, projectInfo.Files)
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

	filesCountByLanguage, err := AnalyzeLanguages(ctx, tmpDir)
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

	searchedTags, err := FindTagsInDirectory(ctx, tmpDir)
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

func TestAnalyzeProjectCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tmpDir := t.TempDir()

	_, err := AnalyzeProject(ctx, tmpDir)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
