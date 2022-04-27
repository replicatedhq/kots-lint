package kots

import (
	"bytes"
	"io"
	"strings"

	"github.com/instrumenta/kubeval/kubeval"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots-lint/pkg/util"
	goyaml "gopkg.in/yaml.v2"
)

func TroubleshootLintSpec(spec string) ([]LintExpression, error) {
	// if there are yaml errors, end early there
	yamlLintExpressions := lintSpecHasValidYAML(spec)
	if lintExpressionsHaveErrors(yamlLintExpressions) {
		return yamlLintExpressions, nil
	}

	kubevalLintExpressions, err := lintSpecWithKubeval(spec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to lint spec with Kubeval")
	}

	allLintExpressions := []LintExpression{}
	allLintExpressions = append(allLintExpressions, yamlLintExpressions...)
	allLintExpressions = append(allLintExpressions, kubevalLintExpressions...)

	return allLintExpressions, nil
}

func lintSpecWithKubeval(spec string) ([]LintExpression, error) {
	return lintSpecWithKubevalSchema(spec, "file://kubernetes-json-schema")
}

func lintSpecWithKubevalSchema(spec string, schemaLocation string) ([]LintExpression, error) {
	lintExpressions := []LintExpression{}

	kubevalConfig := kubeval.Config{
		SchemaLocation:    schemaLocation,
		Strict:            true,
		KubernetesVersion: "1.23.6",
	}

	results, err := kubeval.Validate([]byte(spec), &kubevalConfig)
	if err != nil {
		var lintExpression LintExpression

		if strings.Contains(err.Error(), "Failed initalizing schema") && strings.Contains(err.Error(), "no such file or directory") {
			lintExpression = LintExpression{
				Rule:    "kubeval-schema-not-found",
				Type:    "warn",
				Message: "We currently have no matching schema to lint this type of file",
			}
		} else {
			lintExpression = LintExpression{
				Rule:    "kubeval-error",
				Type:    "error",
				Message: err.Error(),
			}
		}

		lintExpressions = append(lintExpressions, lintExpression)

		return lintExpressions, nil
	}

	for _, validationResult := range results {
		for _, validationError := range validationResult.Errors {
			lintExpression := LintExpression{
				Rule:    validationError.Type(),
				Type:    "warn",
				Message: validationError.Description(),
			}

			yamlPath := validationError.Field()
			line, err := util.GetLineNumberFromYamlPath(spec, yamlPath, 0)
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

	return lintExpressions, nil
}

func lintSpecHasValidYAML(spec string) []LintExpression {
	lintExpressions := []LintExpression{}

	reader := bytes.NewReader([]byte(spec))
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
