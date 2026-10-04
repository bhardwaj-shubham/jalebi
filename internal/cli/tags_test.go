package cli

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const testMaxLineSize = 1024 * 1024

func writeTagTestFile(t *testing.T, name, content string) (string, fs.DirEntry) {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("failed to read test directory: %v", err)
	}

	for _, entry := range entries {
		if entry.Name() == name {
			return path, entry
		}
	}

	t.Fatalf("test file entry not found: %s", name)
	return "", nil
}

func findTagsForTest(
	_ *testing.T,
	ctx context.Context,
	path string,
	entry fs.DirEntry,
) (TagScanResult, error) {
	return findTagsInFile(
		ctx,
		path,
		entry,
		keywords,
		kwBytes,
		allowedExtensions,
		allowedFilenames,
	)
}

func TestFindTagsInFile_NormalLine(t *testing.T) {
	path, entry := writeTagTestFile(
		t,
		"sample.go",
		"package main\n// TODO: fix this\n",
	)

	result, err := findTagsForTest(t, context.Background(), path, entry)
	if err != nil {
		t.Fatalf("findTagsInFile() error = %v", err)
	}

	if len(result.Tags) != 1 {
		t.Fatalf("got %d tags, want 1", len(result.Tags))
	}

	if result.Tags[0].lineNum != 2 {
		t.Errorf("lineNum = %d, want 2", result.Tags[0].lineNum)
	}

	if result.Tags[0].colNum != 4 {
		t.Errorf("colNum = %d, want 4", result.Tags[0].colNum)
	}

	if result.Tags[0].tag != "TODO" {
		t.Errorf("tag = %q, want %q", result.Tags[0].tag, "TODO")
	}
}

func TestFindTagsInFile_FinalLineWithoutNewline(t *testing.T) {
	path, entry := writeTagTestFile(
		t,
		"sample.go",
		"package main\n// TODO: final line",
	)

	result, err := findTagsForTest(t, context.Background(), path, entry)
	if err != nil {
		t.Fatalf("findTagsInFile() error = %v", err)
	}

	if len(result.Tags) != 1 {
		t.Fatalf("got %d tags, want 1", len(result.Tags))
	}

	if result.Tags[0].lineNum != 2 {
		t.Errorf("lineNum = %d, want 2", result.Tags[0].lineNum)
	}

	if result.Tags[0].colNum != 4 {
		t.Errorf("colNum = %d, want 4", result.Tags[0].colNum)
	}
}

func TestFindTagsInFile_CRLF(t *testing.T) {
	path, entry := writeTagTestFile(
		t,
		"sample.go",
		"package main\r\n// TODO: CRLF\r\n// FIXME: second\r\n",
	)

	result, err := findTagsForTest(t, context.Background(), path, entry)
	if err != nil {
		t.Fatalf("findTagsInFile() error = %v", err)
	}

	if len(result.Tags) != 2 {
		t.Fatalf("got %d tags, want 2", len(result.Tags))
	}

	if result.Tags[0].lineNum != 2 || result.Tags[0].colNum != 4 {
		t.Errorf("first tag = line %d, col %d; want line 2, col 4",
			result.Tags[0].lineNum, result.Tags[0].colNum)
	}

	if result.Tags[1].lineNum != 3 || result.Tags[1].colNum != 4 {
		t.Errorf("second tag = line %d, col %d; want line 3, col 4",
			result.Tags[1].lineNum, result.Tags[1].colNum)
	}
}

func TestFindTagsInFile_LineSizeBoundary(t *testing.T) {
	tests := []struct {
		name        string
		lineSize    int
		wantTags    int
		wantSkipped int
	}{
		{
			name:        "max minus one",
			lineSize:    testMaxLineSize - 1,
			wantTags:    1,
			wantSkipped: 0,
		},
		{
			name:        "max",
			lineSize:    testMaxLineSize,
			wantTags:    1,
			wantSkipped: 0,
		},
		{
			name:        "max plus one",
			lineSize:    testMaxLineSize + 1,
			wantTags:    0,
			wantSkipped: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The tag is placed at the beginning so the test can verify
			// whether the entire logical line is accepted or rejected.
			line := "TODO " + strings.Repeat("x", tc.lineSize-len("TODO")-1)

			path, entry := writeTagTestFile(
				t,
				"sample.go",
				line+"\n",
			)

			result, err := findTagsForTest(t, context.Background(), path, entry)

			if err != nil {
				t.Fatalf(
					"findTagsInFile() error = %v",
					err,
				)
			}

			if len(result.SkippedLines) != tc.wantSkipped {
				t.Errorf(
					"got %d skipped lines, want %d",
					len(result.SkippedLines),
					tc.wantSkipped,
				)
			}

			if len(result.Tags) != tc.wantTags {
				t.Errorf("got %d tags, want %d", len(result.Tags), tc.wantTags)
			}
		})
	}
}

