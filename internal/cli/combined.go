package cli

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

type AnalysisResult struct {
	ProjectInfo ProjectInfo
	Languages   FilesCountByLanguage
	Tags        []SearchedTags
}

func Analyze(
	ctx context.Context,
	root string,
	matcher IgnoreMatcher,
	analysisType AnalysisType,
) (AnalysisResult, error) {
	projectInfo := ProjectInfo{}
	filesCountByLanguage := make(FilesCountByLanguage)
	searchedTags := make([]SearchedTags, 0)

	runFiles := analysisType == TypeAll || analysisType == TypeFiles
	runLanguages := analysisType == TypeAll || analysisType == TypeLanguages
	runTags := analysisType == TypeAll || analysisType == TypeTags

	extToLanguage := map[string]string{
		"[no_extension]": "NO_EXT",
		"go":             "Go",
		"py":             "Python",
		"ts":             "TypeScript",
		"js":             "JavaScript",
		"md":             "Markdown",
	}

	keywords := []string{"TODO", "FIXME", "BUG", "HACK", "REFACTOR", "NOTE"}

	kwBytes := make([][]byte, len(keywords))
	for i, kw := range keywords {
		kwBytes[i] = []byte(kw)
	}

	allowedExtensions := map[string]struct{}{
		"go": {}, "py": {}, "ts": {}, "js": {}, "md": {},
	}

	allowedFilenames := map[string]struct{}{
		"Dockerfile": {},
		"Makefile":   {},
	}

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
		if d.IsDir() &&
			(d.Name() == "bin" ||
				d.Name() == ".git" ||
				d.Name() == "node_modules") {
			return fs.SkipDir
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
		if runTags {
			tags, err := findTagsInFile(
				ctx,
				path,
				d,
				keywords,
				kwBytes,
				allowedExtensions,
				allowedFilenames,
			)
			if err != nil {
				return err
			}

			searchedTags = append(searchedTags, tags...)
		}

		return nil
	})

	if err != nil {
		return AnalysisResult{}, fmt.Errorf("analysis project: %w", err)
	}

	return AnalysisResult{
		ProjectInfo: projectInfo,
		Languages:   filesCountByLanguage,
		Tags:        searchedTags,
	}, nil
}
