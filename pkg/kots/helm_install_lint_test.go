package kots

import (
	"context"
	"testing"

	"github.com/replicatedhq/kots-lint/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_lintHelmInstallType(t *testing.T) {
	tests := []struct {
		name              string
		specFiles         domain.SpecFiles
		expectErr         bool
		expectExpressions int
		expectRule        string
		expectPaths       []string
	}{
		{
			name:              "empty spec files returns no expressions",
			specFiles:         domain.SpecFiles{},
			expectExpressions: 0,
		},
		{
			name: "replicated api version resource is skipped",
			specFiles: domain.SpecFiles{
				{
					Name:    "installer.yaml",
					Path:    "installer.yaml",
					Content: "apiVersion: kots.io/v1beta1\nkind: Installer\n",
				},
			},
			expectExpressions: 0,
		},
		{
			name: "embeddedcluster replicated api version resource is skipped",
			specFiles: domain.SpecFiles{
				{
					Name:    "install.yaml",
					Path:    "install.yaml",
					Content: "apiVersion: embeddedcluster.replicated.com/v1beta1\nkind: Installer\n",
				},
			},
			expectExpressions: 0,
		},
		{
			name: "cluster.kurl.sh api version resource is skipped",
			specFiles: domain.SpecFiles{
				{
					Name:    "installer.yaml",
					Path:    "installer.yaml",
					Content: "apiVersion: cluster.kurl.sh/v1beta1\nkind: Installer\n",
				},
			},
			expectExpressions: 0,
		},
		{
			name: "troubleshoot.sh api version resource is skipped",
			specFiles: domain.SpecFiles{
				{
					Name:    "preflight.yaml",
					Path:    "preflight.yaml",
					Content: "apiVersion: troubleshoot.sh/v1beta2\nkind: Preflight\n",
				},
			},
			expectExpressions: 0,
		},
		{
			name: "resource with kots.io/installer-only annotation is skipped",
			specFiles: domain.SpecFiles{
				{
					Name: "deployment.yaml",
					Path: "deployment.yaml",
					Content: `apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    kots.io/installer-only: "true"
`,
				},
			},
			expectExpressions: 0,
		},
		{
			name: "resource without annotation returns lint error",
			specFiles: domain.SpecFiles{
				{
					Name: "deployment.yaml",
					Path: "deployment.yaml",
					Content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
`,
				},
			},
			expectExpressions: 1,
			expectRule:        "helm-install-type-missing-annotation",
			expectPaths:       []string{"deployment.yaml"},
		},
		{
			name: "resource without annotation with embeddedcluster api version but different group produces error",
			specFiles: domain.SpecFiles{
				{
					Name: "endpoints.yaml",
					Path: "endpoints.yaml",
					Content: `apiVersion: apps/v1
kind: Endpoints
metadata:
  name: test
`,
				},
			},
			expectExpressions: 1,
			expectRule:        "helm-install-type-missing-annotation",
			expectPaths:       []string{"endpoints.yaml"},
		},
		{
			name: "files in subdirectories deeper than root are skipped",
			specFiles: domain.SpecFiles{
				{
					Name: "deployment.yaml",
					Path: "charts/subchart/deployment.yaml",
					Content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
`,
				},
			},
			expectExpressions: 0,
		},
		{
			name: "file in single subdirectory is still checked",
			specFiles: domain.SpecFiles{
				{
					Name: "deployment.yaml",
					Path: "base/deployment.yaml",
					Content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
`,
				},
			},
			expectExpressions: 1,
			expectRule:        "helm-install-type-missing-annotation",
			expectPaths:       []string{"base/deployment.yaml"},
		},
		{
			name: "multiple docs in a single file are separated and each checked",
			specFiles: domain.SpecFiles{
				{
					Name: "resources.yaml",
					Path: "resources.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Config
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
`,
				},
			},
			expectExpressions: 1,
			expectRule:        "helm-install-type-missing-annotation",
			expectPaths:       []string{"resources.yaml"},
		},
		{
			name: "mixed annotated and unannotated resources",
			specFiles: domain.SpecFiles{
				{
					Name: "good.yaml",
					Path: "good.yaml",
					Content: `apiVersion: apps/v1
kind: Service
metadata:
  annotations:
    kots.io/installer-only: "true"
`,
				},
				{
					Name: "bad.yaml",
					Path: "bad.yaml",
					Content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
`,
				},
				{
					Name: "replicated.yaml",
					Path: "replicated.yaml",
					Content: "apiVersion: kots.io/v1beta1\nkind: Config\n",
				},
			},
			expectExpressions: 1,
			expectRule:        "helm-install-type-missing-annotation",
			expectPaths:       []string{"bad.yaml"},
		},
		{
			name: "trailing separator produces empty doc that is skipped",
			specFiles: domain.SpecFiles{
				{
					Name: "resources.yaml",
					Path: "resources.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Config
---
`,
				},
			},
			expectExpressions: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			expressions, err := lintHelmInstallType(ctx, tt.specFiles)

			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Len(t, expressions, tt.expectExpressions)

			if tt.expectRule != "" {
				for _, e := range expressions {
					assert.Equal(t, tt.expectRule, e.Rule)
					assert.Equal(t, "error", e.Type)
					assert.NotEmpty(t, e.Message)
				}
			}

			if len(tt.expectPaths) > 0 {
				paths := make([]string, len(expressions))
				for i, e := range expressions {
					paths[i] = e.Path
				}
				assert.ElementsMatch(t, tt.expectPaths, paths)
			}
		})
	}
}

func Test_lintHelmInstallType_contextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	specFiles := domain.SpecFiles{
		{
			Name:    "deployment.yaml",
			Path:    "deployment.yaml",
			Content: "apiVersion: apps/v1\nkind: Deployment\n",
		},
	}

	_, err := lintHelmInstallType(ctx, specFiles)
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}
