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

func TestParseFlagsTagFilter(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "single tag",
			args: []string{"--tag", "TODO", "."},
			want: "TODO",
		},
		{
			name: "multiple tags",
			args: []string{"--tag", "TODO|FIXME|BUG", "."},
			want: "TODO|FIXME|BUG",
		},
		{
			name: "case insensitive with whitespace",
			args: []string{"--tag", " todo |fixme ", "."},
			want: "TODO|FIXME",
		},
		{
			name: "duplicate tags are deduplicated",
			args: []string{"--tag", "TODO|BUG|TODO|BUG", "."},
			want: "TODO|BUG",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseFlags(tt.args, io.Discard)
			if err != nil {
				t.Fatalf("ParseFlags() error = %v", err)
			}

			if got := cfg.Tags.String(); got != tt.want {
				t.Errorf("Tags = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseFlagsInvalidTagFilter(t *testing.T) {
	tests := []struct {
		name string
		tag  string
	}{
		{name: "unknown tag", tag: "UNKNOWN"},
		{name: "empty tag", tag: ""},
		{name: "empty entry", tag: "TODO||BUG"},
		{name: "trailing separator", tag: "TODO|"},
		{name: "leading separator", tag: "|TODO"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFlags(
				[]string{"--tag", tt.tag, "."},
				io.Discard,
			)
			if err == nil {
				t.Fatal("ParseFlags() expected an error for invalid tag filter")
			}
		})
	}
}

func TestParseFlagsLanguageFilter(t *testing.T) {
	tests := []struct {
		name string
		lang string
		want string
	}{
		{name: "single language", lang: "go", want: "Go"},
		{name: "multiple languages", lang: "go|python", want: "Go|Python"},
		{name: "case insensitive with whitespace", lang: "go |PYTHON", want: "Go|Python"},
		{name: "duplicates are deduplicated", lang: "go|python|go", want: "Go|Python"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseFlags([]string{"--lang", tt.lang, "."}, io.Discard)
			if err != nil {
				t.Fatalf("ParseFlags() error = %v", err)
			}

			if got := cfg.Languages.String(); got != tt.want {
				t.Errorf("Languages = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseFlagsInvalidLanguageFilter(t *testing.T) {
	tests := []struct {
		name string
		lang string
	}{
		{name: "unknown language", lang: "rust"},
		{name: "empty language", lang: ""},
		{name: "empty entry", lang: "go||python"},
		{name: "trailing seperator", lang: "go|"},
		{name: "leading seperator", lang: "|go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFlags([]string{"--lang", tt.lang, "."}, io.Discard)
			if err == nil {
				t.Fatal("ParseFlags() expected an error for invalid language filter")
			}
		})
	}
}
