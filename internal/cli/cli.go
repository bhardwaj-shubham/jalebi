package cli

import (
	"bufio"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

func CmdArgs(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("no project directory provided")
	}

	for _, arg := range args {
		if strings.HasPrefix(arg, "--") || arg == "-v" || arg == "-h" {
			return flags(arg)
		}
	}

	path := args[0]

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("cannot access path %q: %w", path, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %q", path)
	}

	start := time.Now()

	projectInfo, err := AnalyzeProject(ctx, path)
	if err != nil {
		return err
	}

	fmt.Printf("DEBUG: AnalyzeProject: %v\n", time.Since(start))

	start = time.Now()

	filesCountByLanguage, err := AnalyzeLanguages(ctx, path)
	if err != nil {
		return err
	}

	fmt.Printf("DEBUG: AnalyzeLanguages: %v\n", time.Since(start))

	start = time.Now()

	topLanguages := TopLanguages(filesCountByLanguage)

	fmt.Printf("DEBUG: TopLanguages: %v\n", time.Since(start))

	start = time.Now()

	searchedTags, err := FindTagsInDirectory(ctx, path)
	if err != nil {
		return err
	}

	fmt.Printf("DEBUG: FindTagsInDirectory: %v\n", time.Since(start))

	fmt.Println("Project:", info.Name())
	fmt.Printf("- Directories: %d\n- Files: %d\n", projectInfo.Directories, projectInfo.Files)

	fmt.Println("\nLanguages:")
	for _, file := range topLanguages {
		fmt.Printf("• %s = %d\n", file.Language, file.Count)
	}
	fmt.Println()

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

	return nil
}

func flags(flag string) error {
	switch flag {
	case "--version", "-v":
		fmt.Println("Jalebi: 0.3.0")
		return nil

	case "--help", "-h":
		fmt.Println("Jalebi is made for developer to understand project.\nJalebi CLI Version: 0.3.0")
		return nil

	default:
		return fmt.Errorf("unknown flag: %s", flag)
	}
}

type ProjectInfo struct {
	Files       int
	Directories int
}

type FilesCountByLanguage map[string]int

func AnalyzeProject(ctx context.Context, root string) (ProjectInfo, error) {
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

func AnalyzeLanguages(ctx context.Context, root string) (FilesCountByLanguage, error) {
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

func FindTagsInDirectory(ctx context.Context, rootPath string) ([]SearchedTags, error) {
	searchedTags := make([]SearchedTags, 0)

	re := regexp.MustCompile(`\b(TODO|FIXME|BUG|HACK|REFACTOR|NOTE)\b`)
	allowedExtensions := map[string]struct{}{
		"go": {},
		"py": {},
		"ts": {},
		"js": {},
		"md": {},
	}
	allowedFilenames := map[string]struct{}{
		"Dockerfile": {},
		"Makefile":   {},
	}

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
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
				line := scanner.Text()

				matches := re.FindAllStringIndex(line, -1)

				for _, loc := range matches {
					match := line[loc[0]:loc[1]]
					colNum := loc[0] + 1

					searchedTags = append(searchedTags, SearchedTags{path: path, lineNum: lineNum, colNum: colNum, tag: match, context: strings.TrimSpace(line)})
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
