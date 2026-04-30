//go:build integration

// Package integration runs HTTP-level tests against a running kots-lint
// service (typically the docker image started by scripts/integration-test.sh).
// Each table-driven case POSTs a fixture to one of the public endpoints and
// asserts that an expected set of lint rule IDs is (or is not) produced.
//
// Run against an already-running service:
//
//	KOTS_LINT_BASE_URL=http://localhost:8082 \
//	  go test -tags integration ./test/integration/...
package integration

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// buildChartTgz returns a base64-encoded helm chart tarball containing the
// provided file map. Used inline so we can construct chart archives that
// trigger specific lint rules without committing binary fixtures.
func buildChartTgz(t *testing.T, files map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for path, content := range files {
		if err := tw.WriteHeader(&tar.Header{Name: path, Mode: 0644, Size: int64(len(content))}); err != nil {
			t.Fatalf("tar header: %v", err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("tar write: %v", err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gz close: %v", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

//go:embed testdata/testchart-with-labels-16.2.2.tgz
var testChartWithLabels []byte

//go:embed testdata/not-a-chart.tgz
var notAChart []byte

type lintExpression struct {
	Rule    string `json:"rule"`
	Type    string `json:"type"`
	Message string `json:"message"`
	Path    string `json:"path"`
}

type lintResponse struct {
	LintExpressions   []lintExpression `json:"lintExpressions"`
	IsLintingComplete bool             `json:"isLintingComplete"`
}

type specFile struct {
	Name            string `json:"name"`
	Path            string `json:"path"`
	Content         string `json:"content"`
	AllowDuplicates bool   `json:"allowDuplicates"`
}

func baseURL(t *testing.T) string {
	t.Helper()
	if u := os.Getenv("KOTS_LINT_BASE_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return "http://localhost:8082"
}

func postJSON(t *testing.T, url string, body any) lintResponse {
	t.Helper()
	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("post %s: %v", url, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST %s returned %d: %s", url, resp.StatusCode, string(respBody))
	}
	var lr lintResponse
	if err := json.Unmarshal(respBody, &lr); err != nil {
		t.Fatalf("decode response: %v\nbody: %s", err, string(respBody))
	}
	return lr
}

// lintFiles posts the given spec files to /v1/lint as a JSON SpecFiles array.
func lintFiles(t *testing.T, files []specFile) lintResponse {
	t.Helper()
	specJSON, err := json.Marshal(files)
	if err != nil {
		t.Fatalf("marshal specfiles: %v", err)
	}
	return postJSON(t, baseURL(t)+"/v1/lint", map[string]string{"spec": string(specJSON)})
}

// troubleshootLint posts a single yaml document to /v1/troubleshoot-lint.
func troubleshootLint(t *testing.T, yaml string) lintResponse {
	t.Helper()
	return postJSON(t, baseURL(t)+"/v1/troubleshoot-lint", map[string]string{"spec": yaml})
}

func ruleSet(exprs []lintExpression) map[string]bool {
	m := make(map[string]bool, len(exprs))
	for _, e := range exprs {
		m[e.Rule] = true
	}
	return m
}

// assertRules checks that every rule in want is present in got, and no rule in
// forbid is present. Extra rules are ignored — the suite is intentionally
// non-strict because OPA evaluation may add unrelated info-level findings as
// rules evolve.
func assertRules(t *testing.T, got []lintExpression, want, forbid []string) {
	t.Helper()
	have := ruleSet(got)
	for _, w := range want {
		if !have[w] {
			t.Errorf("expected rule %q to fire; got rules: %v", w, sortedKeys(have))
		}
	}
	for _, f := range forbid {
		if have[f] {
			t.Errorf("expected rule %q NOT to fire; full output: %s", f, mustJSON(got))
		}
	}
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func mustJSON(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

// TestLivez is a smoke test that the service is reachable. It runs first so
// failures point at infrastructure rather than at lint logic.
func TestLivez(t *testing.T) {
	resp, err := http.Get(baseURL(t) + "/livez")
	if err != nil {
		t.Fatalf("GET /livez: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /livez = %d", resp.StatusCode)
	}
}

// TestLint exercises /v1/lint. Each case is grouped to trigger a related set
// of rules; this keeps the fixture count manageable while still asserting one
// concrete trigger per rule.
func TestLint(t *testing.T) {
	cases := []struct {
		name   string
		files  []specFile
		want   []string
		forbid []string
	}{
		{
			// Empty release: every "missing X spec" warning should fire.
			name:  "empty_release_emits_missing_spec_warnings",
			files: []specFile{},
			want: []string{
				"application-spec",
				"config-spec",
				"preflight-spec",
				"troubleshoot-spec",
			},
		},
		{
			name: "missing_kind_and_api_version",
			files: []specFile{{
				Name:    "broken.yaml",
				Path:    "broken.yaml",
				Content: "metadata:\n  name: no-kind-or-apiversion\n",
			}},
			want: []string{"missing-kind-field", "missing-api-version-field"},
		},
		{
			name: "invalid_yaml",
			files: []specFile{{
				Name:    "bad.yaml",
				Path:    "bad.yaml",
				Content: "apiVersion: v1\nkind: ConfigMap\n  this: is: not: yaml\n",
			}},
			want: []string{"invalid-yaml"},
		},
		{
			name: "kots_application_missing_icon_and_status_informers",
			files: []specFile{{
				Name: "kots-app.yaml",
				Path: "kots-app.yaml",
				Content: `apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: my-app
spec:
  title: My App
`,
			}},
			want:   []string{"application-icon", "application-statusInformers"},
			forbid: []string{"application-spec"},
		},
		{
			name: "kots_application_invalid_target_and_min_versions",
			files: []specFile{{
				Name: "kots-app.yaml",
				Path: "kots-app.yaml",
				Content: `apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: my-app
spec:
  title: My App
  icon: https://example.com/icon.png
  statusInformers:
    - deployment/my-app
  targetKotsVersion: "not-a-version"
  minKotsVersion: "also-bad"
`,
			}},
			want: []string{"invalid-target-kots-version", "invalid-min-kots-version"},
		},
		{
			// The OPA rules for `privileged` and `allow-privilege-escalation`
			// match `spec.privileged` / `spec.allowPrivilegeEscalation` at any
			// of the spec-harvest levels (1/2/3 deep), not the container
			// `securityContext`. We model them at the pod-spec level here.
			name: "container_kitchen_sink_security_and_resources",
			files: []specFile{{
				Name: "deployment.yaml",
				Path: "deployment.yaml",
				Content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
  namespace: hardcoded-ns
spec:
  replicas: 1
  selector:
    matchLabels:
      app: web
  template:
    metadata:
      labels:
        app: web
    spec:
      privileged: true
      allowPrivilegeEscalation: true
      containers:
        - name: web
          image: nginx:latest
      volumes:
        - name: docker-sock
          hostPath:
            path: /var/run/docker.sock
`,
			}},
			want: []string{
				"replicas-1",
				"privileged",
				"allow-privilege-escalation",
				"container-image-latest-tag",
				"container-resources",
				"volumes-host-paths",
				"volume-docker-sock",
				"hardcoded-namespace",
			},
		},
		{
			name: "container_partial_resources_missing_subfields",
			files: []specFile{{
				Name: "deployment.yaml",
				Path: "deployment.yaml",
				Content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
  namespace: '{{repl Namespace }}'
spec:
  replicas: 2
  selector:
    matchLabels:
      app: web
  template:
    metadata:
      labels:
        app: web
    spec:
      containers:
        - name: web
          image: nginx:1.25
          resources:
            limits: {}
            requests: {}
`,
			}},
			want: []string{
				"resource-limits-cpu",
				"resource-limits-memory",
				"resource-requests-cpu",
				"resource-requests-memory",
			},
			forbid: []string{
				"hardcoded-namespace",
				"container-image-latest-tag",
				"replicas-1",
				"container-resources",
			},
		},
		{
			name: "may_contain_secrets",
			files: []specFile{{
				Name:    "creds.yaml",
				Path:    "creds.yaml",
				Content: "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: creds\ndata:\n  password: hunter2\n",
			}},
			want: []string{"may-contain-secrets"},
		},
		{
			name: "duplicate_kots_kind",
			files: []specFile{
				{
					Name: "app1.yaml", Path: "app1.yaml",
					Content: "apiVersion: kots.io/v1beta1\nkind: Application\nmetadata:\n  name: a\nspec:\n  title: A\n  icon: https://e.x/i.png\n  statusInformers: [deployment/x]\n",
				},
				{
					Name: "app2.yaml", Path: "app2.yaml",
					Content: "apiVersion: kots.io/v1beta1\nkind: Application\nmetadata:\n  name: b\nspec:\n  title: B\n  icon: https://e.x/i.png\n  statusInformers: [deployment/y]\n",
				},
			},
			want: []string{"duplicate-kots-kind"},
		},
		{
			name: "helm_chart_invalid_and_duplicate_release_names",
			files: []specFile{
				{
					Name: "chart-bad.yaml", Path: "chart-bad.yaml",
					Content: `apiVersion: kots.io/v1beta2
kind: HelmChart
metadata:
  name: bad
spec:
  chart:
    name: bad
    chartVersion: "1.0.0"
  releaseName: "Bad_Release_Name!"
`,
				},
				{
					Name: "chart-a.yaml", Path: "chart-a.yaml",
					Content: `apiVersion: kots.io/v1beta2
kind: HelmChart
metadata:
  name: a
spec:
  chart:
    name: a
    chartVersion: "1.0.0"
  releaseName: shared-name
`,
				},
				{
					Name: "chart-b.yaml", Path: "chart-b.yaml",
					Content: `apiVersion: kots.io/v1beta2
kind: HelmChart
metadata:
  name: b
spec:
  chart:
    name: b
    chartVersion: "1.0.0"
  releaseName: shared-name
`,
				},
			},
			want: []string{"invalid-helm-release-name", "duplicate-helm-release-name"},
		},
		{
			name: "config_option_invalid_type_and_password_naming",
			files: []specFile{{
				Name: "config.yaml", Path: "config.yaml",
				Content: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: my-config
spec:
  groups:
    - name: g1
      title: Group 1
      items:
        - name: api_password
          title: API Password
          type: text
        - name: weird
          title: Weird
          type: not_a_real_type
`,
			}},
			want: []string{"config-option-invalid-type", "config-option-password-type"},
		},
		{
			name: "config_option_not_found_and_circular",
			files: []specFile{{
				Name: "config.yaml", Path: "config.yaml",
				Content: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: my-config
spec:
  groups:
    - name: g1
      title: Group 1
      items:
        - name: self_ref
          title: Self
          type: text
          default: 'repl{{ ConfigOption "self_ref" }}'
        - name: dangling
          title: Dangling
          type: text
          default: 'repl{{ ConfigOption "does_not_exist" }}'
`,
			}},
			want: []string{"config-option-is-circular", "config-option-not-found"},
		},
		{
			name: "config_option_when_and_regex_validators",
			files: []specFile{{
				Name: "config.yaml", Path: "config.yaml",
				Content: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: my-config
spec:
  groups:
    - name: g1
      title: Group 1
      items:
        - name: when_invalid
          title: When Invalid
          type: text
          when: "this is not a boolean or template"
        - name: bad_regex
          title: Bad Regex
          type: text
          validation:
            regex:
              pattern: "([a-z"
              message: "must match"
        - name: regex_on_bool
          title: Regex On Bool
          type: bool
          validation:
            regex:
              pattern: "^yes$"
              message: "yes only"
`,
			}},
			want: []string{
				"config-option-when-is-invalid",
				"config-option-invalid-regex-validator",
				"config-option-regex-validator-invalid-type",
			},
		},
		{
			name: "config_option_repeatable_misconfigurations",
			files: []specFile{{
				Name: "config.yaml", Path: "config.yaml",
				Content: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: my-config
spec:
  groups:
    - name: g1
      title: Group 1
      items:
        - name: repeat_me
          title: Repeat Me
          type: text
          repeatable: true
          # missing templates, valuesByGroup, and yamlPath ends without []
`,
			}},
			want: []string{
				"repeat-option-missing-template",
				"repeat-option-missing-valuesByGroup",
			},
		},
		{
			name: "backup_resource_required_when_restore_exists",
			files: []specFile{{
				Name: "restore.yaml", Path: "restore.yaml",
				Content: `apiVersion: velero.io/v1
kind: Restore
metadata:
  name: restore-only
spec:
  backupName: missing
`,
			}},
			want: []string{"backup-resource-required-when-restore-exists"},
		},
		{
			// Schema-driven warnings produced by kubeval. Replicas should be
			// an integer; we set a bad type plus an unknown field plus omit
			// the required `selector` to trigger three distinct kubeval rules.
			name: "kubeval_schema_violations",
			files: []specFile{{
				Name: "deployment.yaml", Path: "deployment.yaml",
				Content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: kubeval-test
spec:
  replicas: "not-a-number"
  bogusUnknownField: true
  template:
    metadata:
      labels:
        app: x
    spec:
      containers:
        - name: c
          image: nginx:1.25
          resources:
            limits: {cpu: 100m, memory: 128Mi}
            requests: {cpu: 100m, memory: 128Mi}
`,
			}},
			want: []string{
				"invalid_type",
				"additional_property_not_allowed",
				"required",
			},
		},
		{
			// Kinds the kubeval schema bundle does not know about produce a
			// `kubeval-schema-not-found` warning rather than a hard failure.
			name: "kubeval_schema_not_found_for_unknown_kind",
			files: []specFile{{
				Name: "weird.yaml", Path: "weird.yaml",
				Content: `apiVersion: example.com/v1
kind: TotallyMadeUpKind
metadata:
  name: nope
spec:
  whatever: true
`,
			}},
			want: []string{"kubeval-schema-not-found"},
		},
		{
			// Both branches of deployment-phase-annotation: non-integer value
			// and an integer outside the [-9999, 9999] range. Both must fire.
			name: "deployment_phase_annotation_invalid",
			files: []specFile{
				{
					Name: "cm-bad-int.yaml", Path: "cm-bad-int.yaml",
					Content: `apiVersion: v1
kind: ConfigMap
metadata:
  name: bad-int
  annotations:
    kots.io/creation-phase: "not-an-int"
data: {}
`,
				},
				{
					Name: "cm-out-of-range.yaml", Path: "cm-out-of-range.yaml",
					Content: `apiVersion: v1
kind: ConfigMap
metadata:
  name: out-of-range
  annotations:
    kots.io/deletion-phase: "999999"
data: {}
`,
				},
			},
			want: []string{"deployment-phase-annotation"},
		},
		{
			name: "wait_for_properties_annotation_malformed",
			files: []specFile{{
				Name: "cm.yaml", Path: "cm.yaml",
				Content: `apiVersion: v1
kind: ConfigMap
metadata:
  name: wfp
  annotations:
    kots.io/wait-for-properties: "this-has-no-equals,=missingkey,key="
data: {}
`,
			}},
			want: []string{"wait-for-properties-annotation"},
		},
		{
			// HelmChart manifest without a corresponding .tgz archive.
			name: "helm_archive_missing",
			files: []specFile{{
				Name: "chart.yaml", Path: "chart.yaml",
				Content: `apiVersion: kots.io/v1beta2
kind: HelmChart
metadata:
  name: phantom
spec:
  chart:
    name: phantom
    chartVersion: "9.9.9"
  releaseName: phantom
`,
			}},
			want: []string{"helm-archive-missing"},
		},
		{
			// .tgz archive without a corresponding HelmChart manifest.
			name: "helm_chart_missing",
			files: []specFile{{
				Name:    "testchart-with-labels-16.2.2.tgz",
				Path:    "testchart-with-labels-16.2.2.tgz",
				Content: base64.StdEncoding.EncodeToString(testChartWithLabels),
			}},
			want: []string{"helm-chart-missing"},
		},
		{
			// Embedded Cluster v3 + a Preflight on the older v1beta2 apiVersion.
			name: "ec_v3_preflight_api_version",
			files: []specFile{
				{
					Name: "ec.yaml", Path: "ec.yaml",
					Content: `apiVersion: embeddedcluster.replicated.com/v1beta1
kind: Config
metadata:
  name: ec
spec:
  version: 3.0.0
`,
				},
				{
					Name: "preflight.yaml", Path: "preflight.yaml",
					Content: `apiVersion: troubleshoot.sh/v1beta2
kind: Preflight
metadata:
  name: pf
spec:
  analyzers: []
`,
				},
			},
			want: []string{"ec-v3-preflight-api-version"},
		},
		{
			// Embedded Cluster Config with a helm extension chart that omits
			// version triggers ec-helm-extension-version-required.
			name: "ec_helm_extension_version_required",
			files: []specFile{{
				Name: "ec.yaml", Path: "ec.yaml",
				Content: `apiVersion: embeddedcluster.replicated.com/v1beta1
kind: Config
metadata:
  name: ec
spec:
  version: 2.0.0
  extensions:
    helm:
      charts:
        - name: my-extension
          chartname: oci://example.com/my-extension
          # no version field on purpose
`,
			}},
			want: []string{"ec-helm-extension-version-required"},
		},
		{
			name: "status_informer_invalid_format_and_nonexistent_object",
			files: []specFile{
				{
					Name: "kots-app.yaml", Path: "kots-app.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: my-app
spec:
  title: My App
  icon: https://example.com/icon.png
  statusInformers:
    - "this is not a valid format"
    - deployment/does-not-exist
`,
				},
			},
			want: []string{
				"invalid-status-informer-format",
				"nonexistent-status-informer-object",
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resp := lintFiles(t, tc.files)
			assertRules(t, resp.LintExpressions, tc.want, tc.forbid)
		})
	}
}

// TestTroubleshootLint exercises /v1/troubleshoot-lint. The endpoint is
// schema-only (kubeval) so we test one valid and one schema-violating case.
func TestTroubleshootLint(t *testing.T) {
	t.Run("valid_preflight_passes", func(t *testing.T) {
		resp := troubleshootLint(t, `apiVersion: troubleshoot.sh/v1beta2
kind: Preflight
metadata:
  name: my-preflight
spec:
  analyzers: []
`)
		// We don't require zero output (kubeval may warn on schema-not-found),
		// but we should never see invalid-yaml on a well-formed doc.
		assertRules(t, resp.LintExpressions, nil, []string{"invalid-yaml"})
	})

	t.Run("invalid_yaml_fails", func(t *testing.T) {
		resp := troubleshootLint(t, "this: : is not yaml\n  - bad")
		assertRules(t, resp.LintExpressions, []string{"invalid-yaml"}, nil)
	})
}

// TestLintChartTroubleshootCRDs exercises troubleshoot-spec-in-chart-without-crd:
// when a helm chart packages a Preflight or SupportBundle CR template but
// doesn't ship the matching CRD, lint should warn (because the CR install
// will fail in clusters without the CRD already present). We construct chart
// archives inline so we can flip individual conditions: bare CR template
// (warn fires), CR template + CRD (warn suppressed), CR embedded in a Secret
// — the recommended pattern (warn suppressed).
func TestLintChartTroubleshootCRDs(t *testing.T) {
	const chartYaml = "apiVersion: v2\nname: ts-chart\nversion: 0.1.0\n"
	const preflightTemplate = `apiVersion: troubleshoot.sh/v1beta2
kind: Preflight
metadata:
  name: my-preflight
spec:
  analyzers: []
`
	const supportBundleTemplate = `apiVersion: troubleshoot.sh/v1beta2
kind: SupportBundle
metadata:
  name: my-sb
spec:
  collectors: []
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
	const preflightInSecret = `apiVersion: v1
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
	helmChartManifest := `apiVersion: kots.io/v1beta2
kind: HelmChart
metadata:
  name: ts-chart
spec:
  chart:
    name: ts-chart
    chartVersion: "0.1.0"
  releaseName: ts-chart
`

	chartFiles := func(t *testing.T, contents map[string]string) []specFile {
		return []specFile{
			{Name: "ts-chart.yaml", Path: "ts-chart.yaml", Content: helmChartManifest},
			{
				Name:    "ts-chart-0.1.0.tgz",
				Path:    "ts-chart-0.1.0.tgz",
				Content: buildChartTgz(t, contents),
			},
		}
	}

	t.Run("preflight_template_without_crd_warns", func(t *testing.T) {
		resp := lintFiles(t, chartFiles(t, map[string]string{
			"ts-chart/Chart.yaml":                 chartYaml,
			"ts-chart/templates/preflight.yaml":   preflightTemplate,
		}))
		assertRules(t, resp.LintExpressions, []string{"troubleshoot-spec-in-chart-without-crd"}, nil)
	})

	t.Run("supportbundle_template_without_crd_warns", func(t *testing.T) {
		resp := lintFiles(t, chartFiles(t, map[string]string{
			"ts-chart/Chart.yaml":                       chartYaml,
			"ts-chart/templates/supportbundle.yaml":     supportBundleTemplate,
		}))
		assertRules(t, resp.LintExpressions, []string{"troubleshoot-spec-in-chart-without-crd"}, nil)
	})

	t.Run("preflight_with_matching_crd_does_not_warn", func(t *testing.T) {
		resp := lintFiles(t, chartFiles(t, map[string]string{
			"ts-chart/Chart.yaml":               chartYaml,
			"ts-chart/templates/preflight.yaml": preflightTemplate,
			"ts-chart/crds/preflight-crd.yaml":  preflightCRD,
		}))
		assertRules(t, resp.LintExpressions, nil, []string{"troubleshoot-spec-in-chart-without-crd"})
	})

	t.Run("preflight_embedded_in_secret_does_not_warn", func(t *testing.T) {
		resp := lintFiles(t, chartFiles(t, map[string]string{
			"ts-chart/Chart.yaml":                      chartYaml,
			"ts-chart/templates/preflight-secret.yaml": preflightInSecret,
		}))
		assertRules(t, resp.LintExpressions, nil, []string{"troubleshoot-spec-in-chart-without-crd"})
	})
}

// TestBuildersLint exercises /v1/builders-lint, which expects a tar stream
// (or a single application/gzip chart). One case sends a labeled chart that
// renders cleanly except for a missing preflight; another sends a corrupt
// archive that fails to render. Together they cover both the OPA path and
// the rendering-error path of the handler.
func TestBuildersLint(t *testing.T) {
	tarOf := func(t *testing.T, entries map[string][]byte) []byte {
		t.Helper()
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)
		for name, data := range entries {
			if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0644, Size: int64(len(data))}); err != nil {
				t.Fatalf("tar header: %v", err)
			}
			if _, err := tw.Write(data); err != nil {
				t.Fatalf("tar write: %v", err)
			}
		}
		if err := tw.Close(); err != nil {
			t.Fatalf("tar close: %v", err)
		}
		return buf.Bytes()
	}

	postTar := func(t *testing.T, body []byte) lintResponse {
		t.Helper()
		req, err := http.NewRequest(http.MethodPost, baseURL(t)+"/v1/builders-lint", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		req.Header.Set("Content-Type", "application/tar")
		resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
		if err != nil {
			t.Fatalf("post: %v", err)
		}
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status %d: %s", resp.StatusCode, respBody)
		}
		var lr lintResponse
		if err := json.Unmarshal(respBody, &lr); err != nil {
			t.Fatalf("decode: %v\n%s", err, respBody)
		}
		return lr
	}

	t.Run("clean_chart_with_labels_only_missing_preflight", func(t *testing.T) {
		body := tarOf(t, map[string][]byte{"testchart-with-labels-16.2.2.tgz": testChartWithLabels})
		resp := postTar(t, body)
		assertRules(t, resp.LintExpressions,
			[]string{"preflight-spec"},
			[]string{"informers-labels-not-found", "rendering"},
		)
	})

	t.Run("corrupt_archive_emits_rendering_error", func(t *testing.T) {
		body := tarOf(t, map[string][]byte{"not-a-chart.tgz": notAChart})
		resp := postTar(t, body)
		assertRules(t, resp.LintExpressions, []string{"rendering"}, nil)
	})
}

// TestUnknownEndpoint guards against accidental routing changes — if /v1/lint
// disappears the table tests above blow up with confusing 404 messages.
func TestUnknownEndpoint(t *testing.T) {
	resp, err := http.Post(baseURL(t)+"/v1/this-does-not-exist", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected non-200 from unknown endpoint, got 200: %s", body)
	}
	_ = fmt.Sprintf // keep fmt imported for future cases
}
