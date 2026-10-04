package cli

import (
	"bufio"
	"bytes"
	"context"
	"io"
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

type SkippedLine struct {
	Path    string
	LineNum int
}

type TagScanResult struct {
	Tags         []SearchedTags
	SkippedLines []SkippedLine
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
) (TagScanResult, error) {

	filename := strings.ToLower(d.Name())

	for _, suffix := range skippedFilenameSuffixes {
		if strings.HasSuffix(filename, suffix) {
			return TagScanResult{}, nil
		}
	}

	ext := filepath.Ext(path)

	if ext != "" {
		ext = strings.ToLower(ext[1:])

		if _, exists := allowedExtensions[ext]; !exists {
			return TagScanResult{}, nil
		}
	} else {
		if _, exists := allowedFilenames[d.Name()]; !exists {
			return TagScanResult{}, nil
		}
	}

	file, err := os.Open(path)
	if err != nil {
		return TagScanResult{}, err
	}
	defer file.Close()

	searchedTags := make([]SearchedTags, 0)
	skippedLines := make([]SkippedLine, 0)

	reader := bufio.NewReaderSize(file, tagScanBufferSize)

	lineNum := 0

	for {
		result, err := readLine(ctx, reader)
		if err != nil {
			if err == io.EOF {
				break
			}

			return TagScanResult{}, err
		}

		select {
		case <-ctx.Done():
			return TagScanResult{}, ctx.Err()
		default:
		}

		lineNum++

		if result.oversized {
			skippedLines = append(skippedLines, SkippedLine{
				Path:    path,
				LineNum: lineNum,
			})
			continue
		}

		rawLine := result.line

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

	return TagScanResult{
		Tags:         searchedTags,
		SkippedLines: skippedLines,
	}, nil
}

const (
	tagScanBufferSize  = 64 * 1024
	tagScanMaxLineSize = 1024 * 1024
)

type readLineResult struct {
	line      []byte
	oversized bool
}

func readLine(ctx context.Context, reader *bufio.Reader) (readLineResult, error) {
	var line []byte
	lineSize := 0
	oversized := false

	for {

		select {
		case <-ctx.Done():
			return readLineResult{}, ctx.Err()
		default:
		}

		fragment, isPrefix, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				if lineSize == 0 {
					return readLineResult{}, io.EOF
				}

				return readLineResult{
					line:      line,
					oversized: oversized,
				}, nil
			}

			return readLineResult{}, err
		}

		if !isPrefix && line == nil {
			if len(fragment) > tagScanMaxLineSize {
				return readLineResult{
					oversized: true,
				}, nil
			}

			return readLineResult{
				line: fragment,
			}, nil
		}

		if !oversized {
			lineSize += len(fragment)

			if lineSize > tagScanMaxLineSize {
				oversized = true
				line = nil
			} else {
				// Allocate enough capacity for the first fragment.
				if line == nil {
					line = make([]byte, 0, tagScanMaxLineSize)
				}

				line = append(line, fragment...)
			}
		}

		if !isPrefix {
			return readLineResult{
				line:      line,
				oversized: oversized,
			}, nil
		}
	}
}
