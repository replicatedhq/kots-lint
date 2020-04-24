package kots

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots-lint/pkg/util"
)

type SpecFiles []SpecFile

type SpecFile struct {
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	Content  string    `json:"content"`
	DocIndex int       `json:"docIndex,omitempty"`
	Children SpecFiles `json:"children"`
}

type GVKDoc struct {
	Kind       string      `yaml:"kind" json:"kind" validate:"required"`
	APIVersion string      `yaml:"apiVersion" json:"apiVersion"`
	Metadata   GVKMetadata `yaml:"metadata" json:"metadata"`
}

type GVKMetadata struct {
	Name      string `yaml:"name" json:"name"`
	Namespace string `yaml:"namespace" json:"namespace"`
}

func (f SpecFile) isTarGz() bool {
	return strings.HasSuffix(f.Path, ".tgz") || strings.HasSuffix(f.Path, ".tar.gz")
}

func (f SpecFile) isYAML() bool {
	return strings.HasSuffix(f.Path, ".yaml") || strings.HasSuffix(f.Path, ".yml")
}

func (f SpecFile) hasContent() bool {
	return util.CleanYaml(f.Content) != ""
}

func (fs SpecFiles) unnest() SpecFiles {
	unnestedFiles := SpecFiles{}
	for _, file := range fs {
		if len(file.Children) > 0 {
			unnestedFiles = append(unnestedFiles, file.Children.unnest()...)
		} else {
			unnestedFiles = append(unnestedFiles, file)
		}
	}
	return unnestedFiles
}

func (fs SpecFiles) getFile(path string) (*SpecFile, error) {
	for _, file := range fs {
		if file.Path == path {
			return &file, nil
		}
	}
	return nil, fmt.Errorf("spec file not found for path %s", path)
}

func (fs SpecFiles) separate() (SpecFiles, error) {
	separatedSpecFiles := SpecFiles{}

	for _, file := range fs {
		if !file.isYAML() {
			separatedSpecFiles = append(separatedSpecFiles, file)
			continue
		}

		cleanedContent := util.CleanYaml(file.Content)
		docs := strings.Split(cleanedContent, "\n---\n")

		for index, doc := range docs {
			if strings.HasPrefix(doc, "---\n") {
				doc = doc[4:]
			}

			if len(doc) == 0 {
				continue
			}

			separatedSpecFile := SpecFile{
				Name:     file.Name,
				Path:     file.Path, // keep original path to be able to link it back
				Content:  doc,
				DocIndex: index,
			}

			separatedSpecFiles = append(separatedSpecFiles, separatedSpecFile)
		}
	}

	return separatedSpecFiles, nil
}

func SpecFilesFromTarFile(tarFile []byte) (SpecFiles, error) {
	specFiles := SpecFiles{}

	strReader := bytes.NewReader(tarFile)
	tr := tar.NewReader(strReader)

	for {
		header, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		var data bytes.Buffer
		_, err = io.Copy(&data, tr)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get data for %s", header.Name)
		}

		specFile := SpecFile{
			Name:    header.FileInfo().Name(),
			Path:    header.Name,
			Content: data.String(),
		}

		specFiles = append(specFiles, specFile)
	}

	return specFiles, nil
}
