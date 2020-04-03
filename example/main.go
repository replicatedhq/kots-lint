package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedcom/saaskit/log"
	"github.com/replicatedhq/kots-lint/pkg/kots"
)

func main() {
	files, err := readFiles("example/files-to-lint")
	if err != nil {
		log.Errorf("failed to read files: %v", err)
		os.Exit(1)
	}

	result, err := lintFiles(files)
	if err != nil {
		log.Errorf("failed to lint files: %v", err)
		os.Exit(1)
	}

	prettified, err := prettyJson(result)
	if err != nil {
		log.Errorf("failed to prettify result: %v", err)
		os.Exit(1)
	}

	fmt.Println(prettified)
}

func prettyJson(value string) (string, error) {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, []byte(value), "", "  ")
	if err != nil {
		return "", errors.Wrap(err, "failed to json indent")
	}
	return prettyJSON.String(), nil
}

func lintFiles(specFiles *kots.SpecFiles) (string, error) {
	spec, err := json.Marshal(specFiles)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal files")
	}

	requestBody, err := json.Marshal(map[string]string{
		"spec": string(spec),
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to create request body")
	}

	url := "http://localhost:30082/v1/lint"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to call the lint service, please make sure the service is running")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read response body")
	}

	return string(body), nil
}

func readFiles(dir string) (*kots.SpecFiles, error) {
	specFiles := kots.SpecFiles{}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		content, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "failed to read file %s", path)
		}

		relativePath, err := filepath.Rel(dir, path)
		if err != nil {
			return errors.Wrapf(err, "failed to get relative path for %s", path)
		}

		specFile := kots.SpecFile{
			Name:    info.Name(),
			Path:    relativePath,
			Content: string(content),
		}

		specFiles = append(specFiles, specFile)
		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to walk dir %s", dir)
	}

	return &specFiles, nil
}
