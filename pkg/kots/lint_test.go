package kots

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_lintFileHasValidYAML(t *testing.T) {
	tests := []struct {
		name     string
		specFile SpecFile
		expect   []LintExpression
	}{
		{
			name: "single no errors",
			specFile: SpecFile{
				Path: "file.yaml",
				Content: `apiVersion: v1
kind: ConfigMap
metadata:
  name: example_config
data:
  ENV_VAR_1: "fake"`,
			},
			expect: []LintExpression{},
		},
		{
			name: "single with errors",
			specFile: SpecFile{
				Path: "file.yaml",
				Content: `apiVersion: v1
kind: ConfigMap
metadata:
  name: example_config
data:
  ENV_VAR_1: "fake"
  ENV_VAR_2: kind: test`,
			},
			expect: []LintExpression{
				{
					Rule:    "invalid-yaml",
					Type:    "error",
					Path:    "file.yaml",
					Message: "yaml: line 7: mapping values are not allowed in this context",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 7,
							},
						},
					},
				},
			},
		},
		{
			name: "multi no errors",
			specFile: SpecFile{
				Path: "file.yaml",
				Content: `---
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
			},
			expect: []LintExpression{},
		},
		{
			name: "multi with errors in first",
			specFile: SpecFile{
				Path: "file.yaml",
				Content: `---
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
			},
			expect: []LintExpression{
				{
					Rule:    "invalid-yaml",
					Type:    "error",
					Path:    "file.yaml",
					Message: "yaml: line 8: mapping values are not allowed in this context",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 8,
							},
						},
					},
				},
			},
		},
		{
			name: "multi with errors in second",
			specFile: SpecFile{
				Path: "file.yaml",
				Content: `---
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
			},
			expect: []LintExpression{
				{
					Rule:    "invalid-yaml",
					Type:    "error",
					Path:    "file.yaml",
					Message: "yaml: line 15: mapping values are not allowed in this context",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 15,
							},
						},
					},
				},
			},
		},
		{
			name: "proxy",
			specFile: SpecFile{
				Path: "file.yaml",
				Content: `apiVersion: v1
kind: ConfigMap
metadata:
  name: proxy
data:
  HTTP_PROXY: "{{repl HTTPProxy }}"
  NO_PROXY: "{{repl NoProxy }}"`,
			},
			expect: []LintExpression{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := lintFileHasValidYAML(test.specFile)
			assert.ElementsMatch(t, actual, test.expect)
		})
	}
}

func Test_lintHelmCharts(t *testing.T) {
	tests := []struct {
		name      string
		specFiles SpecFiles
		expect    []LintExpression
	}{
		{
			name: "basic no errors",
			specFiles: SpecFiles{
				{
					Name:    "redis-10.3.5.tar.gz",
					Path:    "redis-10.3.5.tar.gz",
					Content: `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`,
				},
				{
					Name: "redis.yaml",
					Path: "redis-10.3.5.tar.gz/redis.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: HelmChart
metadata:
  name: redis
spec:
  # chart identifies a matching chart from a .tgz
  chart:
    name: redis
    chartVersion: 10.3.5

  # values are used in the customer environment, as a pre-render step
  # these values will be supplied to helm template
  values: {}

  # builder values provide a way to render the chart with all images
  # and manifests. this is used in replicated to create airgap packages
  builder: {}
`,
				},
			},
			expect: []LintExpression{},
		},
		{
			name: "multi doc no errors",
			specFiles: SpecFiles{
				{
					Name:    "redis-10.3.5.tar.gz",
					Path:    "redis-10.3.5.tar.gz",
					Content: `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`,
				},
				{
					Name: "multi-doc.yaml",
					Path: "multi-doc-folder/multi-doc.yaml",
					Content: `apiVersion: v1
kind: ConfigMap
metadata:
  name: example-config
data:
    ENV_VAR_1: "fake"
    ENV_VAR_2: "faker"
---
apiVersion: kots.io/v1beta1
kind: HelmChart
metadata:
  name: redis
spec:
  # chart identifies a matching chart from a .tgz
  chart:
    name: redis
    chartVersion: 10.3.5

  # values are used in the customer environment, as a pre-render step
  # these values will be supplied to helm template
  values: {}

  # builder values provide a way to render the chart with all images
  # and manifests. this is used in replicated to create airgap packages
  builder: {}
`,
				},
			},
			expect: []LintExpression{},
		},
		{
			name: "missing chart manifest",
			specFiles: SpecFiles{
				{
					Name:    "redis-10.3.5.tar.gz",
					Path:    "redis-10.3.5.tar.gz",
					Content: `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`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "helm-chart-missing",
					Type:    "error",
					Message: "Could not find helm chart manifest for archive 'redis-10.3.5.tar.gz'",
				},
			},
		},
		{
			name: "version mismatch",
			specFiles: SpecFiles{
				{
					Name:    "redis-10.3.5.tar.gz",
					Path:    "redis-10.3.5.tar.gz",
					Content: `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`,
				},
				{
					Name: "redis.yaml",
					Path: "redis.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: HelmChart
metadata:
  name: redis
spec:
  # chart identifies a matching chart from a .tgz
  chart:
    name: redis
    chartVersion: 10.3.4

  # values are used in the customer environment, as a pre-render step
  # these values will be supplied to helm template
  values: {}

  # builder values provide a way to render the chart with all images
  # and manifests. this is used in replicated to create airgap packages
  builder: {}
`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "helm-chart-missing",
					Type:    "error",
					Message: "Could not find helm chart manifest for archive 'redis-10.3.5.tar.gz'",
				},
				{
					Rule:    "helm-archive-missing",
					Type:    "error",
					Message: "Could not find helm archive for chart 'redis' version '10.3.4'",
				},
			},
		},
		{
			name: "missing helm archive",
			specFiles: SpecFiles{
				{
					Name: "redis.yaml",
					Path: "redis.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: HelmChart
metadata:
  name: redis
spec:
  # chart identifies a matching chart from a .tgz
  chart:
    name: redis
    chartVersion: 10.3.5

  # values are used in the customer environment, as a pre-render step
  # these values will be supplied to helm template
  values: {}

  # builder values provide a way to render the chart with all images
  # and manifests. this is used in replicated to create airgap packages
  builder: {}
`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "helm-archive-missing",
					Type:    "error",
					Message: "Could not find helm archive for chart 'redis' version '10.3.5'",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := lintHelmCharts(test.specFiles)
			require.NoError(t, err)
			assert.ElementsMatch(t, actual, test.expect)
		})
	}
}

