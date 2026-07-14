package kots

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots-lint/pkg/domain"
	"github.com/replicatedhq/kotskinds/pkg/helmchart"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/lint/support"
)

// lintHelmChartsWithHelmLint runs `helm lint` against each helm chart archive that
// is referenced by a kots HelmChart custom resource, using the HelmChart's
// `spec.builder` values. Lint findings are reported as helm-schema-violation rules.
//
// renderedFiles are the rendered KOTS spec files (used to find HelmChart CRs and
// extract builder values). tarGzFiles are the helm chart archives in the release.
func lintHelmChartsWithHelmLint(renderedFiles domain.SpecFiles, tarGzFiles domain.SpecFiles) ([]domain.LintExpression, error) {
	lintExpressions := []domain.LintExpression{}

	separated, err := renderedFiles.Separate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to separate multi docs")
	}
	allKotsHelmCharts := findAllKotsHelmCharts(separated)
	if len(allKotsHelmCharts) == 0 {
		return lintExpressions, nil
	}

	for _, tarGzFile := range tarGzFiles {
		if !tarGzFile.IsTarGz() {
			continue
		}

		// Tar archives may be base64-encoded (JSON transport) or raw bytes (tar
		// stream transport). Match domain.SpecFilesFromTarGz's fallback.
		content, err := base64.StdEncoding.DecodeString(tarGzFile.Content)
		if err != nil {
			content = []byte(tarGzFile.Content)
		}
		ch, err := loader.LoadArchive(bytes.NewReader(content))
		if err != nil {
			continue
		}

		matched := matchHelmChartCR(ch, allKotsHelmCharts)
		if matched == nil {
			// No matching kots HelmChart CR — that case is reported by lintHelmCharts.
			continue
		}

		// Only lint Helm v3 (and later) charts. Skip explicit v2 HelmChart CRs.
		if v := matched.GetHelmVersion(); v != "" && v != "v3" {
			continue
		}

		builderValues, err := matched.GetBuilderValues()
		if err != nil {
			lintExpressions = append(lintExpressions, domain.LintExpression{
				Rule:    "helm-schema-violation",
				Type:    "warn",
				Path:    tarGzFile.Path,
				Message: fmt.Sprintf("Failed to evaluate spec.builder values for chart %q: %v", ch.Name(), err),
			})
			continue
		}

		// Process dependencies before coalescing so aliased subcharts are keyed under
		// their alias (not their real Metadata.Name) and disabled subcharts are pruned.
		// This mirrors helm's install/upgrade pipeline (action.install uses
		// ProcessDependenciesWithMerge before rendering) and the other lint paths in
		// this package. Without it, a dependency whose real name collides with a
		// top-level parent values key merges its defaults under the wrong key and
		// produces false-positive helm-schema-violations. Mutates ch in place, which is
		// safe: the chart is loaded fresh for each archive in this loop.
		if err := chartutil.ProcessDependenciesWithMerge(ch, builderValues); err != nil {
			lintExpressions = append(lintExpressions, domain.LintExpression{
				Rule:    "helm-schema-violation",
				Type:    "warn",
				Path:    tarGzFile.Path,
				Message: fmt.Sprintf("Failed to process dependencies for chart %q: %v", ch.Name(), err),
			})
			continue
		}

		// Merge the chart's default values (values.yaml + dependency defaults) with the
		// HelmChart spec.builder overrides so helm lint sees both.
		merged, err := chartutil.CoalesceValues(ch, builderValues)
		if err != nil {
			lintExpressions = append(lintExpressions, domain.LintExpression{
				Rule:    "helm-schema-violation",
				Type:    "warn",
				Path:    tarGzFile.Path,
				Message: fmt.Sprintf("Failed to merge builder and default values for chart %q: %v", ch.Name(), err),
			})
			continue
		}

		chartLintExpressions, err := runHelmLintOnArchive(content, tarGzFile.Path, merged.AsMap())
		if err != nil {
			return nil, errors.Wrapf(err, "helm lint chart %s", tarGzFile.Path)
		}
		lintExpressions = append(lintExpressions, chartLintExpressions...)
	}

	return lintExpressions, nil
}

