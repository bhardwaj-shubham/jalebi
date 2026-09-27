package cli

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var version = "dev"

func CmdArgs(ctx context.Context, args []string) error {
	cfg, err := ParseFlags(args, os.Stderr)
	if err != nil {
		return err
	}

	if cfg.ShowHelp {
		return nil
	}

	if cfg.ShowVersion {
		fmt.Println("Jalebi:", version)
		return nil
	}

	info, err := os.Stat(cfg.Path)
	if err != nil {
		return fmt.Errorf("cannot access path %q: %w", cfg.Path, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %q", cfg.Path)
	}

	matcher := NewIgnoreMatcher(cfg.IgnoreList)

	var projectInfo ProjectInfo
	if cfg.Type == TypeAll || cfg.Type == TypeFiles {
		projectInfo, err = AnalyzeProject(ctx, cfg.Path, matcher)
		if err != nil {
			return err
		}

		fmt.Println("Project:", info.Name())
		fmt.Printf("- Directories: %d\n- Files: %d\n", projectInfo.Directories, projectInfo.Files)
	}

	var filesCountByLanguage FilesCountByLanguage
	var topLanguages []LanguageCount
	if cfg.Type == TypeAll || cfg.Type == TypeLanguages {
		filesCountByLanguage, err = AnalyzeLanguages(ctx, cfg.Path, matcher)
		if err != nil {
			return err
		}
		topLanguages = TopLanguages(filesCountByLanguage)

		fmt.Println("Languages:")
		for _, file := range topLanguages {
			fmt.Printf("• %s = %d\n", file.Language, file.Count)
		}
		fmt.Println()
	}

	var searchedTags []SearchedTags
	if cfg.Type == TypeAll || cfg.Type == TypeTags {
		searchedTags, err = FindTagsInDirectory(ctx, cfg.Path, matcher)
		if err != nil {
			return err
		}

		fmt.Println("Developer Notes:")
		if len(searchedTags) == 0 {
			fmt.Println("Nothing found in project!")
		} else {
			for _, foundTags := range searchedTags {
				fmt.Printf("➜ File: %s\n", foundTags.path)
				fmt.Printf("  ├─ Line/Col: %d:%d | Tag: [%s]\n", foundTags.lineNum, foundTags.colNum, foundTags.tag)
				fmt.Printf("  └─ Context: %s\n\n", foundTags.context)
			}
		}
	}

	return nil
}

type ProjectInfo struct {
	Files       int
	Directories int
}

type FilesCountByLanguage map[string]int

func AnalyzeProject(ctx context.Context, root string, matcher IgnoreMatcher) (ProjectInfo, error) {
	files, directories := 0, 0

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

		if d.IsDir() && (d.Name() == "bin" || d.Name() == ".git" || d.Name() == "node_modules") {
			return fs.SkipDir
		}

		if matcher.ShouldIgnore(root, path, d) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			directories++
		} else {
			files++
		}

		return nil
	})

	if err != nil {
		return ProjectInfo{}, err
	}

	return ProjectInfo{
		Files:       files,
		Directories: directories,
	}, nil
}

func AnalyzeLanguages(ctx context.Context, root string, matcher IgnoreMatcher) (FilesCountByLanguage, error) {
	filesCountByLanguage := make(FilesCountByLanguage)

	extToLanguage := map[string]string{
		"[no_extension]": "NO_EXT",
		"go":             "Go",
		"py":             "Python",
		"ts":             "TypeScript",
		"js":             "JavaScript",
		"md":             "Markdown",
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

		if d.IsDir() && (d.Name() == "bin" || d.Name() == ".git" || d.Name() == "node_modules") {
			return fs.SkipDir
		}

		if matcher.ShouldIgnore(root, path, d) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		if !d.IsDir() {
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

		return nil
	})

	if err != nil {
		return nil, err
	}

	return filesCountByLanguage, nil
}

type LanguageCount struct {
	Language string
	Count    int
}

func TopLanguages(filesCountByLanguage FilesCountByLanguage) []LanguageCount {
	topLanguages := make([]LanguageCount, 0)

	for language, count := range filesCountByLanguage {
		if language == "Others" || language == "NO_EXT" {
			continue
		}
		topLanguages = append(topLanguages, LanguageCount{Language: language, Count: count})
	}

	slices.SortFunc(topLanguages, func(pair1 LanguageCount, pair2 LanguageCount) int {
		if c := cmp.Compare(pair2.Count, pair1.Count); c != 0 {
			return c
		}
		return cmp.Compare(pair1.Language, pair2.Language)
	})

	if len(topLanguages) > 3 {
		topLanguages = topLanguages[:3]
	}

	return topLanguages
}

type SearchedTags struct {
	path    string
	lineNum int
	colNum  int
	tag     string
	context string
}

func isWordChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

func FindTagsInDirectory(ctx context.Context, root string, matcher IgnoreMatcher) ([]SearchedTags, error) {
	searchedTags := make([]SearchedTags, 0)

	// Pre-define keywords as byte slices
	keywords := []string{"TODO", "FIXME", "BUG", "HACK", "REFACTOR", "NOTE"}
	kwBytes := make([][]byte, len(keywords))
	for i, kw := range keywords {
		kwBytes[i] = []byte(kw)
	}

	allowedExtensions := map[string]struct{}{
		"go": {}, "py": {}, "ts": {}, "js": {}, "md": {},
	}
	allowedFilenames := map[string]struct{}{
		"Dockerfile": {}, "Makefile": {},
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

		if d.IsDir() && (d.Name() == "bin" || d.Name() == ".git" || d.Name() == "node_modules") {
			return fs.SkipDir
		}

		if matcher.ShouldIgnore(root, path, d) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		if !d.IsDir() {
			ext := filepath.Ext(path)

			if ext != "" {
				ext = strings.ToLower(ext[1:])

				if _, exists := allowedExtensions[ext]; !exists {
					return nil
				}
			} else {
				if _, exists := allowedFilenames[d.Name()]; !exists {
					return nil
				}
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)
			buf := make([]byte, 64*1024)
			scanner.Buffer(buf, 1024*1024)

			lineNum := 0

			for scanner.Scan() {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				lineNum++
				rawLine := scanner.Bytes() // Reuses Scanner's internal buffer

				var contextStr string // Lazily generated only if a match is found
				hasMatchOnLine := false

				// Check each keyword on the line
				for i, kwb := range kwBytes {
					kw := keywords[i]
					currIdx := 0

					for {
						idx := bytes.Index(rawLine[currIdx:], kwb)
						if idx == -1 {
							break
						}

						absoluteIdx := currIdx + idx

						// Emulate \b (word boundary) check
						leftValid := absoluteIdx == 0 || !isWordChar(rawLine[absoluteIdx-1])
						rightIdx := absoluteIdx + len(kwb)
						rightValid := rightIdx == len(rawLine) || !isWordChar(rawLine[rightIdx])

						if leftValid && rightValid {
							// Generate context string only on the first actual match for this line
							if !hasMatchOnLine {
								contextStr = strings.TrimSpace(string(rawLine))
								hasMatchOnLine = true
							}

							colNum := absoluteIdx + 1

							searchedTags = append(searchedTags, SearchedTags{
								path:    path,
								lineNum: lineNum,
								colNum:  colNum,
								tag:     kw,
								context: contextStr,
							})

						}

						currIdx = absoluteIdx + len(kwb)
					}
				}
			}
			return scanner.Err()
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return searchedTags, nil
}