func TestFindTagsInFile_OversizedLineMiddle(t *testing.T) {
	oversizedLine := strings.Repeat("x", testMaxLineSize+1)

	content := strings.Join([]string{
		"// TODO: before",
		oversizedLine,
		"// FIXME: after",
	}, "\n")

	path, entry := writeTagTestFile(t, "sample.go", content)

	result, err := findTagsForTest(t, context.Background(), path, entry)

	if err != nil {
		t.Fatalf("findTagsInFile() error = %v", err)
	}

	if len(result.SkippedLines) != 1 {
		t.Fatalf(
			"got %d skipped lines, want 1",
			len(result.SkippedLines),
		)
	}

	if result.SkippedLines[0].LineNum != 2 {
		t.Errorf(
			"skipped line = %d, want 2",
			result.SkippedLines[0].LineNum,
		)
	}

	if len(result.Tags) != 2 {
		t.Fatalf("got %d tags, want 2", len(result.Tags))
	}

	if result.Tags[0].tag != "TODO" || result.Tags[0].lineNum != 1 {
		t.Errorf("first tag = %+v, want TODO on line 1", result.Tags[0])
	}

	if result.Tags[1].tag != "FIXME" || result.Tags[1].lineNum != 3 {
		t.Errorf("second tag = %+v, want FIXME on line 3", result.Tags[1])
	}
}

func TestFindTagsInFile_TagInsideOversizedLine(t *testing.T) {
	oversizedLine := "TODO" + strings.Repeat("x", testMaxLineSize)

	path, entry := writeTagTestFile(
		t,
		"sample.go",
		oversizedLine+"\n",
	)

	result, err := findTagsForTest(t, context.Background(), path, entry)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Tags) != 0 {
		t.Fatalf("got %d tags, want 0", len(result.Tags))
	}

	if len(result.SkippedLines) != 1 {
		t.Fatalf("got %d skipped lines, want 1", len(result.SkippedLines))
	}

	if result.SkippedLines[0].LineNum != 1 {
		t.Errorf(
			"skipped line = %d, want 1",
			result.SkippedLines[0].LineNum,
		)
	}
}

func TestFindTagsInFile_FilePolicy(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		wantScanned bool
	}{
		// Allowed extensions
		{"Go", "main.go", true},
		{"JavaScript", "app.js", true},
		{"MJS", "app.mjs", true},
		{"CJS", "app.cjs", true},
		{"TypeScript", "app.ts", true},
		{"MTS", "app.mts", true},
		{"CTS", "app.cts", true},
		{"Python", "app.py", true},
		{"Shell", "build.sh", true},
		{"Bash", "build.bash", true},
		{"YAML", "config.yaml", true},
		{"YML", "config.yml", true},
		{"TOML", "config.toml", true},
		{"Markdown", "README.md", true},
		{"SQL", "schema.sql", true},

		// Special filenames
		{"Dockerfile", "Dockerfile", true},
		{"Makefile", "Makefile", true},

		// Unsupported extensions
		{"JSON", "config.json", false},
		{"XML", "config.xml", false},
		{"Text", "notes.txt", false},
		{"Log", "app.log", false},
		{"Source map", "app.js.map", false},
		{"Dockerfile variant", "Dockerfile.dev", false},

		// Minified assets are skipped case-insensitively
		{"Minified JS", "app.min.js", false},
		{"Uppercase minified JS", "app.MIN.JS", false},
		{"Minified CSS", "app.min.css", false},
		{"Uppercase minified CSS", "app.MIN.CSS", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, entry := writeTagTestFile(
				t,
				tt.filename,
				"TODO: policy test\n",
			)

			result, err := findTagsForTest(
				t,
				context.Background(),
				path,
				entry,
			)
			if err != nil {
				t.Fatalf("findTagsForTest() error = %v", err)
			}

			gotScanned := len(result.Tags) > 0
			if gotScanned != tt.wantScanned {
				t.Errorf(
					"scanned = %v, want %v",
					gotScanned,
					tt.wantScanned,
				)
			}
		})
	}
}

