package kots

import (
	"context"
	_ "embed"

	"github.com/mitchellh/mapstructure"
	"github.com/open-policy-agent/opa/rego"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots-lint/pkg/domain"
)

//go:embed rego/enterprise-opa-prepend.rego
var enterpriseRegoPrepend string

type EnterprisePolicy struct {
	Name   string `json:"name"`
	Policy string `json:"policy"`
}

func EnterpriseLintSpecFiles(specFiles domain.SpecFiles, policies []EnterprisePolicy) ([]domain.LintExpression, error) {
	unnestedFiles := specFiles.Unnest()

	filteredFiles := domain.SpecFiles{}
	for _, file := range unnestedFiles {
		if file.IsYAML() {
			filteredFiles = append(filteredFiles, file)
		}
	}

	separatedSpecFiles, err := filteredFiles.Separate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to separate multi docs")
	}

	config, _, err := separatedSpecFiles.FindAndValidateConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to find config")
	}

	builder, err := domain.GetTemplateBuilder(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get template builder")
	}

	// get the rendered version of the spec files before linting
	renderedFiles := domain.SpecFiles{}
	for _, file := range separatedSpecFiles {
		renderedContent, err := file.RenderContent(builder)
		if err != nil {
			return nil, errors.Wrap(err, "failed to render spec file content")
		}
		file.Content = string(renderedContent)
		renderedFiles = append(renderedFiles, file)
	}

	lintExpressions := []domain.LintExpression{}
	for _, policy := range policies {
		expressions, err := lintWithOPAPolicy(renderedFiles, policy.Policy)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to lint with enterprise policy %s", policy.Name)
		}
		lintExpressions = append(lintExpressions, expressions...)
	}

	return lintExpressions, nil
}

func lintWithOPAPolicy(specFiles domain.SpecFiles, policy string) ([]domain.LintExpression, error) {
	ctx := context.Background()

	query, err := rego.New(
		rego.Query("data.kots.enterprise.lint"),
		rego.Module("enterprise-lint.rego", enterpriseRegoPrepend+"\n\n"+policy),
	).PrepareForEval(ctx)

	if err != nil {
		errors.Wrap(err, "failed to prepare query for eval")
	}

	results, err := query.Eval(ctx, rego.EvalInput(specFiles))
	if err != nil {
		return nil, errors.Wrap(err, "failed to evaluate query")
	}
	if len(results) == 0 {
		return []domain.LintExpression{}, nil
	}

	result := results[0]
	if len(result.Expressions) == 0 {
		return []domain.LintExpression{}, nil
	}

	var lintExpressions []domain.LintExpression
	if err := mapstructure.Decode(result.Expressions[0].Value, &lintExpressions); err != nil {
		return nil, errors.Wrap(err, "failed to mapstructure lint expressions")
	}

	return lintExpressions, nil
}
