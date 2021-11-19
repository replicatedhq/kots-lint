package kots

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/instrumenta/kubeval/kubeval"
	"github.com/mitchellh/mapstructure"
	"github.com/open-policy-agent/opa/rego"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots-lint/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"gopkg.in/yaml.v2"
	goyaml "gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chart"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	kotsscheme.AddToScheme(scheme.Scheme)
}

type LintExpression struct {
	Rule      string                       `json:"rule"`
	Type      string                       `json:"type"`
	Message   string                       `json:"message"`
	Path      string                       `json:"path"`
	Positions []LintExpressionItemPosition `json:"positions"`
}

type OPALintExpression struct {
	Rule     string `json:"rule"`
	Type     string `json:"type"`
	Message  string `json:"message"`
	Path     string `json:"path"`
	DocIndex int    `json:"docIndex"`
	Field    string `json:"field"`
	Match    string `json:"match"`
}

type LintExpressionItemPosition struct {
	Start LintExpressionItemLinePosition `json:"start"`
}

type LintExpressionItemLinePosition struct {
	Line int `json:"line"`
}

// a prepared rego query for linting NON-rendered files
var nonRenderedRegoQuery *rego.PreparedEvalQuery

// a prepared rego query for linting RENDERED files
var renderedRegoQuery *rego.PreparedEvalQuery

func InitOPALinting(regoPath string) error {
	ctx := context.Background()

	// prepare rego query for linting non-rendered files
	nonRenderedRegoContent, err := ioutil.ReadFile(fmt.Sprintf("%s/kots-spec-opa-nonrendered.rego", regoPath))
	if err != nil {
		return errors.Wrap(err, "failed to read non-rendered rego file")
	}

	nonRenderedQuery, err := rego.New(
		rego.Query("data.kots.spec.nonrendered.lint"),
		rego.Module("kots-spec-opa-nonrendered.rego", string(nonRenderedRegoContent)),
	).PrepareForEval(ctx)

	if err != nil {
		return errors.Wrap(err, "failed to prepare non-rendered query for eval")
	}

	nonRenderedRegoQuery = &nonRenderedQuery

	// prepare rego query for linting rendered files
	renderedRegoContent, err := ioutil.ReadFile(fmt.Sprintf("%s/kots-spec-opa-rendered.rego", regoPath))
	if err != nil {
		return errors.Wrap(err, "failed to read rendered rego file")
	}

	renderedQuery, err := rego.New(
		rego.Query("data.kots.spec.rendered.lint"),
		rego.Module("kots-spec-opa-rendered.rego", string(renderedRegoContent)),
	).PrepareForEval(ctx)

	if err != nil {
		return errors.Wrap(err, "failed to prepare rendered query for eval")
	}

	renderedRegoQuery = &renderedQuery

	return nil
}

func LintSpecFiles(specFiles SpecFiles) ([]LintExpression, bool, error) {
	unnestedFiles := specFiles.unnest()

	filteredFiles := SpecFiles{}
	for _, file := range unnestedFiles {
		if file.isYAML() {
			filteredFiles = append(filteredFiles, file)
		}
	}

	// if there are yaml errors, end early there
	yamlLintExpressions := lintIsValidYAML(filteredFiles)
	if lintExpressionsHaveErrors(yamlLintExpressions) {
		return yamlLintExpressions, false, nil
	}

	// if helm charts are missing corresponding manifests or vise versa, end early there
	helmChartsLintExpressions, err := lintHelmCharts(unnestedFiles)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to lint helm charts")
	}
	if lintExpressionsHaveErrors(helmChartsLintExpressions) {
		return helmChartsLintExpressions, false, nil
	}

	opaNonRenderedLintExpressions, err := lintWithOPANonRendered(filteredFiles)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to lint with OPA non-rendered")
	}
	// if there are opa NON-rendered errors, end early there
	if lintExpressionsHaveErrors(opaNonRenderedLintExpressions) {
		return opaNonRenderedLintExpressions, false, nil
	}

	renderContentLintExpressions, renderedFiles, err := lintRenderContent(filteredFiles)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to lint render content")
	}
	// if there are render content errors, end early there
	if lintExpressionsHaveErrors(renderContentLintExpressions) {
		return renderContentLintExpressions, false, nil
	}

	opaRenderedLintExpressions, err := lintWithOPARendered(renderedFiles, filteredFiles)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to lint with OPA rendered")
	}
	// if there are opa RENDERED errors, end early there
	if lintExpressionsHaveErrors(opaRenderedLintExpressions) {
		return opaRenderedLintExpressions, false, nil
	}

	kubevalLintExpressions, err := lintWithKubeval(renderedFiles, filteredFiles)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to lint with Kubeval")
	}

	allLintExpressions := []LintExpression{}
	allLintExpressions = append(allLintExpressions, yamlLintExpressions...)
	allLintExpressions = append(allLintExpressions, opaNonRenderedLintExpressions...)
	allLintExpressions = append(allLintExpressions, opaRenderedLintExpressions...)
	allLintExpressions = append(allLintExpressions, renderContentLintExpressions...)
	allLintExpressions = append(allLintExpressions, kubevalLintExpressions...)

	return allLintExpressions, true, nil
}