func TestAnalyze_TypeTags(t *testing.T) {
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

func TestAnalyze_TypeTags_WithIgnore(t *testing.T) {
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

func BenchmarkFindTagsInFile(b *testing.B) {
	tests := []struct {
		name    string
		content []byte
	}{
		{
			name: "Normal",
			content: []byte(
				"package main\n" +
					"// TODO: fix this\n" +
					"func main() {}\n",
			),
		}, {
			name: "ManyTags",
			content: bytes.Repeat(
				[]byte("// TODO FIXME BUG HACK REFACTOR NOTE\n"),
				1000,
			),
		}, {
			name: "LongValidLine",
			content: append(
				bytes.Repeat([]byte("a"), testMaxLineSize-20),
				[]byte(" TODO: fix this\n")...,
			),
		}, {
			name: "OversizedLine",
			content: append(
				bytes.Repeat([]byte("a"), testMaxLineSize+1),
				[]byte(" TODO: skipped\n")...,
			),
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			dir := b.TempDir()
			path := filepath.Join(dir, "sample.go")

			if err := os.WriteFile(path, tt.content, 0644); err != nil {
				b.Fatal(err)
			}

			entries, err := os.ReadDir(dir)
			if err != nil {
				b.Fatal(err)
			}

			entry := entries[0]

			b.ReportAllocs()
			b.SetBytes(int64(len(tt.content)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result, err := findTagsInFile(
					context.Background(),
					path,
					entry,
					keywords,
					kwBytes,
					allowedExtensions,
					allowedFilenames,
				)
				if err != nil {
					b.Fatal(err)
				}

				_ = result
			}
		})
	}
}

func BenchmarkReadLine(b *testing.B) {
	tests := []struct {
		name    string
		content []byte
	}{
		{
			name:    "Normal",
			content: []byte("package main\n// TODO: fix this\n"),
		},
		{
			name: "LongValidLine",
			content: append(
				bytes.Repeat([]byte("a"), testMaxLineSize-20),
				[]byte(" TODO: fix this\n")...,
			),
		},
		{
			name: "OversizedLine",
			content: append(
				bytes.Repeat([]byte("a"), testMaxLineSize+1),
				[]byte(" TODO: skipped\n")...,
			),
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(tt.content)))

			for i := 0; i < b.N; i++ {
				reader := bufio.NewReaderSize(
					bytes.NewReader(tt.content),
					tagScanBufferSize,
				)

				for {
					result, err := readLine(context.Background(), reader)
					if err != nil {
						if err == io.EOF {
							break
						}
						b.Fatal(err)
					}

					_ = result
				}
			}
		})
	}
}

func BenchmarkTagSearchLongLine(b *testing.B) {
	line := append(
		bytes.Repeat([]byte("a"), testMaxLineSize-20),
		[]byte(" TODO: fix this\n")...,
	)

	b.ReportAllocs()
	b.SetBytes(int64(len(line)))

	for b.Loop() {
		var contextStr string
		hasMatch := false

		for _, kwb := range kwBytes {
			currIdx := 0

			for {
				idx := bytes.Index(line[currIdx:], kwb)
				if idx == -1 {
					break
				}

				absoluteIdx := currIdx + idx

				leftValid := absoluteIdx == 0 ||
					!isWordChar(line[absoluteIdx-1])

				rightIdx := absoluteIdx + len(kwb)
				rightValid := rightIdx == len(line) ||
					!isWordChar(line[rightIdx])

				if leftValid && rightValid {
					if !hasMatch {
						contextStr = strings.TrimSpace(string(line))
						hasMatch = true
					}
				}

				currIdx = absoluteIdx + len(kwb)
			}
		}

		_ = contextStr
	}
}
