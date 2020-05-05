package kots

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"strings"

	"github.com/instrumenta/kubeval/kubeval"
	"github.com/mitchellh/mapstructure"
	"github.com/open-policy-agent/opa/rego"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots-lint/pkg/util"
	goyaml "gopkg.in/yaml.v2"
)

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

var regoQuery *rego.PreparedEvalQuery

func InitOPALinting(regoPath string) error {
	regoContent, err := ioutil.ReadFile(regoPath)
	if err != nil {
		return errors.Wrap(err, "failed to read rego file")
	}

	ctx := context.Background()

	query, err := rego.New(
		rego.Query("data.kots.spec.lint"),
		rego.Module("kots-spec-default.rego", string(regoContent)),
	).PrepareForEval(ctx)

	if err != nil {
		errors.Wrap(err, "failed to prepare query for eval")
	}

	regoQuery = &query

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

	opaLintExpressions, err := lintWithOPA(filteredFiles)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to lint with OPA")
	}
	// if there are opa errors, end early there
	if lintExpressionsHaveErrors(opaLintExpressions) {
		return opaLintExpressions, false, nil
	}

	kubevalLintExpressions, err := lintWithKubeval(filteredFiles)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to lint with Kubeval")
	}

	allLintExpressions := []LintExpression{}
	allLintExpressions = append(allLintExpressions, yamlLintExpressions...)
	allLintExpressions = append(allLintExpressions, opaLintExpressions...)
	allLintExpressions = append(allLintExpressions, kubevalLintExpressions...)

	return allLintExpressions, true, nil
}

// InitOPALinting needs to be called first with a rego policy
// in order for this function to run successfully
func lintWithOPA(specFiles SpecFiles) ([]LintExpression, error) {
	lintExpressions := []LintExpression{}

	separatedSpecFiles, err := specFiles.separate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to separate multi docs")
	}

	ctx := context.Background()

	results, err := regoQuery.Eval(ctx, rego.EvalInput(separatedSpecFiles))
	if err != nil {
		return nil, errors.Wrap(err, "failed to evaluate query")
	}
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

		// we need to get the line number for the original file content not the separated document
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

func lintWithKubeval(specFiles SpecFiles) ([]LintExpression, error) {
	return lintWithKubevalSchema(specFiles, "file://kubernetes-json-schema")
}

func lintWithKubevalSchema(specFiles SpecFiles, schemaLocation string) ([]LintExpression, error) {
	lintExpressions := []LintExpression{}

	separatedSpecFiles, err := specFiles.separate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to separate multi docs")
	}

	// check if config is valid
	config, path, err := separatedSpecFiles.findAndValidateConfig()
	if err != nil {
		lintExpression := LintExpression{
			Rule:    "config-is-invalid",
			Type:    "error",
			Path:    path, // TODO maybe add line number?
			Message: err.Error(),
		}
		lintExpressions = append(lintExpressions, lintExpression)
	}

	// get the rendered version of the spec files before linting
	renderedFiles := SpecFiles{}
	for _, file := range separatedSpecFiles {
		renderedContent, err := file.renderContent(config)
		if err == nil {
			file.Content = string(renderedContent)
			renderedFiles = append(renderedFiles, file)
			continue
		}
		// check if the error is coming from kots RenderTemplate function
		if err, ok := errors.Cause(err).(RenderTemplateError); ok {
			lintExpression := LintExpression{
				Rule:    "unable-to-render",
				Type:    "error",
				Path:    file.Path,
				Message: err.Error(),
			}

			if err.Line() != -1 {
				lintExpression.Positions = []LintExpressionItemPosition{
					{
						Start: LintExpressionItemLinePosition{
							Line: err.Line(),
						},
					},
				}
			}

			lintExpressions = append(lintExpressions, lintExpression)
			continue
		}
		// error is not caused by kots RenderTemplate, something went wrong
		return nil, errors.Wrap(err, "failed to render spec file content")
	}

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
				foundSpecFile, err := specFiles.getFile(renderedFile.Path)
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