// matchHelmChartCR returns the HelmChart CR whose chart name and version match
// the metadata in the chart archive, or nil if none match.
func matchHelmChartCR(ch *chart.Chart, helmCharts []chartIdentity) helmchart.HelmChartInterface {
	if ch.Metadata == nil {
		return nil
	}
	for _, hc := range helmCharts {
		if hc.GetChartName() == ch.Metadata.Name && hc.GetChartVersion() == ch.Metadata.Version {
			if matched, ok := hc.(helmchart.HelmChartInterface); ok {
				return matched
			}
		}
	}
	return nil
}

// runHelmLintOnArchive extracts the chart archive bytes to a temp dir and runs
// `helm lint` against it with the supplied values.
func runHelmLintOnArchive(archive []byte, chartPath string, values map[string]interface{}) ([]domain.LintExpression, error) {
	tempDir, err := os.MkdirTemp("", "kots-lint-helm-")
	if err != nil {
		return nil, errors.Wrap(err, "create temp dir")
	}
	defer os.RemoveAll(tempDir)

	tgzPath := filepath.Join(tempDir, "chart.tgz")
	if err := os.WriteFile(tgzPath, archive, 0644); err != nil {
		return nil, errors.Wrap(err, "write chart archive")
	}

	linter := action.NewLint()
	result := linter.Run([]string{tgzPath}, values)

	return helmLintResultToLintExpressions(result, chartPath), nil
}

// helmLintResultToLintExpressions converts helm lint messages and any top-level
// errors into kots-lint expressions under the helm-schema-violation rule.
func helmLintResultToLintExpressions(result *action.LintResult, chartPath string) []domain.LintExpression {
	expressions := []domain.LintExpression{}
	if result == nil {
		return expressions
	}

	// Errors returned outside of Messages typically indicate the chart could not be
	// loaded at all (missing Chart.yaml, etc.). The per-message errors are already
	// included in result.Messages, so we only surface non-message-level errors here
	// to avoid duplicates.
	seenErrors := map[string]bool{}
	for _, m := range result.Messages {
		if m.Err != nil {
			seenErrors[m.Err.Error()] = true
		}
	}
	for _, err := range result.Errors {
		if err == nil {
			continue
		}
		if seenErrors[err.Error()] {
			continue
		}
		for _, finding := range splitHelmLintMessage(err.Error()) {
			expressions = append(expressions, domain.LintExpression{
				Rule:    "helm-schema-violation",
				Type:    "error",
				Path:    chartPath,
				Message: finding,
			})
		}
	}

	for _, msg := range result.Messages {
		if msg.Err == nil {
			continue
		}
		t := helmSeverityToLintType(msg.Severity)
		if t == "" {
			continue
		}
		prefix := ""
		if msg.Path != "" && msg.Path != "." {
			prefix = msg.Path + ": "
		}
		for _, finding := range splitHelmLintMessage(msg.Err.Error()) {
			expressions = append(expressions, domain.LintExpression{
				Rule:    "helm-schema-violation",
				Type:    t,
				Path:    chartPath,
				Message: prefix + finding,
			})
		}
	}

	return expressions
}

// splitHelmLintMessage splits a helm schema validation error message into one
// string per top-level finding. Helm's jsonschema validation collects every
// violation into a single multi-line error: each top-level violation begins on
// a new line with `- at '/...': `, and may be followed by indented children.
// We keep each top-level violation (with its indented children) together while
// preserving any preamble (e.g. "values.yaml:" or chart name) on every finding
// so context is not lost.
func splitHelmLintMessage(text string) []string {
	const marker = "- at "
	idx := strings.Index(text, marker)
	if idx < 0 {
		return []string{strings.TrimRight(text, "\n")}
	}
	preamble := strings.TrimRight(text[:idx], " :\n\t")
	body := text[idx:]
	parts := strings.Split(body, "\n"+marker)
	findings := make([]string, 0, len(parts))
	for i, p := range parts {
		if i > 0 {
			p = marker + p
		}
		p = strings.TrimRight(p, "\n")
		if p == "" {
			continue
		}
		if preamble != "" {
			p = preamble + ": " + p
		}
		findings = append(findings, p)
	}
	if len(findings) == 0 {
		return []string{strings.TrimRight(text, "\n")}
	}
	return findings
}

func helmSeverityToLintType(sev int) string {
	switch sev {
	case support.ErrorSev:
		return "error"
	case support.WarningSev:
		return "warn"
	default:
		// Info and Unknown severities are not surfaced as lint expressions.
		return ""
	}
}
