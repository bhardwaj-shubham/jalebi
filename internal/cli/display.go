package cli

import (
	"fmt"
	"io"
)

func DisplayProject(projectName string, info ProjectInfo, w io.Writer) {
	fmt.Fprintln(w, "Project:", projectName)
	fmt.Fprintf(w, "- Directories: %d\n- Files: %d\n",
		info.Directories,
		info.Files,
	)
}

func DisplayLanguages(languages FilesCountByLanguage, w io.Writer) {
	topLanguages := TopLanguages(languages)

	fmt.Fprintln(w, "Languages:")
	for _, file := range topLanguages {
		fmt.Fprintf(w, "• %s = %d\n", file.Language, file.Count)
	}
	fmt.Fprintln(w)
}

func DisplayTags(tags []SearchedTags, w io.Writer) {
	fmt.Fprintln(w, "Developer Notes:")
	if len(tags) == 0 {
		fmt.Fprintln(w, "Nothing found in project!")
		return
	}

	for _, foundTag := range tags {
		fmt.Fprintf(w, "➜ File: %s\n", foundTag.path)
		fmt.Fprintf(w, "  ├─ Line/Col: %d:%d | Tag: [%s]\n",
			foundTag.lineNum,
			foundTag.colNum,
			foundTag.tag,
		)
		fmt.Fprintf(w, "  └─ Context: %s\n\n", foundTag.context)
	}
}

func DisplayAnalysis(
	projectName string,
	result AnalysisResult,
	w io.Writer,
) {

	DisplayProject(projectName, result.ProjectInfo, w)
	DisplayLanguages(result.Languages, w)
	DisplayTags(result.Tags, w)
}