func Test_lintWithKubeval(t *testing.T) {
	tests := []struct {
		name      string
		specFiles SpecFiles
		expect    []LintExpression
	}{
		{
			name: "non-int replicas after rendering",
			specFiles: SpecFiles{
				{
					Name: "config.yaml",
					Path: "config.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: config-sample
spec:
  groups:
    - name: example_settings
      title: My Example Config
      description: Configuration to serve as an example for creating your own
      items:
        - name: a_templated_value
          title: my value
          type: text
          value: "asd"`,
				},
				{
					Name: "test.yaml",
					Path: "test.yaml",
					Content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: example-nginx
  label:
    app: example
    component: nginx
spec:
  replicas: repl{{ConfigOption "a_templated_value"}}
  selector:
    matchLabels:
      app: example
      component: nginx
  template:
    metadata:
      labels:
        app: example
        component: nginx
    spec:
      containers:
        - image: nginx
          envFrom:
          - configMapRe:
              name: example-config
          resources:
            limi:
              memory: '256Mi'
              cpu: '500m'`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "additional_property_not_allowed",
					Type:    "warn",
					Path:    "test.yaml",
					Message: "Additional property label is not allowed",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 3,
							},
						},
					},
				},
				{
					Rule:    "invalid_type",
					Type:    "warn",
					Path:    "test.yaml",
					Message: "Invalid type. Expected: [integer,null], given: string",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 9,
							},
						},
					},
				},
				{
					Rule:    "required",
					Type:    "warn",
					Path:    "test.yaml",
					Message: "name is required",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 21,
							},
						},
					},
				},
				{
					Rule:    "additional_property_not_allowed",
					Type:    "warn",
					Path:    "test.yaml",
					Message: "Additional property configMapRe is not allowed",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 23,
							},
						},
					},
				},
				{
					Rule:    "additional_property_not_allowed",
					Type:    "warn",
					Path:    "test.yaml",
					Message: "Additional property limi is not allowed",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 25,
							},
						},
					},
				},
			},
		},
		{
			name: "int replicas after rendering",
			specFiles: SpecFiles{
				{
					Name: "config.yaml",
					Path: "config.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: config-sample
spec:
  groups:
    - name: example_settings
      title: My Example Config
      description: Configuration to serve as an example for creating your own
      items:
        - name: a_templated_value
          title: a text field with a value provided by a template function
          type: text
          value: "6"`,
				},
				{
					Name: "test.yaml",
					Path: "test.yaml",
					Content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: example-nginx
  labels:
    app: example
    component: nginx
spec:
  replicas: repl{{ConfigOption "a_templated_value"}}
  selector:
    matchLabels:
      app: example
      component: nginx
  template:
    metadata:
      labels:
        app: example
        component: nginx
    spec:
      containers:
        - name: nginx
          image: nginx
          envFrom:
          - configMapRef:
              name: example-config
          resources:
            limits:
              memory: '256Mi'
              cpu: '500m'`,
				},
			},
			expect: []LintExpression{},
		},
		{
			name: "kubeval no matching schema",
			specFiles: SpecFiles{
				{
					Name: "test.yaml",
					Path: "test.yaml",
					Content: `apiVersion: nomatchingschema.io/v1
kind: NoMatchingKind
metadata:
  name: no-matching-metadata`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "kubeval-schema-not-found",
					Type:    "warn",
					Path:    "test.yaml",
					Message: "We currently have no matching schema to lint this type of file",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := lintWithKubevalSchema(test.specFiles, "file://../../kubernetes-json-schema")
			require.NoError(t, err)
			assert.ElementsMatch(t, actual, test.expect)
		})
	}
}

