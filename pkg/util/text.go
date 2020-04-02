package util

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/pkg/errors"
)

func TryGetLineNumberFromValue(value string) (int, error) {
	r, err := regexp.Compile("line ([^:]*)")
	if err != nil {
		return -1, errors.Wrap(err, "failed to compile regex")
	}

	matches := r.FindStringSubmatch(value)

	if len(matches) < 2 {
		return -1, nil
	}

	line, err := strconv.Atoi(matches[1])
	if err != nil {
		return -1, errors.Wrap(err, "failed to get line number as integer")
	}

	return line, nil
}

// GetLineNumberFromMatch returns the line number for a given substring in "content"
func GetLineNumberFromMatch(content string, match string, docIndex int) (int, error) {
	if content == "" {
		return -1, errors.New("content is empty")
	}

	if match == "" {
		return -1, errors.New("match is empty")
	}

	if docIndex < 0 {
		return -1, errors.New("document index can not be negative")
	}

	docLineNum, err := GetLineNumberForDoc(content, docIndex)
	if err != nil {
		return -1, errors.Wrap(err, "failed to get line number for doc")
	}

	lines := strings.Split(content, "\n")

	for index := docLineNum - 1; index < len(lines); index++ {
		line := lines[index]
		if IsLineEmpty(line) {
			continue
		}
		unquotedLine := strings.Replace(line, "\"", "", -1)
		unquotedLine = strings.Replace(unquotedLine, "'", "", -1)
		if strings.Contains(line, match) || strings.Contains(unquotedLine, match) {
			return index + 1, nil
		}
	}

	return -1, nil
}

// GetLineNumberFromYamlPath returns the line number in a yaml text given the yaml path
// pass 0 as docIndex in case of a single yaml document
func GetLineNumberFromYamlPath(content string, path string, docIndex int) (int, error) {
	if content == "" {
		return -1, errors.New("content is empty")
	}

	if path == "" {
		return -1, errors.New("yaml path is empty")
	}

	if docIndex < 0 {
		return -1, errors.New("document index can not be negative")
	}

	docLineNum, err := GetLineNumberForDoc(content, docIndex)
	if err != nil {
		return -1, errors.Wrap(err, "failed to get line number for doc")
	}

	parts := strings.Split(path, ".")

	// global variables
	isPartFound := false
	currentPartIndex := 0
	lines := strings.Split(content, "\n")

	// line variables
	currentLine := -1
	indentation := ""

	// array variables
	currentArrayIndex := -1

	for index := docLineNum - 1; index < len(lines); index++ {
		line := lines[index]
		if IsLineEmpty(line) {
			continue
		}

		isArray := false
		arrayIndexToFind, err := strconv.Atoi(parts[currentPartIndex])
		if err == nil {
			isArray = true
		}

		if isArray {
			prefixToFind := indentation + "-"
			if strings.HasPrefix(line, prefixToFind) {
				currentArrayIndex++
				if currentArrayIndex == arrayIndexToFind {
					// check next part
					currentPartIndex++
					if currentPartIndex < len(parts) {
						_, err := strconv.Atoi(parts[currentPartIndex])
						if err != nil {
							// next part is not an array, check if the key is on the first line of the array
							nextPartPrefix := parts[currentPartIndex] + ":"
							textToCheck := strings.TrimLeft(line, "\t -")
							if strings.HasPrefix(textToCheck, nextPartPrefix) {
								currentPartIndex++
							}
						}
					}

					isPartFound = true
					currentLine = index + 1
					currentArrayIndex = -1
				}
			}
		} else {
			currentPrefix := indentation + parts[currentPartIndex] + ":"
			if strings.HasPrefix(line, currentPrefix) {
				isPartFound = true
				currentLine = index + 1
				currentPartIndex++
			}
		}

		// break if there is no next part
		if currentPartIndex > len(parts)-1 {
			break
		}

		if !isPartFound {
			continue
		}

		// find next indentation starting from next line
		for i := index + 1; i < len(lines); i++ {
			nextLine := lines[i]
			if IsLineEmpty(nextLine) {
				continue
			}
			indentation = GetLineIndentation(nextLine)
			break
		}

		isPartFound = false
	}

	return currentLine, nil
}

// GetLineNumberForDoc returns the line number of the first line of a document (disregards empty lines and comments)
func GetLineNumberForDoc(content string, docIndex int) (int, error) {
	if content == "" {
		return -1, errors.New("content is empty")
	}

	if docIndex < 0 {
		return -1, errors.New("document index can not be negative")
	}

	foundFirstDoc := false
	currentDocIndex := 0
	lines := strings.Split(content, "\n")

	for index, line := range lines {
		if IsLineEmpty(line) {
			continue
		}

		if strings.HasPrefix(line, "---") {
			if foundFirstDoc {
				currentDocIndex++
			}
			continue
		} else {
			foundFirstDoc = true
		}

		if currentDocIndex < docIndex {
			continue
		}

		return index + 1, nil
	}

	return -1, nil
}

func IsLineEmpty(line string) bool {
	trimmedLine := strings.TrimSpace(line)
	isComment := strings.HasPrefix(trimmedLine, "#")
	return trimmedLine == "" || isComment
}

func GetLineIndentation(line string) string {
	indentation := ""
	runes := []rune(line)
	for _, r := range runes {
		if !unicode.IsSpace(r) {
			break
		}
		indentation += string(r)
	}
	return indentation
}
