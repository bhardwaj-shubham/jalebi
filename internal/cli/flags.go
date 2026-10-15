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

type TagList []string

func (t *TagList) String() string {
	return strings.Join(*t, "|")
}

func (t *TagList) Set(value string) error {
	parts := strings.Split(value, "|")
	parsed := make([]string, 0, len(parts))

	validTags := make(map[string]struct{}, len(keywords))
	seen := make(map[string]struct{}, len(parts))

	for _, tag := range keywords {
		validTags[tag] = struct{}{}
	}

	for _, part := range parts {
		tag := strings.ToUpper(strings.TrimSpace(part))
		if tag == "" {
			return errors.New("tag list cannot contain empty tags")
		}

		if _, ok := validTags[tag]; !ok {
			return errors.New(
				"invalid tag: must be one of TODO, FIXME, BUG, HACK, REFACTOR, or NOTE",
			)
		}

		if _, exists := seen[tag]; exists {
			continue
		}

		seen[tag] = struct{}{}
		parsed = append(parsed, tag)
	}

	*t = parsed
	return nil
}

type LanguageList []string

func (l *LanguageList) String() string {
	return strings.Join(*l, "|")
}

func (l *LanguageList) Set(value string) error {
	parts := strings.Split(value, "|")
	parsed := make([]string, 0, len(parts))

	validLanguages := make(map[string]string, len(extToLanguage))
	seen := make(map[string]struct{}, len(parts))

	for _, language := range extToLanguage {
		if language == "" || language == "NO_EXT" {
			continue
		}
		validLanguages[strings.ToLower(language)] = language
	}

	for _, part := range parts {
		name := strings.ToLower(strings.TrimSpace(part))
		if name == "" {
			return errors.New("language list cannot contain empty languages")
		}

		language, ok := validLanguages[name]
		if !ok {
			return errors.New(
				"invalid language: must be one of Go, Python, TypeScript, JavaScript, or Markdown",
			)
		}

		key := strings.ToLower(language)
		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = struct{}{}
		parsed = append(parsed, language)
	}

	*l = parsed
	return nil
}

type Config struct {
	Path        string
	ShowHelp    bool
	ShowVersion bool
	Type        AnalysisType
	IgnoreList  IgnoreList
	Limit       int
	Tags        TagList
	Languages   LanguageList
}

func ParseFlags(args []string, errorWriter io.Writer) (*Config, error) {
	fs := flag.NewFlagSet("jalebi", flag.ContinueOnError)
	fs.SetOutput(errorWriter)

	cfg := &Config{Type: TypeAll}

	fs.BoolVar(&cfg.ShowVersion, "version", false, "Show Jalebi version")
	fs.BoolVar(&cfg.ShowVersion, "v", false, "Show Jalebi version (shorthand)")

	fs.Var(&cfg.Type, "type", "Jalebi analysis type (files, languages, tags)")
	fs.Var(&cfg.IgnoreList, "ignore", "Files or Projects to ignore, separated by '|'")
	fs.Var(&cfg.Tags, "tag", "Tags to scan, separated by '|' (TODO, FIXME, BUG, HACK, REFACTOR, NOTE)")
	fs.Var(&cfg.Languages, "lang", "Languages to scan for tags, separated by '|' (go, python, typescript, javascript, markdown")

	fs.IntVar(&cfg.Limit, "limit", 0, "Maximum number of tag occurrences to scan (0 = unlimited)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			cfg.ShowHelp = true
			return cfg, nil
		}

		return nil, err
	}

	if cfg.Limit < 0 {
		return nil, errors.New("limit cannot be negative")
	}

	if cfg.ShowVersion {
		return cfg, nil
	}

	positional := fs.Args()

	if len(positional) == 0 {
		return nil, errors.New("no project directory provided")
	}

	if len(positional) > 1 {
		return nil, errors.New("too many project directories provided")
	}

	cfg.Path = positional[0]

	return cfg, nil
}
