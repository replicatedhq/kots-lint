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
			name: "v3 preflight with correct v1beta3 apiVersion",
			specFiles: domain.SpecFiles{
				{
					Path: "ec-config.yaml",
					Content: `apiVersion: embeddedcluster.replicated.com/v1beta1
kind: Config
spec:
  version: "3.0.0-alpha-31+k8s-1.34"`,
				},
				{
					Path: "preflight.yaml",
					Content: `apiVersion: troubleshoot.sh/v1beta3
kind: Preflight
metadata:
  name: preflight-sample
spec:
  analyzers: []`,
				},
			},
			expect:    []domain.LintExpression{},
			apiResult: []byte(`{}`),
		},
		{
			name: "v3 preflight with wrong v1beta2 apiVersion",
			specFiles: domain.SpecFiles{
				{
					Path: "ec-config.yaml",
					Content: `apiVersion: embeddedcluster.replicated.com/v1beta1
kind: Config
spec:
  version: "3.0.0-alpha-31+k8s-1.34"`,
				},
				{
					Path: "preflight.yaml",
					Content: `apiVersion: troubleshoot.sh/v1beta2
kind: Preflight
metadata:
  name: preflight-sample
spec:
  analyzers: []`,
				},
			},
			expect: []domain.LintExpression{
				{
					Rule:    "ec-v3-preflight-api-version",
					Type:    "error",
					Path:    "preflight.yaml",
					Message: "Preflight spec must use apiVersion troubleshoot.sh/v1beta3 with Embedded Cluster v3",
				},
			},
			apiResult: []byte(`{}`),
		},
		{
			name: "v3 with no preflight",
			specFiles: domain.SpecFiles{
				{
					Path: "ec-config.yaml",
					Content: `apiVersion: embeddedcluster.replicated.com/v1beta1
kind: Config
spec:
  version: "3.0.0-alpha-31+k8s-1.34"`,
				},
			},
			expect:    []domain.LintExpression{},
			apiResult: []byte(`{}`),
		},
		{
			name: "v3 preflight with no apiVersion",
			specFiles: domain.SpecFiles{
				{
					Path: "ec-config.yaml",
					Content: `apiVersion: embeddedcluster.replicated.com/v1beta1
kind: Config
spec:
  version: "3.0.0-alpha-31+k8s-1.34"`,
				},
				{
					Path: "preflight.yaml",
					Content: `kind: Preflight
metadata:
  name: preflight-sample
spec:
  analyzers: []`,
				},
			},
			expect: []domain.LintExpression{
				{
					Rule:    "ec-v3-preflight-api-version",
					Type:    "error",
					Path:    "preflight.yaml",
					Message: "Preflight spec must use apiVersion troubleshoot.sh/v1beta3 with Embedded Cluster v3",
				},
			},
			apiResult: []byte(`{}`),
		},
		{
			name: "v2 preflight with v1beta2 apiVersion no error",
			specFiles: domain.SpecFiles{
				{
					Path: "ec-config.yaml",
					Content: `apiVersion: embeddedcluster.replicated.com/v1beta1
kind: Config
spec:
  version: "v2.0.0+k8s-1.29"`,
				},
				{
					Path: "preflight.yaml",
					Content: `apiVersion: troubleshoot.sh/v1beta2
kind: Preflight
metadata:
  name: preflight-sample
spec:
  analyzers: []`,
				},
			},
			expect:    []domain.LintExpression{},
			apiResult: []byte(`{}`),
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
