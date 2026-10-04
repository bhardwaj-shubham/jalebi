package cli

import (
	"io"
	"reflect"
	"testing"
)

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
