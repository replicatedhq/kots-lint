package ec

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/replicatedhq/kots-lint/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Lint(t *testing.T) {

	tests := []struct {
		name      string
		specFiles domain.SpecFiles
		expect    []domain.LintExpression
		apiResult []byte
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
			expect:    []domain.LintExpression{},
			apiResult: []byte(`{}`),
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
		{
			name: "pre-release version",
			specFiles: domain.SpecFiles{
				{
					Path: "",
					Content: `apiVersion: embeddedcluster.replicated.com/v1beta1
kind: Config
spec:
  version: "pre-release-version"`,
				},
			},
			expect: []domain.LintExpression{
				{
					Rule:    "non-existent-ec-version",
					Type:    "error",
					Message: "Embedded Cluster version is a pre-release",
				},
			},
			apiResult: []byte(`{"prerelease": true}`),
		},
		{
			name: "ec v3 version skips github check",
			specFiles: domain.SpecFiles{
				{
					Path: "cluster-config.yaml",
					Content: `apiVersion: embeddedcluster.replicated.com/v1beta1
kind: Config
spec:
  version: "3.0.0+k8s-1.34"`,
				},
			},
			expect:    []domain.LintExpression{},
			apiResult: nil, // server returns 404; must not be reached
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oldURL := githubAPIURL
			oldVersions := ecVersions
			defer func() {
				githubAPIURL = oldURL
				ecVersions = oldVersions
			}()
			ecVersions = make(map[string]EmbeddedClusterVersion)

			server := httptest.NewServer(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						if test.apiResult == nil {
							w.WriteHeader(http.StatusNotFound)
							return
						}
						w.Header().Set("Content-Type", "application/json")
						w.Write(test.apiResult)
					},
				),
			)
			defer server.Close()

			githubAPIURL = server.URL
			actual, err := Lint(test.specFiles)
			require.NoError(t, err)
			assert.ElementsMatch(t, actual, test.expect)
		})
	}
}
