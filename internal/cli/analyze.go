package cli

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

var extToLanguage = map[string]string{
	"[no_extension]": "NO_EXT",
	"go":             "Go",
	"py":             "Python",
	"ts":             "TypeScript",
	"js":             "JavaScript",
	"md":             "Markdown",
}

var keywords = []string{
	"TODO", "FIXME", "BUG", "HACK", "REFACTOR", "NOTE",
}

var kwBytes = initKeywordBytes()

var allowedExtensions = map[string]struct{}{
	"go":   {},
	"js":   {},
	"mjs":  {},
	"cjs":  {},
	"ts":   {},
	"mts":  {},
	"cts":  {},
	"py":   {},
	"sh":   {},
	"bash": {},
	"yaml": {},
	"yml":  {},
	"toml": {},
	"md":   {},
	"sql":  {},
}

var defaultSkippedDirs = map[string]struct{}{
	".git":         {},
	".svn":         {},
	".hg":          {},
	"node_modules": {},
	".next":        {},
	".nuxt":        {},
	".astro":       {},
	".svelte-kit":  {},
	"__pycache__":  {},
	".venv":        {},
	"venv":         {},
	".tox":         {},
	"vendor":       {},
	"target":       {},
	"bin":          {},
	"obj":          {},
	"build":        {},
	"dist":         {},
	".idea":        {},
	".vscode":      {},
	".vs":          {},
	"Pods":         {},
	".bundle":      {},
}

var allowedFilenames = map[string]struct{}{
	"Dockerfile": {},
	"Makefile":   {},
}

var skippedFilenameSuffixes = []string{
	".min.js",
	".min.css",
}

func initKeywordBytes() [][]byte {
	result := make([][]byte, len(keywords))
	for i, keyword := range keywords {
		result[i] = []byte(keyword)
	}
	return result
}

type AnalysisResult struct {
	ProjectInfo  ProjectInfo
	Languages    FilesCountByLanguage
	Tags         []SearchedTags
	SkippedLines []SkippedLine
}

func Analyze(
	ctx context.Context,
	root string,
	matcher IgnoreMatcher,
	analysisType AnalysisType,
) (AnalysisResult, error) {
	return AnalyzeWithLimit(ctx, root, matcher, analysisType, 0)
}

func AnalyzeWithLimit(
	ctx context.Context,
	root string,
	matcher IgnoreMatcher,
	analysisType AnalysisType,
	limit int,
) (AnalysisResult, error) {
	projectInfo := ProjectInfo{}
	filesCountByLanguage := make(FilesCountByLanguage)

	searchedTags := make([]SearchedTags, 0)
	skippedLines := make([]SkippedLine, 0)

	runFiles := analysisType == TypeAll || analysisType == TypeFiles
	runLanguages := analysisType == TypeAll || analysisType == TypeLanguages
	runTags := analysisType == TypeAll || analysisType == TypeTags

	tagComplete := false

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if path == root {
			return nil
		}

		// Preserve built-in exclusions.
		if d.IsDir() {
			if _, exists := defaultSkippedDirs[d.Name()]; exists {
				return fs.SkipDir
			}
		}

		// Preserve --ignore behavior.
		if matcher.ShouldIgnore(root, path, d) {
			if d.IsDir() {
				return fs.SkipDir
			}

			return nil
		}

		// Directory analysis.
		if d.IsDir() {
			if runFiles {
				projectInfo.Directories++
			}
			return nil
		}

		// File analysis.
		if runFiles {
			projectInfo.Files++
		}

		// Language analysis.
		if runLanguages {
			ext := filepath.Ext(path)

			if ext == "" {
				ext = "[no_extension]"
			} else {
				ext = strings.ToLower(ext[1:])
			}

			language := extToLanguage[ext]

			if language == "" {
				filesCountByLanguage["Others"]++
			} else {
				filesCountByLanguage[language]++
			}
		}

		// Tag analysis.
		if runTags && !tagComplete {
			remaining := 0
			if limit > 0 {
				remaining = limit - len(searchedTags)
			}

			scanResult, err := findTagsInFileWithLimit(
				ctx,
				path,
				d,
				keywords,
				kwBytes,
				allowedExtensions,
				allowedFilenames,
				remaining,
			)
			if err != nil {
				return err
			}

			searchedTags = append(searchedTags, scanResult.Tags...)
			skippedLines = append(skippedLines, scanResult.SkippedLines...)

			if limit > 0 && len(searchedTags) >= limit {
				tagComplete = true

				if analysisType == TypeTags {
					return fs.SkipAll
				}
			}
		}

		return nil
	})

	if err != nil {
		return AnalysisResult{}, fmt.Errorf("analysis project: %w", err)
	}

	return AnalysisResult{
		ProjectInfo:  projectInfo,
		Languages:    filesCountByLanguage,
		Tags:         searchedTags,
		SkippedLines: skippedLines,
	}, nil
}