// InitOPALinting needs to be called first in order for this function to run successfully
// This function will lint using the prepared query for NON-rendered files
func lintWithOPANonRendered(specFiles SpecFiles) ([]LintExpression, error) {
	separatedSpecFiles, err := specFiles.separate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to separate multi docs")
	}

	ctx := context.Background()
	results, err := nonRenderedRegoQuery.Eval(ctx, rego.EvalInput(separatedSpecFiles))
	if err != nil {
		return nil, errors.Wrap(err, "failed to evaluate query")
	}

	return opaResultsToLintExpressions(results, specFiles)
}

// InitOPALinting needs to be called first in order for this function to run successfully
// This function will lint using the prepared query for RENDERED files
// renderedFiles are the rendered files to be linted (we don't render on the fly because it is an expensive process)
// originalFiles are the non-rendered non-separated files, which are needed to find the actual line number
func lintWithOPARendered(renderedFiles SpecFiles, originalFiles SpecFiles) ([]LintExpression, error) {
	ctx := context.Background()
	results, err := renderedRegoQuery.Eval(ctx, rego.EvalInput(renderedFiles))
	if err != nil {
		return nil, errors.Wrap(err, "failed to evaluate query")
	}
	return opaResultsToLintExpressions(results, originalFiles)
}

func opaResultsToLintExpressions(results rego.ResultSet, specFiles SpecFiles) ([]LintExpression, error) {
	lintExpressions := []LintExpression{}

	if len(results) == 0 {
		return lintExpressions, nil
	}

	result := results[0]
	if len(result.Expressions) == 0 {
		return lintExpressions, nil
	}

	var opaLintExpressions []OPALintExpression
	if err := mapstructure.Decode(result.Expressions[0].Value, &opaLintExpressions); err != nil {
		return nil, errors.Wrap(err, "failed to mapstructure opa lint expressions")
	}

	// map opa lint expressions to standard lint expressions
	for _, opaLintExpression := range opaLintExpressions {
		lintExpression := LintExpression{
			Rule:    opaLintExpression.Rule,
			Type:    opaLintExpression.Type,
			Message: opaLintExpression.Message,
		}

		if opaLintExpression.Path == "" {
			lintExpressions = append(lintExpressions, lintExpression)
			continue
		}

		lintExpression.Path = opaLintExpression.Path

		// we need to get the line number for the original file content not the separated document nor the rendered one
		foundSpecFile, err := specFiles.getFile(opaLintExpression.Path)
		if err != nil {
			lintExpressions = append(lintExpressions, lintExpression)
			continue
		}

		line := -1
		if opaLintExpression.Field != "" {
			line, _ = util.GetLineNumberFromYamlPath(foundSpecFile.Content, opaLintExpression.Field, opaLintExpression.DocIndex)
		} else if opaLintExpression.Match != "" {
			line, _ = util.GetLineNumberFromMatch(foundSpecFile.Content, opaLintExpression.Match, opaLintExpression.DocIndex)
		} else if opaLintExpression.Type == "error" {
			line, _ = util.GetLineNumberForDoc(foundSpecFile.Content, opaLintExpression.DocIndex)
		}

		if line == -1 {
			lintExpressions = append(lintExpressions, lintExpression)
			continue
		}

		lintExpression.Positions = []LintExpressionItemPosition{
			{
				Start: LintExpressionItemLinePosition{
					Line: line,
				},
			},
		}

		lintExpressions = append(lintExpressions, lintExpression)
	}

	return lintExpressions, nil
}

