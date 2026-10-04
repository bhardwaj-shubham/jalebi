package cli

import (
	"bytes"
	"reflect"
	"testing"
)

func TestDisplayAnalysis(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		result      AnalysisResult
		want        string
	}{
		{
			name:        "full analysis",
			projectName: "test-project",
			result: AnalysisResult{
				ProjectInfo: ProjectInfo{
					Directories: 2,
					Files:       5,
				},
				Languages: FilesCountByLanguage{
					"Go":         3,
					"TypeScript": 2,
				},
				Tags: []SearchedTags{
					{
						path:    "main.go",
						lineNum: 3,
						colNum:  4,
						tag:     "TODO",
						context: "// TODO: improve this",
					},
				},
			},
			want: "Project: test-project\n" +
				"- Directories: 2\n" +
				"- Files: 5\n" +
				"Languages:\n" +
				"• Go = 3\n" +
				"• TypeScript = 2\n\n" +
				"Developer Notes:\n" +
				"➜ File: main.go\n" +
				"  ├─ Line/Col: 3:4 | Tag: [TODO]\n" +
				"  └─ Context: // TODO: improve this\n\n",
		},
		{
			name:        "no tags",
			projectName: "empty-project",
			result: AnalysisResult{
				ProjectInfo: ProjectInfo{
					Directories: 1,
					Files:       1,
				},
				Languages: FilesCountByLanguage{
					"Go": 1,
				},
				Tags: []SearchedTags{},
			},
			want: "Project: empty-project\n" +
				"- Directories: 1\n" +
				"- Files: 1\n" +
				"Languages:\n" +
				"• Go = 1\n\n" +
				"Developer Notes:\n" +
				"Nothing found in project!\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w bytes.Buffer

			DisplayAnalysis(tt.projectName, tt.result, &w)

			if got := w.String(); got != tt.want {
				t.Errorf("DisplayAnalysis() output mismatch:\n got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestDisplaySkippedLineWarnings(t *testing.T) {
	var buf bytes.Buffer

	skippedLines := []SkippedLine{
		{
			Path:    "project/main.go",
			LineNum: 10,
		},
		{
			Path:    "project/generated.go",
			LineNum: 25,
		},
	}

	DisplaySkippedLineWarnings(skippedLines, &buf)

	want := "" +
		"warning: skipped oversized line in project/main.go at line 10 (maximum line size: 1048576 bytes)\n" +
		"warning: skipped oversized line in project/generated.go at line 25 (maximum line size: 1048576 bytes)\n"

	if got := buf.String(); got != want {
		t.Fatalf("unexpected warnings:\n%s", got)
	}
}

func TestDisplaySkippedLineWarnings_Empty(t *testing.T) {
	var buf bytes.Buffer

	DisplaySkippedLineWarnings(nil, &buf)

	if got := buf.String(); got != "" {
		t.Fatalf("expected no warning, got %q", got)
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
