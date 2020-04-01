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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := lintFileHasValidYAML(test.specFile)
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
      title: My Example Confi
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
      title: My Example Confi
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := lintWithKubevalSchema(test.specFiles, "file://../../kubernetes-json-schema")
			require.NoError(t, err)
			assert.ElementsMatch(t, actual, test.expect)
		})
	}
}

func Test_lintWithOPA(t *testing.T) {
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
      title: My Example Confi
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
      title: My Example Confi
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
			},
		},
	}

	InitOPALinting("./rego/kots-spec-default.rego")

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := lintWithOPA(test.specFiles)
			require.NoError(t, err)
			assert.ElementsMatch(t, actual, test.expect)
		})
	}
}
