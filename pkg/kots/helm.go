package kots

import (
	"context"
	_ "embed"
	"io"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots-lint/pkg/domain"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

// GetFilesFromChartReader will render chart templates and return the resulting files
// This function will ignore missing required values.
// This function will also not validate value types.
func GetFilesFromChartReader(ctx context.Context, r io.Reader) (domain.SpecFiles, error) {
	chart, err := loader.LoadArchive(r)
	if err != nil {
		return nil, errors.Wrap(err, "load chart archive")
	}

	options := chartutil.ReleaseOptions{
		Name: "app-chart",
	}

	// If chart has a schema file, it will be used to validate values, which will fail if there are missing required values.
	chart.Schema = nil
	if err := chartutil.ProcessDependencies(chart, chartutil.Values{}); err != nil {
		return nil, errors.Wrap(err, "process dependencies")
	}

	rValues, err := chartutil.ToRenderValues(chart, chart.Values, options, nil)
	if err != nil {
		return nil, errors.Wrap(err, "convert values to render values")
	}

	eng := new(engine.Engine)
	eng.LintMode = true // setting this to true makes `required` and `fail` not fail

	renderedTemplates, err := eng.Render(chart, rValues)
	if err != nil {
		return nil, errors.Wrap(err, "render templates")
	}

	specFiles := domain.SpecFiles{}
	for fileName, fileData := range renderedTemplates {
		if ext := filepath.Ext(fileName); ext != ".yaml" && ext != ".yml" {
			continue
		}

		specFile := domain.SpecFile{
			Name:     fileName,
			Path:     filepath.Dir(fileName),
			Content:  fileData,
			DocIndex: len(specFiles),
		}

		specFiles = append(specFiles, specFile)
	}

	return specFiles, nil
}
