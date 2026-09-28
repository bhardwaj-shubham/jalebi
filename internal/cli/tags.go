package cli

import (
	"bufio"
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

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

func findTagsInFile(
	ctx context.Context,
	path string,
	d fs.DirEntry,
	keywords []string,
	kwBytes [][]byte,
	allowedExtensions map[string]struct{},
	allowedFilenames map[string]struct{},
) ([]SearchedTags, error) {
	ext := filepath.Ext(path)

	if ext != "" {
		ext = strings.ToLower(ext[1:])

		if _, exists := allowedExtensions[ext]; !exists {
			return nil, nil
		}
	} else {
		if _, exists := allowedFilenames[d.Name()]; !exists {
			return nil, nil
		}
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	searchedTags := make([]SearchedTags, 0)

	scanner := bufio.NewScanner(file)

	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lineNum := 0

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		lineNum++

		rawLine := scanner.Bytes()

		var contextStr string
		hasMatchOnLine := false

		for i, kwb := range kwBytes {
			kw := keywords[i]
			currIdx := 0

			for {
				idx := bytes.Index(rawLine[currIdx:], kwb)
				if idx == -1 {
					break
				}

				absoluteIdx := currIdx + idx

				leftValid := absoluteIdx == 0 ||
					!isWordChar(rawLine[absoluteIdx-1])

				rightIdx := absoluteIdx + len(kwb)
				rightValid := rightIdx == len(rawLine) ||
					!isWordChar(rawLine[rightIdx])

				if leftValid && rightValid {
					if !hasMatchOnLine {
						contextStr = strings.TrimSpace(string(rawLine))
						hasMatchOnLine = true
					}

					searchedTags = append(searchedTags, SearchedTags{
						path:    path,
						lineNum: lineNum,
						colNum:  absoluteIdx + 1,
						tag:     kw,
						context: contextStr,
					})
				}

				currIdx = absoluteIdx + len(kwb)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return searchedTags, nil
}
