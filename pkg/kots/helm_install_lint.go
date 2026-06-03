package kots

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots-lint/pkg/domain"
	"gopkg.in/yaml.v2"
)

type helmInstallDoc struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Annotations map[string]string `yaml:"annotations"`
	} `yaml:"metadata"`
}

var replicatedKubernetesAPIVersions = map[string]bool{
	"embeddedcluster.replicated.com/v1beta1": true,
	"cluster.kurl.sh/v1beta1":                true,
	"kots.io/v1beta1":                        true,
	"kots.io/v1beta2":                        true,
	"troubleshoot.sh/v1beta2":                true,
}

// lintHelmInstallType checks the requirements for Helm CLI install types, that are
// - must have HelmChart CR and chart tgz (already checked by lintHelmCharts)
// - any non Replicated Kubernetes resources must include the kots.io/installer-only annotation.
func lintHelmInstallType(ctx context.Context, specFiles domain.SpecFiles) ([]domain.LintExpression, error) {
	lintExpressions := []domain.LintExpression{}

	separatedSpecFiles, err := specFiles.Separate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to separate multi docs")
	}

	for _, spec := range separatedSpecFiles {
		// stop early if the caller cancelled or the deadline passed
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// only check files in the root directory of the release
		if !isInRootDir(spec.Path) {
			continue
		}

		var doc helmInstallDoc
		if err := yaml.Unmarshal([]byte(spec.Content), &doc); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal spec content")
		}

		// Separate() can emit empty docs (e.g. a trailing `---`); skip them.
		if doc.APIVersion == "" && doc.Kind == "" {
			continue
		}

		// Helm CLI install can include Replicated custom resources
		if replicatedKubernetesAPIVersions[doc.APIVersion] {
			continue
		}

		// All other Kubernetes resources must be annotated with kots.io/installer-only
		if _, ok := doc.Metadata.Annotations["kots.io/installer-only"]; ok {
			continue
		}

		lintExpressions = append(lintExpressions, domain.LintExpression{
			Rule:    "helm-install-type-missing-annotation",
			Type:    "error",
			Path:    spec.Path,
			Message: fmt.Sprintf("%s %s is missing the required kots.io/installer-only annotation for Helm install type", doc.APIVersion, doc.Kind),
		})
	}

	return lintExpressions, nil
}

// isInRootDir checks if the file is a file in root directory of Replicated release
func isInRootDir(path string) bool {
	return strings.Count(path, "/") <= 1
}
