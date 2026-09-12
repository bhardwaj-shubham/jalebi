package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCmdArgs_NoArgs(t *testing.T) {
	err := CmdArgs([]string{})
	if err == nil || err.Error() != "no project directory provided" {
		t.Errorf("expected 'no project directory provided' error, got %v", err)
	}
}

func TestCmdArgs_Flags(t *testing.T) {
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
			err := CmdArgs(tc.args)

			if (err != nil) != tc.wantErr {
				t.Errorf("CmdArgs(%q) error = %v, wantErr %v", tc.args, err, tc.wantErr)
			}
		})
	}
}

func TestCmdArgs_NonExistentPath(t *testing.T) {
	err := CmdArgs([]string{"/path/does/not/exist/1234"})

	if err == nil {
		t.Fatal("expected error for non-existent path, got nil")
	}
}

func TestCmdArgs_PathIsFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "testFile")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	err = CmdArgs([]string{tmpFile.Name()})
	if err == nil {
		t.Error("expected error when path is a file instead of a directory, got nil")
	}
}

func TestCmdArgs_ValidDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	err := CmdArgs([]string{tmpDir})
	if err != nil {
		t.Errorf("expected no error for valid directory, got %v", err)
	}
}

func TestCmdArgs_AnalyzeProject(t *testing.T) {
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

	projectInfo, err := AnalyzeProject(tmpDir)
	if err != nil {
		t.Errorf("failed to analyze project: %v", err)
	}

	if projectInfo.Directories != 2 && projectInfo.Files != 3 {
		t.Errorf("expected directories=3 & files=2, got directories=%d & files=%d", projectInfo.Directories, projectInfo.Files)
	}
}
