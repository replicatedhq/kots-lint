package kots

import (
	"strings"
	"testing"

	"github.com/replicatedhq/kots-lint/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_splitHelmLintMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "no findings returns single entry",
			input:    "some other helm error",
			expected: []string{"some other helm error"},
		},
		{
			name:  "single top-level finding with preamble",
			input: "values.yaml: - at '/seed': missing property 'resources'",
			expected: []string{
				"values.yaml: - at '/seed': missing property 'resources'",
			},
		},
		{
			name: "multiple top-level findings split, preamble preserved",
			input: "values.yaml: - at '/seed': missing property 'resources'\n" +
				"- at '/worker': missing properties 'livenessProbe', 'readinessProbe'\n",
			expected: []string{
				"values.yaml: - at '/seed': missing property 'resources'",
				"values.yaml: - at '/worker': missing properties 'livenessProbe', 'readinessProbe'",
			},
		},
		{
			name: "nested children stay with their parent",
			input: "values.yaml: - at '/ingress': validation failed\n" +
				"  - at '/ingress': missing property 'certManager'\n" +
				"  - at '/ingress/tls': missing property 'secretName'\n" +
				"- at '/web': missing property 'service'\n",
			expected: []string{
				"values.yaml: - at '/ingress': validation failed\n" +
					"  - at '/ingress': missing property 'certManager'\n" +
					"  - at '/ingress/tls': missing property 'secretName'",
				"values.yaml: - at '/web': missing property 'service'",
			},
		},
		{
			name: "multi-line preamble preserved",
			input: "templates/: values don't meet the specifications of the schema(s) in the following chart(s):\n" +
				"vendorflow:\n" +
				"- at '/seed': missing property 'resources'\n" +
				"- at '/api': missing properties 'service'\n",
			expected: []string{
				"templates/: values don't meet the specifications of the schema(s) in the following chart(s):\nvendorflow: - at '/seed': missing property 'resources'",
				"templates/: values don't meet the specifications of the schema(s) in the following chart(s):\nvendorflow: - at '/api': missing properties 'service'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitHelmLintMessage(tt.input)
			assert.Equal(t, tt.expected, got, "input:\n%s", tt.input)
			for _, f := range got {
				assert.False(t, strings.HasSuffix(f, "\n"), "finding should not end with newline: %q", f)
			}
		})
	}
}

const helmChartCRWithBuilder = `apiVersion: kots.io/v1beta1
kind: HelmChart
metadata:
  name: testchart
spec:
  chart:
    name: testchart
    chartVersion: 0.1.0
  builder:
    image:
      repository: nginx
      tag: stable
`

// chart whose templates fail to render unless `image.repository` is set as a string.
const requireValueChart = `apiVersion: v2
name: testchart
version: 0.1.0
`
const requireValueDeployment = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: app
  template:
    metadata:
      labels:
        app: app
    spec:
      containers:
        - name: app
          image: {{ required "image.repository is required" .Values.image.repository }}:{{ .Values.image.tag }}
`

// Regression: an umbrella chart whose dependency uses an `alias` where the
// dependency's real name collides with a top-level values key the parent chart
// itself uses. If dependencies are not processed before coalescing, the subchart
// defaults are keyed under the subchart's real Metadata.Name ("envoyproxy")
// instead of its alias ("gatewayenvoyproxy"), so they merge into the parent's
// own `envoyproxy` workload map and corrupt it — producing false-positive
// helm-schema-violations. Processing dependencies first scopes the subchart under
// its alias, so the parent key stays clean and the chart lints cleanly (as
// `helm lint`/`helm install` already do).
const aliasCollisionChartYAML = `apiVersion: v2
name: testchart
version: 0.1.0
dependencies:
  - name: envoyproxy
    version: 0.1.0
    alias: gatewayenvoyproxy
`

// Parent uses `envoyproxy` as a map of workloads; each workload must carry a
// resources block. Iterating a workload that lacks it dereferences a nil map and
// fails to render.
const aliasCollisionValuesYAML = `envoyproxy:
  gateway1:
    resources:
      requests:
        cpu: 100m
`

const aliasCollisionDeployment = `{{- range $name, $cfg := .Values.envoyproxy }}
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ $name }}
data:
  cpu: {{ $cfg.resources.requests.cpu }}
{{- end }}
`

// The aliased subchart, keyed by its real name "envoyproxy", ships defaults that
// do NOT carry a resources block — exactly the keys that corrupt the parent's
// workload map when the alias is not applied.
const aliasCollisionSubchartYAML = `apiVersion: v2
name: envoyproxy
version: 0.1.0
`
const aliasCollisionSubchartValues = `config:
  foo: bar
deployment:
  replicas: 1
`
const aliasCollisionSubchartTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: gateway-config
data:
  foo: {{ .Values.config.foo }}
`

const aliasCollisionHelmChartCR = `apiVersion: kots.io/v1beta1
kind: HelmChart
metadata:
  name: testchart
spec:
  chart:
    name: testchart
    chartVersion: 0.1.0
`

