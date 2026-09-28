package cli

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"slices"
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
	writer := os.Stdout

	result, err := Analyze(ctx, cfg.Path, matcher, cfg.Type)
	if err != nil {
		return err
	}

	switch cfg.Type {
	case TypeAll:
		DisplayAnalysis(info.Name(), result, writer)

	case TypeFiles:
		DisplayProject(info.Name(), result.ProjectInfo, writer)

	case TypeLanguages:
		DisplayLanguages(result.Languages, writer)

	case TypeTags:
		DisplayTags(result.Tags, writer)
	}

	return nil
}

type ProjectInfo struct {
	Files       int
	Directories int
}

type FilesCountByLanguage map[string]int

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
