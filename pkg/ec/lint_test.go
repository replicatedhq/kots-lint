package ec

import (
	"testing"

	"github.com/replicatedhq/kots-lint/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_LintEmbeddedClusterVersion(t *testing.T) {

	tests := []struct {
		name      string
		specFiles domain.SpecFiles
		expect    []domain.LintExpression
	}{
		{
			name: "valid version",
			specFiles: domain.SpecFiles{
				{
					Path: "",
					Content: `apiVersion: embeddedcluster.replicated.com/v1beta1
kind: Config
spec:
  version: "v1.2.2+k8s-1.29"`,
				},
			},
			expect: []domain.LintExpression{},
		},
		{
			name: "invalid version",
			specFiles: domain.SpecFiles{
				{
					Path: "",
					Content: `apiVersion: embeddedcluster.replicated.com/v1beta1
kind: Config
spec:
  version: "unlikely-to-exist"`,
				},
			},
			expect: []domain.LintExpression{
				{
					Rule:    "non-existent-ec-version",
					Type:    "error",
					Message: "Embedded Cluster version not found",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := LintEmbeddedClusterVersion(test.specFiles)
			require.NoError(t, err)
			assert.ElementsMatch(t, actual, test.expect)
		})
	}
}
