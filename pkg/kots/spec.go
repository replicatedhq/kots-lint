package kots

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path"
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
	return util.CleanUpYaml(f.Content) != ""
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

		cleanedContent := util.CleanUpYaml(file.Content)
		docs := strings.Split(cleanedContent, "\n---\n")

		for index, doc := range docs {
			doc = strings.TrimPrefix(doc, "---")
			doc = strings.TrimLeft(doc, "\n")

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

func SpecFilesFromTar(reader io.Reader) (SpecFiles, error) {
	specFiles := SpecFiles{}

	tr := tar.NewReader(reader)

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

func SpecFilesFromTarGz(tarGz SpecFile) (SpecFiles, error) {
	content, err := base64.StdEncoding.DecodeString(tarGz.Content)
	if err != nil {
		// tarGz content is not base64 encoded, read as bytes
		content = []byte(tarGz.Content)
	}

	gzf, err := gzip.NewReader(bytes.NewReader(content))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create gzip reader")
	}

	files, err := SpecFilesFromTar(gzf)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read chart archive")
	}

	// remove any common prefix from all files
	if len(files) > 0 {
		firstFileDir, _ := path.Split(files[0].Path)
		commonPrefix := strings.Split(firstFileDir, string(os.PathSeparator))

		for _, file := range files {
			d, _ := path.Split(file.Path)
			dirs := strings.Split(d, string(os.PathSeparator))
			commonPrefix = util.CommonSlicePrefix(commonPrefix, dirs)
		}

		cleanedFiles := SpecFiles{}
		for _, file := range files {
			d, f := path.Split(file.Path)
			d2 := strings.Split(d, string(os.PathSeparator))

			cleanedFile := file
			d2 = d2[len(commonPrefix):]
			cleanedFile.Path = path.Join(path.Join(d2...), f)

			cleanedFiles = append(cleanedFiles, cleanedFile)
		}

		files = cleanedFiles
	}

	return files, nil
}
