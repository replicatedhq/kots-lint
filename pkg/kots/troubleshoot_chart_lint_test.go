package kots

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"testing"

	"github.com/replicatedhq/kots-lint/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func chartFile(path, content string) tarChartFile {
	return tarChartFile{Path: path, Content: content}
}

type tarChartFile struct {
	Path    string
	Content string
}

// buildChartArchive returns a base64-encoded tar.gz archive of the given files.
func buildChartArchive(t *testing.T, files []tarChartFile) string {
	t.Helper()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	for _, f := range files {
		hdr := &tar.Header{
			Name: f.Path,
			Mode: 0644,
			Size: int64(len(f.Content)),
		}
		require.NoError(t, tw.WriteHeader(hdr))
		_, err := tw.Write([]byte(f.Content))
		require.NoError(t, err)
	}

	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())

	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

const minimalChartYAML = `apiVersion: v2
name: testchart
version: 0.1.0
`

const preflightCRTemplate = `apiVersion: troubleshoot.sh/v1beta2
kind: Preflight
metadata:
  name: my-preflight
spec:
  analyzers: []
`

const supportBundleCRTemplate = `apiVersion: troubleshoot.sh/v1beta2
kind: SupportBundle
metadata:
  name: my-sb
spec:
  collectors: []
`

const preflightSecretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: my-preflight
  labels:
    troubleshoot.sh/kind: preflight
stringData:
  preflight.yaml: |
    apiVersion: troubleshoot.sh/v1beta2
    kind: Preflight
    metadata:
      name: my-preflight
    spec:
      analyzers: []
`

const preflightCRD = `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: preflights.troubleshoot.sh
spec:
  group: troubleshoot.sh
  names:
    kind: Preflight
    plural: preflights
  scope: Namespaced
  versions: []
`

const supportBundleCRD = `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: supportbundles.troubleshoot.sh
spec:
  group: troubleshoot.sh
  names:
    kind: SupportBundle
    plural: supportbundles
  scope: Namespaced
  versions: []
`

func Test_lintHelmChartTroubleshootCRDs(t *testing.T) {
	tests := []struct {
		name        string
		archive     []tarChartFile
		expectKinds []string // kinds we expect warnings for
	}{
		{
			name: "preflight CR in chart without CRD warns",
			archive: []tarChartFile{
				chartFile("testchart/Chart.yaml", minimalChartYAML),
				chartFile("testchart/templates/preflight.yaml", preflightCRTemplate),
			},
			expectKinds: []string{"Preflight"},
		},
		{
			name: "supportbundle CR in chart without CRD warns",
			archive: []tarChartFile{
				chartFile("testchart/Chart.yaml", minimalChartYAML),
				chartFile("testchart/templates/sb.yaml", supportBundleCRTemplate),
			},
			expectKinds: []string{"SupportBundle"},
		},
		{
			name: "preflight CR with CRD in crds/ does not warn",
			archive: []tarChartFile{
				chartFile("testchart/Chart.yaml", minimalChartYAML),
				chartFile("testchart/crds/preflight.yaml", preflightCRD),
				chartFile("testchart/templates/preflight.yaml", preflightCRTemplate),
			},
			expectKinds: nil,
		},
		{
			name: "preflight CR with CRD in templates/ does not warn",
			archive: []tarChartFile{
				chartFile("testchart/Chart.yaml", minimalChartYAML),
				chartFile("testchart/templates/preflight-crd.yaml", preflightCRD),
				chartFile("testchart/templates/preflight.yaml", preflightCRTemplate),
			},
			expectKinds: nil,
		},
		{
			name: "preflight in Secret does not warn",
			archive: []tarChartFile{
				chartFile("testchart/Chart.yaml", minimalChartYAML),
				chartFile("testchart/templates/preflight-secret.yaml", preflightSecretTemplate),
			},
			expectKinds: nil,
		},
		{
			name: "both CRs without CRDs warn for each",
			archive: []tarChartFile{
				chartFile("testchart/Chart.yaml", minimalChartYAML),
				chartFile("testchart/templates/preflight.yaml", preflightCRTemplate),
				chartFile("testchart/templates/sb.yaml", supportBundleCRTemplate),
			},
			expectKinds: []string{"Preflight", "SupportBundle"},
		},
		{
			name: "preflight has CRD but supportbundle does not warns only for supportbundle",
			archive: []tarChartFile{
				chartFile("testchart/Chart.yaml", minimalChartYAML),
				chartFile("testchart/crds/preflight.yaml", preflightCRD),
				chartFile("testchart/templates/preflight.yaml", preflightCRTemplate),
				chartFile("testchart/templates/sb.yaml", supportBundleCRTemplate),
			},
			expectKinds: []string{"SupportBundle"},
		},
		{
			name: "supportbundle has CRD in templates does not warn",
			archive: []tarChartFile{
				chartFile("testchart/Chart.yaml", minimalChartYAML),
				chartFile("testchart/templates/sb-crd.yaml", supportBundleCRD),
				chartFile("testchart/templates/sb.yaml", supportBundleCRTemplate),
			},
			expectKinds: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archive := buildChartArchive(t, tt.archive)
			tarGzFiles := domain.SpecFiles{
				{
					Name:    "testchart-0.1.0.tar.gz",
					Path:    "testchart-0.1.0.tar.gz",
					Content: archive,
				},
			}

			expressions, err := lintHelmChartTroubleshootCRDs(context.Background(), tarGzFiles)
			require.NoError(t, err)

			actualKinds := []string{}
			for _, e := range expressions {
				assert.Equal(t, "warn", e.Type)
				assert.Equal(t, "troubleshoot-spec-in-chart-without-crd", e.Rule)
				assert.Equal(t, "testchart-0.1.0.tar.gz", e.Path)
				switch {
				case containsKind(e.Message, "Preflight"):
					actualKinds = append(actualKinds, "Preflight")
				case containsKind(e.Message, "SupportBundle"):
					actualKinds = append(actualKinds, "SupportBundle")
				}
			}
			assert.ElementsMatch(t, tt.expectKinds, actualKinds)
		})
	}
}

func containsKind(message, kind string) bool {
	return bytes.Contains([]byte(message), []byte("contains a "+kind+" custom resource"))
}