func Test_lintRenderContent(t *testing.T) {
	tests := []struct {
		name      string
		specFiles SpecFiles
		expect    []LintExpression
	}{
		{
			name: "basic with no errors",
			specFiles: SpecFiles{
				{
					Name: "config.yaml",
					Path: "config.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: config-sample
spec:
  groups:
    - name: example_settings
      title: My Example Config
      description: Configuration to serve as an example for creating your own
      items:
        - name: a_templated_value
          title: my value
          type: text
          value: value`,
				},
				{
					Name: "test.yaml",
					Path: "test.yaml",
					Content: `apiVersion: v1
kind: ConfigMap
metadata:
  name: example-config
data:
    ENV_VAR_1: "fake"
    ENV_VAR_2: '{{repl ConfigOption "a_templated_value" }}'
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: example-config-2
data:
    ENV_VAR_1: "fake"
    ENV_VAR_2: '{{repl ConfigOption "a_templated_value" }}'`,
				},
			},
			expect: []LintExpression{},
		},
		{
			name: "invalid config",
			specFiles: SpecFiles{
				{
					Name: "config.yaml",
					Path: "config.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: config-sample
spec:
  groups:
    - name: example_settings
      title: My Example Config
      description: Configuration to serve as an example for creating your own
      items:
        - name: a_templated_value
          title: 2
          type: text
          value: "6"`,
				},
				{
					Name: "test.yaml",
					Path: "test.yaml",
					Content: `apiVersion: v1
kind: ConfigMap
metadata:
  name: example-config
data:
    ENV_VAR_1: "fake"
    ENV_VAR_2: '{{repl ConfigOption "a_templated_value" }}'
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: example-config-2
data:
    ENV_VAR_1: "fake"
    ENV_VAR_2: '{{repl ConfigOption "a_templated_value" }}'`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "config-is-invalid",
					Type:    "error",
					Path:    "config.yaml",
					Message: `failed to decode config content: v1beta1.Config.Spec: v1beta1.ConfigSpec.Groups: []v1beta1.ConfigGroup: v1beta1.ConfigGroup.Items: []v1beta1.ConfigItem: v1beta1.ConfigItem.Value: Type: Title: ReadString: expects " or n, but found 2, error found in #10 byte of ...|,"title":2,"type":"t|..., bigger context ...|wn","items":[{"name":"a_templated_value","title":2,"type":"text","value":"6"}],"name":"example_setti|...`,
				},
			},
		},
		{
			name: "unable to render different errors",
			specFiles: SpecFiles{
				{
					Name: "config.yaml",
					Path: "config.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: config-sample
spec:
  groups:
    - name: example_settings
      title: My Example Config
      description: Configuration to serve as an example for creating your own
      items:
        - name: a_templated_value
          title: title
          type: text
          value: "6"`,
				},
				{
					Name: "test.yaml",
					Path: "test.yaml",
					Content: `apiVersion: v1
kind: ConfigMap
metadata:
  name: example-config
data:
    ENV_VAR_1: "fake"

    ENV_VAR_2: '{{repl print "whatever" | sha256 }}'
---

apiVersion: v1
kind: ConfigMap
metadata:
  name: example-config-2
data:
    ENV_VAR_1: "fake"
# this is a comment
    ENV_VAR_2: '{{repl print "whatever }}'`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "unable-to-render",
					Type:    "error",
					Path:    "test.yaml",
					Message: `function "sha256" not defined`,
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 8,
							},
						},
					},
				},
				{
					Rule:    "unable-to-render",
					Type:    "error",
					Path:    "test.yaml",
					Message: "unterminated quoted string",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 18,
							},
						},
					},
				},
			},
		},
		{
			name: "unable to render same error but different docs",
			specFiles: SpecFiles{
				{
					Name: "config.yaml",
					Path: "config.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: config-sample
spec:
  groups:
    - name: example_settings
      title: My Example Config
      description: Configuration to serve as an example for creating your own
      items:
        - name: a_templated_value
          title: title
          type: text
          value: "6"`,
				},
				{
					Name: "test.yaml",
					Path: "test.yaml",
					Content: `apiVersion: v1
kind: ConfigMap
metadata:
  name: example-config
data:
    ENV_VAR_1: "fake"

    ENV_VAR_2: '{{repl print "whatever" | sha256 }}'
---

apiVersion: v1
kind: ConfigMap
metadata:
  name: example-config-2
data:
    ENV_VAR_1: "fake"
# this is a comment

    ENV_VAR_2: '{{repl print "whatever" | sha256 }}'`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "unable-to-render",
					Type:    "error",
					Path:    "test.yaml",
					Message: `function "sha256" not defined`,
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 8,
							},
						},
					},
				},
				{
					Rule:    "unable-to-render",
					Type:    "error",
					Path:    "test.yaml",
					Message: `function "sha256" not defined`,
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 19,
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := lintRenderContent(test.specFiles)
			require.NoError(t, err)
			assert.ElementsMatch(t, actual, test.expect)
		})
	}
}

