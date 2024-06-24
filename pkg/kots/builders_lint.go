package kots

import (
	"context"
	_ "embed"

	"github.com/open-policy-agent/opa/rego"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots-lint/pkg/domain"
)

func LintBuilders(ctx context.Context, files domain.SpecFiles) ([]domain.LintExpression, error) {
	opaResults, err := buildersRegoQuery.Eval(ctx, rego.EvalInput(files))

	if err != nil {
		return nil, errors.Wrap(err, "evaluate query")
	}

	lintResult, err := opaResultsToLintExpressions(opaResults, files)
	if err != nil {
		return nil, errors.Wrap(err, "convert opa results to lint expressions")
	}

	return lintResult, nil
}
