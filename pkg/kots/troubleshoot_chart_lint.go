package kots

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots-lint/pkg/domain"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

// troubleshootCRDKinds maps the troubleshoot CR kind to the expected CRD name (plural.group)
var troubleshootCRDKinds = map[string]string{
	"Preflight":     "preflights.troubleshoot.sh",
	"SupportBundle": "supportbundles.troubleshoot.sh",
}

// lintHelmChartTroubleshootCRDs warns when a helm chart contains a top-level
// kind: Preflight or kind: SupportBundle resource that would require the
// corresponding CRD to be installed in the cluster, but the chart does not
// include that CRD. Embedding these specs in a Secret (with the
// troubleshoot.sh/kind label) is the recommended pattern because it does not
// require cluster-admin permissions or pre-installed CRDs.
func lintHelmChartTroubleshootCRDs(ctx context.Context, tarGzFiles domain.SpecFiles) ([]domain.LintExpression, error) {
	lintExpressions := []domain.LintExpression{}

	for _, tarGzFile := range tarGzFiles {
		if !tarGzFile.IsTarGz() {
			continue
		}

		content, err := base64.StdEncoding.DecodeString(tarGzFile.Content)
		if err != nil {
			log.Debugf("failed to base64 decode tarGz content: %v", err)
			continue
		}

		ch, err := loader.LoadArchive(bytes.NewReader(content))
		if err != nil {
			log.Debugf("failed to load chart archive %s: %v", tarGzFile.Path, err)
			continue
		}

		chartLintExpressions, err := lintChartForTroubleshootCRDs(ch, tarGzFile.Path)
		if err != nil {
			return nil, errors.Wrapf(err, "lint chart %s", tarGzFile.Path)
		}
		lintExpressions = append(lintExpressions, chartLintExpressions...)
	}

	return lintExpressions, nil
}

func lintChartForTroubleshootCRDs(ch *chart.Chart, chartPath string) ([]domain.LintExpression, error) {
	lintExpressions := []domain.LintExpression{}

	declaredCRDs, err := collectChartCRDs(ch)
	if err != nil {
		return nil, errors.Wrap(err, "collect CRDs")
	}

	renderedTemplates, err := renderChartTemplates(ch)
	if err != nil {
		// Rendering can fail for charts that depend on values not exercised here;
		// don't block linting in that case.
		log.Debugf("failed to render chart %s for troubleshoot CRD lint: %v", chartPath, err)
		return lintExpressions, nil
	}

	// templates can also declare CRDs (some charts use templates/crds/ instead of crds/)
	for _, content := range renderedTemplates {
		for _, doc := range splitYAMLDocs(content) {
			if name, ok := crdNameFromDoc(doc); ok {
				declaredCRDs[name] = true
			}
		}
	}

	seen := map[string]bool{}
	for templateName, content := range renderedTemplates {
		for _, doc := range splitYAMLDocs(content) {
			kind, apiVersion := kindAndAPIVersion(doc)
			if !isTroubleshootCRKind(kind, apiVersion) {
				continue
			}

			expectedCRD, ok := troubleshootCRDKinds[kind]
			if !ok {
				continue
			}
			if declaredCRDs[expectedCRD] {
				continue
			}

			key := fmt.Sprintf("%s|%s|%s", chartPath, templateName, kind)
			if seen[key] {
				continue
			}
			seen[key] = true

			lintExpressions = append(lintExpressions, domain.LintExpression{
				Rule: "troubleshoot-spec-in-chart-without-crd",
				Type: "warn",
				Path: chartPath,
				Message: fmt.Sprintf(
					"Helm chart template %q contains a %s custom resource but the chart does not include the %s CRD. "+
						"Installing the CR will fail in clusters where the CRD is not already present. "+
						"Embed the spec in a Kubernetes Secret with the troubleshoot.sh/kind label instead. "+
						"See https://docs.replicated.com/vendor/preflight-defining for an example.",
					templateName, kind, expectedCRD,
				),
			})
		}
	}

	return lintExpressions, nil
}

func collectChartCRDs(ch *chart.Chart) (map[string]bool, error) {
	declared := map[string]bool{}
	for _, crdFile := range ch.CRDObjects() {
		for _, doc := range splitYAMLDocs(string(crdFile.File.Data)) {
			if name, ok := crdNameFromDoc(doc); ok {
				declared[name] = true
			}
		}
	}
	for _, dep := range ch.Dependencies() {
		depCRDs, err := collectChartCRDs(dep)
		if err != nil {
			return nil, err
		}
		for name := range depCRDs {
			declared[name] = true
		}
	}
	return declared, nil
}

func renderChartTemplates(ch *chart.Chart) (map[string]string, error) {
	options := chartutil.ReleaseOptions{
		Name: "app-chart",
	}

	// If chart has a schema file, it will be used to validate values, which will
	// fail if there are missing required values. Match the behavior of
	// GetFilesFromChartReader so we render best-effort.
	ch.Schema = nil
	if err := chartutil.ProcessDependencies(ch, chartutil.Values{}); err != nil {
		return nil, errors.Wrap(err, "process dependencies")
	}

	rValues, err := chartutil.ToRenderValues(ch, ch.Values, options, nil)
	if err != nil {
		return nil, errors.Wrap(err, "convert values to render values")
	}

	eng := new(engine.Engine)
	eng.LintMode = true

	rendered, err := eng.Render(ch, rValues)
	if err != nil {
		return nil, errors.Wrap(err, "render templates")
	}
	return rendered, nil
}

func splitYAMLDocs(content string) []string {
	parts := strings.Split(content, "\n---")
	docs := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		docs = append(docs, part)
	}
	return docs
}

func kindAndAPIVersion(doc string) (string, string) {
	parsed := struct {
		Kind       string `yaml:"kind"`
		APIVersion string `yaml:"apiVersion"`
	}{}
	if err := yaml.Unmarshal([]byte(doc), &parsed); err != nil {
		return "", ""
	}
	return parsed.Kind, parsed.APIVersion
}

func isTroubleshootCRKind(kind, apiVersion string) bool {
	if _, ok := troubleshootCRDKinds[kind]; !ok {
		return false
	}
	return strings.HasPrefix(apiVersion, "troubleshoot.sh/") ||
		strings.HasPrefix(apiVersion, "troubleshoot.replicated.com/")
}

func crdNameFromDoc(doc string) (string, bool) {
	parsed := struct {
		Kind       string `yaml:"kind"`
		APIVersion string `yaml:"apiVersion"`
		Metadata   struct {
			Name string `yaml:"name"`
		} `yaml:"metadata"`
	}{}
	if err := yaml.Unmarshal([]byte(doc), &parsed); err != nil {
		return "", false
	}
	if parsed.Kind != "CustomResourceDefinition" {
		return "", false
	}
	if !strings.HasPrefix(parsed.APIVersion, "apiextensions.k8s.io/") {
		return "", false
	}
	if parsed.Metadata.Name == "" {
		return "", false
	}
	return parsed.Metadata.Name, true
}