func Test_lintWithOPANonRendered(t *testing.T) {
	tests := []struct {
		name      string
		specFiles SpecFiles
		expect    []LintExpression
	}{
		{
			name: "config option found",
			specFiles: SpecFiles{
				{
					Name: "config.yaml",
					Path: "config.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: config-sample
spec:
  groups:
    - name: example_settings
      title: My Example Config
      description: Configuration to serve as an example for creating your own
      items:
        - name: a_templated_value
          title: a text field with a value provided by a template function
          type: text
          value: "6"`,
				},
				{
					Name: "test.yaml",
					Path: "test.yaml",
					Content: `apiVersion: v1
kind: ConfigMap
metadata:
  name: example-config
data:
    ENV_VAR_1: "fake"
    ENV_VAR_2: '{{repl ConfigOption "a_templated_value" }}'
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: example-config-2
data:
    ENV_VAR_1: "faker"
    ENV_VAR_2: '{{repl ConfigOption "a_templated_value" }}'`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "preflight-spec",
					Type:    "warn",
					Message: "Missing preflight spec",
				},
				{
					Rule:    "troubleshoot-spec",
					Type:    "warn",
					Message: "Missing troubleshoot spec",
				},
				{
					Rule:    "application-spec",
					Type:    "warn",
					Message: "Missing application spec",
				},
			},
		},
		{
			name: "config option not found",
			specFiles: SpecFiles{
				{
					Name: "config.yaml",
					Path: "config.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: config-sample
spec:
  groups:
    - name: example_settings
      title: My Example Config
      description: Configuration to serve as an example for creating your own
      items:
        - name: a_templated_value
          title: a text field with a value provided by a template function
          type: text
          value: "6"`,
				},
				{
					Name: "test.yaml",
					Path: "test.yaml",
					Content: `apiVersion: v1
kind: ConfigMap
metadata:
  name: example-config
data:
    ENV_VAR_1: "fake"
    ENV_VAR_2: '{{repl ConfigOption "does_not_exist" }}'
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: example-config-2
data:
    ENV_VAR_1: "faker"
    ENV_VAR_2: '{{repl ConfigOption "does_not_exist" }}'`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "preflight-spec",
					Type:    "warn",
					Message: "Missing preflight spec",
				},
				{
					Rule:    "troubleshoot-spec",
					Type:    "warn",
					Message: "Missing troubleshoot spec",
				},
				{
					Rule:    "application-spec",
					Type:    "warn",
					Message: "Missing application spec",
				},
				{
					Rule:    "config-option-not-found",
					Type:    "warn",
					Path:    "test.yaml",
					Message: "Config option \"does_not_exist\" not found",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 7,
							},
						},
					},
				},
				{
					Rule:    "config-option-not-found",
					Type:    "warn",
					Path:    "test.yaml",
					Message: "Config option \"does_not_exist\" not found",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 15,
							},
						},
					},
				},
			},
		},
		{
			name: "support bundle is troubelshoot spec",
			specFiles: SpecFiles{
				{
					Name: "supportbundle.yaml",
					Path: "supportbundle.yaml",
					Content: `apiVersion: troubleshoot.sh/v1beta2
kind: SupportBundle
metadata:
  name: support-bundle`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "preflight-spec",
					Type:    "warn",
					Message: "Missing preflight spec",
				},
				{
					Rule:    "application-spec",
					Type:    "warn",
					Message: "Missing application spec",
				},
				{
					Rule:    "config-spec",
					Type:    "warn",
					Message: "Missing config spec",
				},
			},
		},
	}

	InitOPALinting("./rego")

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := lintWithOPANonRendered(test.specFiles)
			require.NoError(t, err)
			assert.ElementsMatch(t, actual, test.expect)
		})
	}
}

