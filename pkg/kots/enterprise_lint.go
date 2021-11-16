package kots

import (
	"context"
	"io/ioutil"

	"github.com/mitchellh/mapstructure"
	"github.com/open-policy-agent/opa/rego"
	"github.com/pkg/errors"
)

type EnterprisePolicy struct {
	Name   string `json:"name"`
	Policy string `json:"policy"`
}

func EnterpriseLintSpecFiles(specFiles SpecFiles, policies []EnterprisePolicy) ([]LintExpression, error) {
	unnestedFiles := specFiles.unnest()

	filteredFiles := SpecFiles{}
	for _, file := range unnestedFiles {
		if file.isYAML() {
			filteredFiles = append(filteredFiles, file)
		}
	}

	separatedSpecFiles, err := filteredFiles.separate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to separate multi docs")
	}

	config, _, err := separatedSpecFiles.findAndValidateConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to find config")
	}

	builder, err := getTemplateBuilder(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get template builder")
	}

	// get the rendered version of the spec files before linting
	renderedFiles := SpecFiles{}
	for _, file := range separatedSpecFiles {
		renderedContent, err := file.renderContent(builder)
		if err != nil {
			return nil, errors.Wrap(err, "failed to render spec file content")
		}
		file.Content = string(renderedContent)
		renderedFiles = append(renderedFiles, file)
	}

	lintExpressions := []LintExpression{}
	for _, policy := range policies {
		expressions, err := lintWithOPAPolicy(renderedFiles, policy.Policy)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to lint with enterprise policy %s", policy.Name)
		}
		lintExpressions = append(lintExpressions, expressions...)
	}

	return lintExpressions, nil
}

func lintWithOPAPolicy(specFiles SpecFiles, policy string) ([]LintExpression, error) {
	b, err := ioutil.ReadFile("/rego/enterprise-opa-prepend.rego")
	if err != nil {
		return nil, errors.Wrap(err, "failed to read rego file")
	}
	regoPrepend := string(b)

	ctx := context.Background()

	query, err := rego.New(
		rego.Query("data.kots.enterprise.lint"),
		rego.Module("enterprise-lint.rego", regoPrepend+"\n\n"+policy),
	).PrepareForEval(ctx)

	if err != nil {
		errors.Wrap(err, "failed to prepare query for eval")
	}

	results, err := query.Eval(ctx, rego.EvalInput(specFiles))
	if err != nil {
		return nil, errors.Wrap(err, "failed to evaluate query")
	}
	if len(results) == 0 {
		return []LintExpression{}, nil
	}

	result := results[0]
	if len(result.Expressions) == 0 {
		return []LintExpression{}, nil
	}

	var lintExpressions []LintExpression
	if err := mapstructure.Decode(result.Expressions[0].Value, &lintExpressions); err != nil {
		return nil, errors.Wrap(err, "failed to mapstructure lint expressions")
	}

	return lintExpressions, nil
}
