package kurl

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/replicatedhq/kots-lint/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kurllint "github.com/replicatedhq/kurlkinds/pkg/lint"
)

func Test_lintKurlInstaller(t *testing.T) {
	tests := []struct {
		name      string
		specFiles domain.SpecFiles
		expect    []domain.LintExpression
	}{
		{
			name: "deployment file",
			specFiles: domain.SpecFiles{
				{
					Name: "deployment.yaml",
					Path: "deployment.yaml",
					Content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: example-nginx
  labels:
    app: example
spec:
  replicas: 1
  selector:
    matchLabels:
      app: example
  template:
    metadata:
      labels:
        app: example
    spec:
      containers:
        - image: nginx`,
				},
			},
		},
		{
			name: "multiple declarations",
			specFiles: domain.SpecFiles{
				{
					Name: "deployment.yaml",
					Path: "deployment.yaml",
					Content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: example-nginx
  labels:
    app: example
spec:
  replicas: 1
  selector:
    matchLabels:
      app: example
  template:
    metadata:
      labels:
        app: example
    spec:
      containers:
        - image: nginx
---
apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  containers:
  - name: nginx
    image: nginx:1.14.2
    ports:
    - containerPort: 80`,
				},
			},
		},
		{
			name: "valid installer",
			specFiles: domain.SpecFiles{
				{
					Name: "installer.yaml",
					Path: "installer.yaml",
					Content: `apiVersion: cluster.kurl.sh/v1beta1
kind: Installer
metadata:
  name: latest
spec:
  kubernetes:
    version: latest
  containerd:
    version: latest
  weave:
    version: latest
`,
				},
			},
		},
		{
			name: "invalid installer",
			specFiles: domain.SpecFiles{
				{
					Name: "installer.yaml",
					Path: "installer.yaml",
					Content: `apiVersion: cluster.kurl.sh/v1beta1
kind: Installer
metadata:
  name: latest
spec:
  kubernetes:
    version: latest
  weave:
    version: latest`,
				},
			},
			expect: []domain.LintExpression{
				{
					Rule:    "kubernetes-installer-misconfiguration",
					Type:    "error",
					Path:    "installer.yaml",
					Message: "No container runtime (Docker or Containerd) selected",
				},
			},
		},
		{
			name: "multiple invalid installers in a single file",
			specFiles: domain.SpecFiles{
				{
					Name: "installer.yaml",
					Path: "installer.yaml",
					Content: `apiVersion: cluster.kurl.sh/v1beta1
kind: Installer
metadata:
  name: latest
spec:
  kubernetes:
    version: latest
  weave:
    version: latest
---
apiVersion: cluster.kurl.sh/v1beta1
kind: Installer
metadata:
  name: latest
spec:
  kubernetes:
    version: latest
  containerd:
    version: latest`,
				},
			},
			expect: []domain.LintExpression{
				{
					Rule:    "kubernetes-installer-misconfiguration",
					Type:    "error",
					Path:    "installer.yaml",
					Message: "No container runtime (Docker or Containerd) selected",
				},
				{
					Rule:    "kubernetes-installer-misconfiguration",
					Type:    "error",
					Path:    "installer.yaml",
					Message: "No CNI plugin (Flannel, Weave or Antrea) selected",
				},
			},
		},
		{
			name: "multiple invalid installers",
			specFiles: domain.SpecFiles{
				{
					Name: "installer.yaml",
					Path: "installer.yaml",
					Content: `apiVersion: cluster.kurl.sh/v1beta1
kind: Installer
metadata:
  name: latest
spec:
  kubernetes:
    version: latest
  weave:
    version: latest`,
				},
				{
					Name: "installer-2.yaml",
					Path: "installer-2.yaml",
					Content: `apiVersion: cluster.kurl.sh/v1beta1
kind: Installer
metadata:
  name: latest
spec:
  kubernetes:
    version: latest
  containerd:
    version: 8.8.8
  weave:
    version: latest`,
				},
			},
			expect: []domain.LintExpression{
				{
					Rule:    "kubernetes-installer-misconfiguration",
					Type:    "error",
					Path:    "installer.yaml",
					Message: "No container runtime (Docker or Containerd) selected",
				},
				{
					Rule:    "kubernetes-installer-unknown-addon",
					Type:    "error",
					Path:    "installer-2.yaml",
					Message: "Unknown containerd add-on version 8.8.8",
				},
			},
		},
	}

	versions := map[string][]string{
		"kubernetes": {
			"latest",
			"1.25.2",
			"1.25.1",
		},
		"weave": {
			"latest",
			"2.6.5",
			"2.6.4",
			"2.5.2",
		},
		"containerd": {
			"latest",
			"1.6.8",
			"1.6.7",
			"1.6.6",
		},
	}

	mocksrv := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("content-type", "application/json")
				if err := json.NewEncoder(w).Encode(versions); err != nil {
					t.Fatalf("unexpected marshal error: %s", err)
				}
			},
		),
	)
	defer mocksrv.Close()

	u, err := url.Parse(mocksrv.URL)
	if err != nil {
		t.Fatalf("unable to parse test url: %s", err)
	}
	linter := kurllint.New(kurllint.WithAPIBaseURL(u))
	kurlLinter := &KurlLinter{
		Linter: linter,
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := kurlLinter.LintKurlInstaller(test.specFiles)
			require.NoError(t, err)
			assert.ElementsMatch(t, actual, test.expect)
		})
	}
}