func Test_lintWithOPARendered(t *testing.T) {
	tests := []struct {
		name      string
		specFiles SpecFiles
		expect    []LintExpression
	}{
		{
			name: "type/name no errors",
			specFiles: SpecFiles{
				{
					Name: "deployment.yaml",
					Path: "deployment.yaml",
					Content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: example-nginx
  labels:
    app: example
    component: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: example
      component: nginx
  template:
    metadata:
      labels:
        app: example
        component: nginx
    spec:
      containers:
        - image: nginx
          envFrom:
          - configMapRe:
              name: example-config
          resources:
            limits:
              memory: '256Mi'
              cpu: '500m'`,
				},
				{
					Name: "test.yaml",
					Path: "test.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-slug
spec:
  title: App Name
  icon: https://github.com/cncf/artwork/blob/master/projects/kubernetes/icon/color/kubernetes-icon-color.png
  statusInformers:
    - deployment/example-nginx
  ports:
    - serviceName: "example-nginx"
      servicePort: 80
      localPort: 8888
      applicationUrl: "http://example-nginx"`,
				},
			},
			expect: []LintExpression{},
		},
		{
			name: "type/name with errors",
			specFiles: SpecFiles{
				{
					Name: "deployment.yaml",
					Path: "deployment.yaml",
					Content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: example-nginx
  labels:
    app: example
    component: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: example
      component: nginx
  template:
    metadata:
      labels:
        app: example
        component: nginx
    spec:
      containers:
        - image: nginx
          envFrom:
          - configMapRe:
              name: example-config
          resources:
            limits:
              memory: '256Mi'
              cpu: '500m'`,
				},
				{
					Name: "test.yaml",
					Path: "test.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-slug
spec:
  title: App Name
  icon: https://github.com/cncf/artwork/blob/master/projects/kubernetes/icon/color/kubernetes-icon-color.png
  statusInformers:
    - deployment-example-nginx
    - service/example-nginx
  ports:
    - serviceName: "example-nginx"
      servicePort: 80
      localPort: 8888
      applicationUrl: "http://example-nginx"`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "invalid-status-informer-format",
					Type:    "warn",
					Path:    "test.yaml",
					Message: "Invalid status informer format",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 9,
							},
						},
					},
				},
				{
					Rule:    "nonexistent-status-informer-object",
					Type:    "warn",
					Path:    "test.yaml",
					Message: "Status informer points to a nonexistent kubernetes object",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 10,
							},
						},
					},
				},
			},
		},
		{
			name: "namespace/type/name does not exist",
			specFiles: SpecFiles{
				{
					Name: "deployment.yaml",
					Path: "deployment.yaml",
					Content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: example-nginx
  namespace: test
  labels:
    app: example
    component: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: example
      component: nginx
  template:
    metadata:
      labels:
        app: example
        component: nginx
    spec:
      containers:
        - image: nginx
          envFrom:
          - configMapRe:
              name: example-config
          resources:
            limits:
              memory: '256Mi'
              cpu: '500m'`,
				},
				{
					Name: "test.yaml",
					Path: "test.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-slug
spec:
  title: App Name
  icon: https://github.com/cncf/artwork/blob/master/projects/kubernetes/icon/color/kubernetes-icon-color.png
  statusInformers:
    - test/deployment/example-nginx
    - test/service/example-nginx
  ports:
    - serviceName: "example-nginx"
      servicePort: 80
      localPort: 8888
      applicationUrl: "http://example-nginx"`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "nonexistent-status-informer-object",
					Type:    "warn",
					Path:    "test.yaml",
					Message: "Status informer points to a nonexistent kubernetes object",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 10,
							},
						},
					},
				},
			},
		},
		{
			name: "render namespace/type/name 1",
			specFiles: SpecFiles{
				{
					Name: "config.yaml",
					Path: "config.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: config-sample
spec:
  groups:
    - name: example_settings
      title: My Example Config
      description: My Example Description
      items:
        - name: db_type
          type: select_one
          default: embedded
          items:
            - name: external
              title: External
            - name: embedded
              title: Embedded DB`,
				},
				{
					Name: "deployment.yaml",
					Path: "deployment.yaml",
					Content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: example-nginx
  namespace: test
  labels:
    app: example
    component: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: example
      component: nginx
  template:
    metadata:
      labels:
        app: example
        component: nginx
    spec:
      containers:
        - image: nginx
          envFrom:
          - configMapRe:
              name: example-config
          resources:
            limits:
              memory: '256Mi'
              cpu: '500m'`,
				},
				{
					Name: "test.yaml",
					Path: "test.yaml",
					Content: `apiVersion: v1
kind: ConfigMap
metadata:
  name: example-config
data:
    ENV_VAR_1: "fake"
    ENV_VAR_2: '{{repl ConfigOption "does_not_exist" }}'
---
apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-slug
spec:
  title: App Name
  icon: https://github.com/cncf/artwork/blob/master/projects/kubernetes/icon/color/kubernetes-icon-color.png
  statusInformers:
    - service-example-nginx
    - '{{repl if ConfigOptionEquals "db_type" "embedded"}}test/service/example-nginx{{repl else}}{{repl end}}'
    - '{{repl if ConfigOptionEquals "db_type" "external"}}test/service/example-nginx{{repl else}}{{repl end}}'
  ports:
    - serviceName: "example-nginx"
      servicePort: 80
      localPort: 8888
      applicationUrl: "http://example-nginx"`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "invalid-status-informer-format",
					Type:    "warn",
					Path:    "test.yaml",
					Message: "Invalid status informer format",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 17,
							},
						},
					},
				},
				{
					Rule:    "nonexistent-status-informer-object",
					Type:    "warn",
					Path:    "test.yaml",
					Message: "Status informer points to a nonexistent kubernetes object",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 18,
							},
						},
					},
				},
			},
		},
		{
			name: "render namespace/type/name 2",
			specFiles: SpecFiles{
				{
					Name: "config.yaml",
					Path: "config.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: config-sample
spec:
  groups:
    - name: example_settings
      title: My Example Config
      description: My Example Description
      items:
        - name: db_type
          type: select_one
          default: embedded
          items:
            - name: external
              title: External
            - name: embedded
              title: Embedded DB`,
				},
				{
					Name: "deployment.yaml",
					Path: "deployment.yaml",
					Content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: example-nginx
  namespace: test
  labels:
    app: example
    component: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: example
      component: nginx
  template:
    metadata:
      labels:
        app: example
        component: nginx
    spec:
      containers:
        - image: nginx
          envFrom:
          - configMapRe:
              name: example-config
          resources:
            limits:
              memory: '256Mi'
              cpu: '500m'`,
				},
				{
					Name: "test.yaml",
					Path: "test.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-slug
spec:
  title: App Name
  icon: https://github.com/cncf/artwork/blob/master/projects/kubernetes/icon/color/kubernetes-icon-color.png
  statusInformers:
    - test/deployment/example-nginx
    - '{{repl if ConfigOptionEquals "db_type" "embedded"}}test/deployment/example-nginx{{repl else}}{{repl end}}'
    - '{{repl if ConfigOptionEquals "db_type" "external"}}test/service/example-nginx{{repl else}}{{repl end}}'
  ports:
    - serviceName: "example-nginx"
      servicePort: 80
      localPort: 8888
      applicationUrl: "http://example-nginx"`,
				},
			},
			expect: []LintExpression{},
		},
	}

	InitOPALinting("./rego")

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := lintWithOPARendered(test.specFiles)
			require.NoError(t, err)
			assert.ElementsMatch(t, actual, test.expect)
		})
	}
}
