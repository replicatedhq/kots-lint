package kots

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
)

func GetFilesFromChartReader(r io.Reader) (SpecFiles, error) {
	cfg := &action.Configuration{
		// Log: nil,
	}
	client := action.NewInstall(cfg)
	client.DryRun = true
	client.ReleaseName = "release-name"
	client.Replace = true
	client.ClientOnly = true
	client.IncludeCRDs = true
	client.Namespace = "default"

	chart, err := loader.LoadArchive(r)
	if err != nil {
		return nil, errors.Wrap(err, "load chart archive")
	}

	rel, err := client.Run(chart, nil)
	if err != nil {
		return nil, errors.Wrap(err, "helm template")
	}

	var manifests bytes.Buffer
	fmt.Fprintln(&manifests, strings.TrimSpace(rel.Manifest))

	docs := strings.Split(manifests.String(), "\n---\n")
	specFiles := SpecFiles{}
	for index, doc := range docs {
		doc = strings.TrimPrefix(doc, "---\n")

		fileName := getFileNameFromDoc(doc)
		if fileName == "" {
			fileName = fmt.Sprintf("doc[%d]", index)
		}

		separatedSpecFile := SpecFile{
			Name:     fileName,
			Path:     "",
			Content:  doc,
			DocIndex: index,
		}

		specFiles = append(specFiles, separatedSpecFile)
	}

	return specFiles, nil
}

// Get file name from the input doc. File name is the string after "# Source: " in the first line of the doc.
func getFileNameFromDoc(doc string) string {
	lines := strings.SplitN(doc, "\n", 2)
	if len(lines) > 1 && strings.HasPrefix(lines[0], "# Source: ") {
		return strings.TrimPrefix(lines[0], "# Source: ")
	}
	return ""
}
