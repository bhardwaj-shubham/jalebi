package cli

import (
	"errors"
	"flag"
	"io"
	"strings"
)

type AnalysisType string

const (
	TypeAll       AnalysisType = "all"
	TypeFiles     AnalysisType = "files"
	TypeLanguages AnalysisType = "languages"
	TypeTags      AnalysisType = "tags"
)

func (t *AnalysisType) String() string {
	return string(*t)
}

func (t *AnalysisType) Set(value string) error {
	switch AnalysisType(value) {
	case TypeFiles, TypeLanguages, TypeTags:
		*t = AnalysisType(value)
		return nil
	default:
		return errors.New(`invalid value: must be "files", "languages", or "tags"`)
	}
}

type IgnoreList []string

func (i *IgnoreList) String() string {
	return strings.Join(*i, "|")
}

func (i *IgnoreList) Set(value string) error {
	if value == "" {
		return errors.New("ignore list cannot be empty")
	}

	*i = strings.Split(value, "|")
	return nil
}

type Config struct {
	Path        string
	ShowHelp    bool
	ShowVersion bool
	Type        AnalysisType
	IgnoreList  IgnoreList
}

func ParseFlags(args []string, errorWriter io.Writer) (*Config, error) {
	fs := flag.NewFlagSet("jalebi", flag.ContinueOnError)
	fs.SetOutput(errorWriter)

	cfg := &Config{Type: TypeAll}

	fs.BoolVar(&cfg.ShowVersion, "version", false, "Show Jalebi version")
	fs.BoolVar(&cfg.ShowVersion, "v", false, "Show Jalebi version (shorthand)")

	fs.Var(&cfg.Type, "type", "Jalebi analysis type (files, languages, tags)")
	fs.Var(&cfg.IgnoreList, "ignore", "Files or Projects to ignore, separated by '|'")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			cfg.ShowHelp = true
			return cfg, nil
		}

		return nil, err
	}

	positional := fs.Args()

	if cfg.ShowVersion {
		return cfg, nil
	}

	if len(positional) == 0 {
		return nil, errors.New("no project directory provided")
	}

	if len(positional) > 1 {
		return nil, errors.New("too many project directories provided")
	}

	cfg.Path = positional[0]

	return cfg, nil
}