// renderedFiles are the rendered files to be linted (we don't render on the fly because it is an expensive process)
// originalFiles are the non-rendered non-separated files, which are needed to find the actual line number
func lintWithKubeval(renderedFiles SpecFiles, originalFiles SpecFiles) ([]LintExpression, error) {
	return lintWithKubevalSchema(renderedFiles, originalFiles, "file://kubernetes-json-schema")
}

// renderedFiles are the rendered files to be linted (we don't render on the fly because it is an expensive process)
// originalFiles are the non-rendered non-separated files, which are needed to find the actual line number
func lintWithKubevalSchema(renderedFiles SpecFiles, originalFiles SpecFiles, schemaLocation string) ([]LintExpression, error) {
	lintExpressions := []LintExpression{}

	kubevalConfig := kubeval.Config{
		SchemaLocation:    schemaLocation,
		Strict:            true,
		KubernetesVersion: "1.17.0",
	}
	for _, renderedFile := range renderedFiles {
		kubevalConfig.FileName = renderedFile.Path
		results, err := kubeval.Validate([]byte(renderedFile.Content), &kubevalConfig)
		if err != nil {
			var lintExpression LintExpression

			if strings.Contains(err.Error(), "Failed initalizing schema") && strings.Contains(err.Error(), "no such file or directory") {
				lintExpression = LintExpression{
					Rule:    "kubeval-schema-not-found",
					Type:    "warn",
					Path:    renderedFile.Path,
					Message: "We currently have no matching schema to lint this type of file",
				}
			} else {
				lintExpression = LintExpression{
					Rule:    "kubeval-error",
					Type:    "error",
					Path:    renderedFile.Path,
					Message: err.Error(),
				}
			}

			lintExpressions = append(lintExpressions, lintExpression)

			continue // don't stop
		}

		for _, validationResult := range results {
			for _, validationError := range validationResult.Errors {
				lintExpression := LintExpression{
					Rule:    validationError.Type(),
					Type:    "warn",
					Path:    renderedFile.Path,
					Message: validationError.Description(),
				}

				// we need to get the line number for the original file content
				// not the rendered version of it, and not the separated document
				yamlPath := validationError.Field()
				foundSpecFile, err := originalFiles.getFile(renderedFile.Path)
				if err != nil {
					lintExpressions = append(lintExpressions, lintExpression)
					continue
				}

				line, err := util.GetLineNumberFromYamlPath(foundSpecFile.Content, yamlPath, renderedFile.DocIndex)
				if err != nil || line == -1 {
					lintExpressions = append(lintExpressions, lintExpression)
					continue
				}

				lintExpression.Positions = []LintExpressionItemPosition{
					{
						Start: LintExpressionItemLinePosition{
							Line: line,
						},
					},
				}

				lintExpressions = append(lintExpressions, lintExpression)
			}
		}
	}

	return lintExpressions, nil
}

