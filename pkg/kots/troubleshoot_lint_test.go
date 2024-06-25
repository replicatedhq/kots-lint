package kots

import (
	"testing"

	"github.com/replicatedhq/kots-lint/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_lintSpecHasValidYAML(t *testing.T) {
	tests := []struct {
		name   string
		spec   string
		expect []domain.LintExpression
	}{
		{
			name: "single no errors",
			spec: `apiVersion: v1
kind: ConfigMap
metadata:
  name: example_config
data:
  ENV_VAR_1: "fake"`,
			expect: []domain.LintExpression{},
		},
		{
			name: "single with errors",
			spec: `apiVersion: v1
kind: ConfigMap
metadata:
  name: example_config
data:
  ENV_VAR_1: "fake"
  ENV_VAR_2: kind: test`,
			expect: []domain.LintExpression{
				{
					Rule:    "invalid-yaml",
					Type:    "error",
					Message: "yaml: line 7: mapping values are not allowed in this context",
					Positions: []domain.LintExpressionItemPosition{
						{
							Start: domain.LintExpressionItemLinePosition{
								Line: 7,
							},
						},
					},
				},
			},
		},
		{
			name: "multi no errors",
			spec: `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: example_config
data:
  ENV_VAR_1: "fake"
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: example_config
data:
  ENV_VAR_1: "fake"`,
			expect: []domain.LintExpression{},
		},
		{
			name: "multi with errors in first",
			spec: `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: example_config
data:
  ENV_VAR_1: "fake"
  ENV_VAR_2: kind: test
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: example_config
data:
  ENV_VAR_1: "fake"`,
			expect: []domain.LintExpression{
				{
					Rule:    "invalid-yaml",
					Type:    "error",
					Message: "yaml: line 8: mapping values are not allowed in this context",
					Positions: []domain.LintExpressionItemPosition{
						{
							Start: domain.LintExpressionItemLinePosition{
								Line: 8,
							},
						},
					},
				},
			},
		},
		{
			name: "multi with errors in second",
			spec: `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: example_config
data:
  ENV_VAR_1: "fake"
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: example_config
data:
  ENV_VAR_1: "fake"
  ENV_VAR_2: kind: test`,
			expect: []domain.LintExpression{
				{
					Rule:    "invalid-yaml",
					Type:    "error",
					Message: "yaml: line 15: mapping values are not allowed in this context",
					Positions: []domain.LintExpressionItemPosition{
						{
							Start: domain.LintExpressionItemLinePosition{
								Line: 15,
							},
						},
					},
				},
			},
		},
		{
			name: "proxy",
			spec: `apiVersion: v1
kind: ConfigMap
metadata:
  name: proxy
data:
  HTTP_PROXY: "{{repl HTTPProxy }}"
  NO_PROXY: "{{repl NoProxy }}"`,
			expect: []domain.LintExpression{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := lintSpecHasValidYAML(test.spec)
			assert.ElementsMatch(t, actual, test.expect)
		})
	}
}

func Test_lintSpecWithKubeval(t *testing.T) {
	tests := []struct {
		name   string
		spec   string
		expect []domain.LintExpression
	}{
		{
			name: "preflight-no-errors",
			spec: `apiVersion: troubleshoot.replicated.com/v1beta1
kind: Preflight
metadata:
  name: minimum-node-count
spec:
  collectors: []
  analyzers:
    - nodeResources:
        checkName: Must have at least 3 nodes in the cluster
        outcomes:
          - fail:
              when: count() < 3
              message: This application requires at least 3 nodes
          - warn:
              when: count() < 6
              message: This application recommends at last 6 nodes.
          - pass:
              message: This cluster has enough nodes.`,
			expect: []domain.LintExpression{},
		},
		{
			name: "preflight-type-warning",
			spec: `apiVersion: troubleshoot.replicated.com/v1beta1
kind: Preflight
metadata:
  name: minimum-node-count
spec:
  collectors: []
  analyzers:
    - nodeResources:
        checkName: Must have at least 3 nodes in the cluster
        outcomes:
          - fail:
              when: 3
              message: This application requires at least 3 nodes
          - warn:
              when: count() < 6
              message: This application recommends at last 6 nodes.
          - pass:
              message: This cluster has enough nodes.`,
			expect: []domain.LintExpression{
				{
					Rule:    "invalid_type",
					Type:    "warn",
					Message: "Invalid type. Expected: string, given: integer",
					Positions: []domain.LintExpressionItemPosition{
						{
							Start: domain.LintExpressionItemLinePosition{
								Line: 12,
							},
						},
					},
				},
			},
		},
		{
			name: "supportbundle-no-errors",
			spec: `apiVersion: troubleshoot.replicated.com/v1beta1
kind: SupportBundle
metadata:
  name: minimum-node-count
spec:
  collectors: []
  analyzers:
    - nodeResources:
        checkName: Must have at least 3 nodes in the cluster
        outcomes:
          - fail:
              when: count() < 3
              message: This application requires at least 3 nodes
          - warn:
              when: count() < 6
              message: This application recommends at last 6 nodes.
          - pass:
              message: This cluster has enough nodes.`,
			expect: []domain.LintExpression{},
		},
		{
			name: "supportbundle-type-warning",
			spec: `apiVersion: troubleshoot.replicated.com/v1beta1
kind: SupportBundle
metadata:
  name: minimum-node-count
spec:
  collectors: []
  analyzers:
    - nodeResources:
        checkName: Must have at least 3 nodes in the cluster
        outcomes:
          - fail:
              when: count() < 3
              message: This application requires at least 3 nodes
          - warn:
              when: 6
              message: This application recommends at last 6 nodes.
          - pass:
              message: This cluster has enough nodes.`,
			expect: []domain.LintExpression{
				{
					Rule:    "invalid_type",
					Type:    "warn",
					Message: "Invalid type. Expected: string, given: integer",
					Positions: []domain.LintExpressionItemPosition{
						{
							Start: domain.LintExpressionItemLinePosition{
								Line: 15,
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := lintSpecWithKubevalSchema(test.spec, "file://../../kubernetes_json_schema/schema")
			require.NoError(t, err)
			assert.ElementsMatch(t, actual, test.expect)
		})
	}
}
