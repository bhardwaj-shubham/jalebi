package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func CmdArgs(args []string) error {
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

	fmt.Println("Project:", info.Name())

	projectInfo, err := AnalyzeProject(path)
	if err != nil {
		return err
	}

	filesCountByLanguage, err := AnalyzeLanguages(path)
	if err != nil {
		return err
	}

	fmt.Printf("Directories: %d\nFiles: %d\n", projectInfo.Directories, projectInfo.Files)

	fmt.Println("\n---Files Counts---")
	for fileType, cnt := range filesCountByLanguage {
		fmt.Printf("%s = %d\n", fileType, cnt)
	}
	fmt.Println()

	return nil
}

func flags(flag string) error {
	switch flag {
	case "--version", "-v":
		fmt.Println("Jalebi: 0.2.0")
		return nil

	case "--help", "-h":
		fmt.Println("Jalebi is made for developer to understand project.\nJalebi CLI Version: 0.2.0")
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

func AnalyzeProject(root string) (ProjectInfo, error) {
	files, directories := 0, 0

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
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

func AnalyzeLanguages(root string) (FilesCountByLanguage, error) {
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