func lintRenderContent(specFiles SpecFiles) ([]LintExpression, SpecFiles, error) {
	lintExpressions := []LintExpression{}

	separatedSpecFiles, err := specFiles.separate()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to separate multi docs")
	}

	// check if config is valid
	config, path, err := separatedSpecFiles.findAndValidateConfig()
	if err != nil {
		lintExpression := LintExpression{
			Rule:    "config-is-invalid",
			Type:    "error",
			Path:    path,
			Message: err.Error(),
		}
		lintExpressions = append(lintExpressions, lintExpression)
	}

	builder, err := getTemplateBuilder(config)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get template builder")
	}

	// rendering files is an expensive process, store and return the rendered files
	// from this function so that they can be used later instead of rendering again on the fly
	renderedFiles := SpecFiles{}

	for _, file := range separatedSpecFiles {
		renderedContent, err := file.renderContent(builder)
		if err == nil {
			file.Content = string(renderedContent)
			renderedFiles = append(renderedFiles, file)
			continue
		}
		// check if the error is coming from kots RenderTemplate function
		if renderErr, ok := errors.Cause(err).(RenderTemplateError); ok {
			lintExpression := LintExpression{
				Rule:    "unable-to-render",
				Type:    "error",
				Path:    file.Path,
				Message: renderErr.Error(),
			}

			if renderErr.Match() != "" {
				// we need to get the line number for the original file content not the separated document
				foundSpecFile, err := specFiles.getFile(file.Path)
				if err != nil {
					lintExpressions = append(lintExpressions, lintExpression)
					continue
				}

				line, err := util.GetLineNumberFromMatch(foundSpecFile.Content, renderErr.Match(), file.DocIndex)
				if err != nil || line == -1 {
					lintExpressions = append(lintExpressions, lintExpression)
					continue
				}
				lintExpression.Positions = []LintExpressionItemPosition{
					{
						Start: LintExpressionItemLinePosition{
							Line: line,
						},
					},
				}
			}

			lintExpressions = append(lintExpressions, lintExpression)
			continue
		}
		// error is not caused by kots RenderTemplate, something went wrong
		return nil, nil, errors.Wrapf(err, "failed to render spec file %s", file.Path)
	}

	return lintExpressions, renderedFiles, nil
}

func lintHelmCharts(specFiles SpecFiles) ([]LintExpression, error) {
	lintExpressions := []LintExpression{}

	// separate multi docs because the manifest can be a part of a multi doc yaml file
	separatedSpecFiles, err := specFiles.separate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to separate multi docs")
	}

	// check if all helm charts have corresponding archives
	allKotsHelmCharts := findAllKotsHelmCharts(separatedSpecFiles)
	for _, helmChart := range allKotsHelmCharts {
		archiveExists, err := archiveForHelmChartExists(separatedSpecFiles, helmChart)
		if err != nil {
			return nil, errors.Wrap(err, "failed to check if archive for helm chart exists")
		}

		if !archiveExists {
			lintExpression := LintExpression{
				Rule:    "helm-archive-missing",
				Type:    "error",
				Message: fmt.Sprintf("Could not find helm archive for chart '%s' version '%s'", helmChart.Spec.Chart.Name, helmChart.Spec.Chart.ChartVersion),
			}
			lintExpressions = append(lintExpressions, lintExpression)
		}
	}

	// check if all archives have corresponding helm chart manifests
	for _, specFile := range separatedSpecFiles {
		if !specFile.isTarGz() {
			continue
		}

		chartExists, err := helmChartForArchiveExists(allKotsHelmCharts, specFile)
		if err != nil {
			return nil, errors.Wrap(err, "failed to check if helm chart for archive exists")
		}

		if !chartExists {
			lintExpression := LintExpression{
				Rule:    "helm-chart-missing",
				Type:    "error",
				Message: fmt.Sprintf("Could not find helm chart manifest for archive '%s'", specFile.Path),
			}
			lintExpressions = append(lintExpressions, lintExpression)
		}
	}

	return lintExpressions, nil
}

func lintIsValidYAML(specFiles SpecFiles) []LintExpression {
	lintExpressions := []LintExpression{}

	// all files must be valid YAML, so without a schema, attempt to parse them
	// we do this separately because it's really hard to get kubeval to
	// return valid errors on all types of invalid yaml

	for _, specFile := range specFiles {
		fileLintExpressions := lintFileHasValidYAML(specFile)
		lintExpressions = append(lintExpressions, fileLintExpressions...)
	}

	return lintExpressions
}

