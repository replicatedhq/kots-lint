package kots

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/instrumenta/kubeval/kubeval"
	"github.com/pkg/errors"
	kjs "github.com/replicatedhq/kots-lint/kubernetes_json_schema"
	"github.com/replicatedhq/kots-lint/pkg/domain"
	"github.com/replicatedhq/kots-lint/pkg/util"
	goyaml "gopkg.in/yaml.v2"
)

func TroubleshootLintSpec(spec string) ([]domain.LintExpression, error) {
	// if there are yaml errors, end early there
	yamlLintExpressions := lintSpecHasValidYAML(spec)
	if lintExpressionsHaveErrors(yamlLintExpressions) {
		return yamlLintExpressions, nil
	}

	kubevalLintExpressions, err := lintSpecWithKubeval(spec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to lint spec with Kubeval")
	}

	allLintExpressions := []domain.LintExpression{}
	allLintExpressions = append(allLintExpressions, yamlLintExpressions...)
	allLintExpressions = append(allLintExpressions, kubevalLintExpressions...)

	return allLintExpressions, nil
}

func lintSpecWithKubeval(spec string) ([]domain.LintExpression, error) {
	return lintSpecWithKubevalSchema(spec, fmt.Sprintf("file://%s", kjs.KubernetesJsonSchemaDir))
}

func lintSpecWithKubevalSchema(spec string, schemaLocation string) ([]domain.LintExpression, error) {
	lintExpressions := []domain.LintExpression{}

	kubevalConfig := kubeval.Config{
		SchemaLocation:    schemaLocation,
		Strict:            true,
		KubernetesVersion: kjs.KUBERNETES_LINT_VERSION,
	}

	results, err := kubeval.Validate([]byte(spec), &kubevalConfig)
	if err != nil {
		var lintExpression domain.LintExpression

		if strings.Contains(err.Error(), "Failed initalizing schema") && strings.Contains(err.Error(), "no such file or directory") {
			lintExpression = domain.LintExpression{
				Rule:    "kubeval-schema-not-found",
				Type:    "warn",
				Message: "We currently have no matching schema to lint this type of file",
			}
		} else {
			lintExpression = domain.LintExpression{
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
			lintExpression := domain.LintExpression{
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

			lintExpression.Positions = []domain.LintExpressionItemPosition{
				{
					Start: domain.LintExpressionItemLinePosition{
						Line: line,
					},
				},
			}

			lintExpressions = append(lintExpressions, lintExpression)
		}
	}

	return lintExpressions, nil
}

func lintSpecHasValidYAML(spec string) []domain.LintExpression {
	lintExpressions := []domain.LintExpression{}

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

		lintExpression := domain.LintExpression{
			Rule:    "invalid-yaml",
			Type:    "error",
			Message: err.Error(),
		}

		line, err := util.TryGetLineNumberFromValue(err.Error())
		if err == nil && line > -1 {
			lintExpression.Positions = []domain.LintExpressionItemPosition{
				{
					Start: domain.LintExpressionItemLinePosition{
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
