package cli

import (
	"context"
	"os"
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