func lintFileHasValidYAML(file SpecFile) []LintExpression {
	lintExpressions := []LintExpression{}

	reader := bytes.NewReader([]byte(file.Content))
	decoder := goyaml.NewDecoder(reader)
	decoder.SetStrict(true)

	for {
		var doc interface{}
		err := decoder.Decode(&doc)

		if err == nil {
			continue
		}

		if err == io.EOF {
			break
		}

		lintExpression := LintExpression{
			Rule:    "invalid-yaml",
			Type:    "error",
			Path:    file.Path,
			Message: err.Error(),
		}

		line, err := util.TryGetLineNumberFromValue(err.Error())
		if err == nil && line > -1 {
			lintExpression.Positions = []LintExpressionItemPosition{
				{
					Start: LintExpressionItemLinePosition{
						Line: line,
					},
				},
			}
		}

		lintExpressions = append(lintExpressions, lintExpression)

		break // break on first error, otherwise decoder will panic
	}

	return lintExpressions
}

func lintExpressionsHaveErrors(lintExpressions []LintExpression) bool {
	for _, lintExpression := range lintExpressions {
		if lintExpression.Type == "error" {
			return true
		}
	}
	return false
}

// archiveForHelmChartExists iterates through all files, looking for a helm chart archive
// that matches the chart name and version specified in the kotsHelmChart parameter
func archiveForHelmChartExists(specFiles SpecFiles, kotsHelmChart *kotsv1beta1.HelmChart) (bool, error) {
	for _, specFile := range specFiles {
		if !specFile.isTarGz() {
			continue
		}

		// We treat all .tar.gz archives as helm charts
		files, err := SpecFilesFromTarGz(specFile)
		if err != nil {
			return false, errors.Wrap(err, "failed to read chart archive")
		}

		for _, file := range files {
			if file.Path == "Chart.yaml" {
				chartManifest := new(chart.Metadata)
				if err := yaml.Unmarshal([]byte(file.Content), chartManifest); err != nil {
					return false, errors.Wrap(err, "failed to unmarshal chart yaml")
				}

				if chartManifest.Name == kotsHelmChart.Spec.Chart.Name {
					if chartManifest.Version == kotsHelmChart.Spec.Chart.ChartVersion {
						return true, nil
					}
				}
			}
		}
	}

	return false, nil
}

// helmChartForArchiveExists iterates through all existing helm charts, looking for a helm chart manifest
// that matches the chart name and version specified in the Chart.yaml file in the archive
func helmChartForArchiveExists(allKotsHelmCharts []*kotsv1beta1.HelmChart, archive SpecFile) (bool, error) {
	files, err := SpecFilesFromTarGz(archive)
	if err != nil {
		return false, errors.Wrap(err, "failed to read chart archive")
	}

	for _, file := range files {
		if file.Path != "Chart.yaml" {
			continue
		}

		chartManifest := new(chart.Metadata)
		if err := yaml.Unmarshal([]byte(file.Content), chartManifest); err != nil {
			return false, errors.Wrap(err, "failed to unmarshal chart yaml")
		}

		for _, kotsHelmChart := range allKotsHelmCharts {
			if chartManifest.Name == kotsHelmChart.Spec.Chart.Name {
				if chartManifest.Version == kotsHelmChart.Spec.Chart.ChartVersion {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

func findAllKotsHelmCharts(specFiles SpecFiles) []*kotsv1beta1.HelmChart {
	kotsHelmCharts := []*kotsv1beta1.HelmChart{}
	for _, specFile := range specFiles {
		kotsHelmChart := tryParsingAsHelmChartGVK([]byte(specFile.Content))
		if kotsHelmChart != nil {
			kotsHelmCharts = append(kotsHelmCharts, kotsHelmChart)
		}
	}

	return kotsHelmCharts
}

func tryParsingAsHelmChartGVK(content []byte) *kotsv1beta1.HelmChart {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(content, nil, nil)
	if err != nil {
		return nil
	}

	if gvk.Group == "kots.io" {
		if gvk.Version == "v1beta1" {
			if gvk.Kind == "HelmChart" {
				return obj.(*kotsv1beta1.HelmChart)
			}
		}
	}

	return nil
}
