package kurl

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots-lint/pkg/domain"
	kurllint "github.com/replicatedhq/kurlkinds/pkg/lint"
)

type KurlLinter struct {
	Linter *kurllint.Linter
}

// lintKurlInstaller searches installer yamls for errors or misconfigurations.
func (kurlLinter *KurlLinter) LintKurlInstaller(specFiles domain.SpecFiles) ([]domain.LintExpression, error) {
	separated, err := specFiles.Separate()
	if err != nil {
		return nil, errors.Wrap(err, "error separating spec files")
	}

	var expressions []domain.LintExpression
	for _, file := range separated {
		if !file.IsYAML() {
			continue
		}

		output, err := kurlLinter.Linter.ValidateMarshaledYAML(context.Background(), file.Content)
		if err != nil {
			if err != kurllint.ErrNotInstaller {
				return nil, errors.Wrap(err, "unable to lint installer")
			}
			continue
		}

		for _, out := range output {
			expressions = append(
				expressions, domain.LintExpression{
					Rule:    fmt.Sprintf("kubernetes-installer-%s", out.Type),
					Type:    "error",
					Path:    file.Path,
					Message: out.Message,
				},
			)
		}
	}
	return expressions, nil
}