func Test_lintHelmChartsWithHelmLint_aliasedDependencyCollision(t *testing.T) {
	tarGzFiles := domain.SpecFiles{
		{
			Name: "testchart-0.1.0.tar.gz",
			Path: "testchart-0.1.0.tar.gz",
			Content: buildChartArchive(t, []tarChartFile{
				chartFile("testchart/Chart.yaml", aliasCollisionChartYAML),
				chartFile("testchart/values.yaml", aliasCollisionValuesYAML),
				chartFile("testchart/templates/deployment.yaml", aliasCollisionDeployment),
				chartFile("testchart/charts/envoyproxy/Chart.yaml", aliasCollisionSubchartYAML),
				chartFile("testchart/charts/envoyproxy/values.yaml", aliasCollisionSubchartValues),
				chartFile("testchart/charts/envoyproxy/templates/configmap.yaml", aliasCollisionSubchartTemplate),
			}),
		},
	}

	renderedFiles := domain.SpecFiles{
		{
			Name:    "helmchart.yaml",
			Path:    "helmchart.yaml",
			Content: aliasCollisionHelmChartCR,
		},
	}

	expressions, err := lintHelmChartsWithHelmLint(renderedFiles, tarGzFiles)
	require.NoError(t, err)
	assert.Empty(t, expressions, "aliased dependency must be scoped under its alias so the parent's colliding key is not corrupted")
}

func Test_lintHelmChartsWithHelmLint(t *testing.T) {
	tests := []struct {
		name              string
		archive           []tarChartFile
		archivePath       string
		helmChartCR       string
		expectExpressions int
		expectRule        string
	}{
		{
			name: "valid chart with builder values produces no errors",
			archive: []tarChartFile{
				chartFile("testchart/Chart.yaml", requireValueChart),
				chartFile("testchart/templates/deployment.yaml", requireValueDeployment),
			},
			archivePath:       "testchart-0.1.0.tar.gz",
			helmChartCR:       helmChartCRWithBuilder,
			expectExpressions: 0,
		},
		{
			name: "chart with no matching HelmChart CR is skipped",
			archive: []tarChartFile{
				chartFile("testchart/Chart.yaml", requireValueChart),
				chartFile("testchart/templates/deployment.yaml", requireValueDeployment),
			},
			archivePath:       "testchart-0.1.0.tar.gz",
			helmChartCR:       "", // no HelmChart CR at all
			expectExpressions: 0,
		},
		{
			name: "chart default values from values.yaml are merged with builder values",
			archive: []tarChartFile{
				chartFile("testchart/Chart.yaml", requireValueChart),
				chartFile("testchart/values.yaml", "image:\n  repository: nginx\n"),
				chartFile("testchart/templates/deployment.yaml", requireValueDeployment),
			},
			archivePath: "testchart-0.1.0.tar.gz",
			// Builder only sets tag; repository must come from chart defaults.
			helmChartCR: `apiVersion: kots.io/v1beta1
kind: HelmChart
metadata:
  name: testchart
spec:
  chart:
    name: testchart
    chartVersion: 0.1.0
  builder:
    image:
      tag: stable
`,
			expectExpressions: 0,
		},
		{
			name: "helm v2 HelmChart CR is skipped",
			archive: []tarChartFile{
				chartFile("testchart/Chart.yaml", requireValueChart),
				chartFile("testchart/templates/deployment.yaml", requireValueDeployment),
			},
			archivePath: "testchart-0.1.0.tar.gz",
			helmChartCR: `apiVersion: kots.io/v1beta1
kind: HelmChart
metadata:
  name: testchart
spec:
  chart:
    name: testchart
    chartVersion: 0.1.0
  helmVersion: v2
`,
			expectExpressions: 0,
		},
		{
			name: "chart that fails template rendering produces helm-schema-violation",
			archive: []tarChartFile{
				chartFile("testchart/Chart.yaml", requireValueChart),
				chartFile("testchart/templates/deployment.yaml", requireValueDeployment),
			},
			archivePath: "testchart-0.1.0.tar.gz",
			// HelmChart CR with no builder values, so `required` fails.
			helmChartCR: `apiVersion: kots.io/v1beta1
kind: HelmChart
metadata:
  name: testchart
spec:
  chart:
    name: testchart
    chartVersion: 0.1.0
`,
			expectExpressions: 1,
			expectRule:        "helm-schema-violation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tarGzFiles := domain.SpecFiles{
				{
					Name:    tt.archivePath,
					Path:    tt.archivePath,
					Content: buildChartArchive(t, tt.archive),
				},
			}

			renderedFiles := domain.SpecFiles{}
			if tt.helmChartCR != "" {
				renderedFiles = append(renderedFiles, domain.SpecFile{
					Name:    "helmchart.yaml",
					Path:    "helmchart.yaml",
					Content: tt.helmChartCR,
				})
			}

			expressions, err := lintHelmChartsWithHelmLint(renderedFiles, tarGzFiles)
			require.NoError(t, err)

			if tt.expectExpressions == 0 {
				assert.Empty(t, expressions)
				return
			}

			assert.GreaterOrEqual(t, len(expressions), tt.expectExpressions)
			for _, e := range expressions {
				assert.Equal(t, tt.expectRule, e.Rule)
				assert.Equal(t, tt.archivePath, e.Path)
			}
		})
	}
}
