package kots

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_renderSpecFiles(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		files      SpecFiles
		want       SpecFiles
	}{
		{
			name:       "basic",
			configPath: "config.yaml",
			files: SpecFiles{
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
        - name: a_templated_text
          title: a text field with a value provided by a template function
          type: text
          value: a templated value
        - name: try_to_template_me
          title: try to template me
          type: text
          value: '{{repl ConfigOption "a_templated_text"}}'
`,
				},
				{
					Name: "application.yaml",
					Path: "application.yaml",
					Content: `apiVersion: app.k8s.io/v1beta1
kind: Application
metadata:
  name: "example-app"
  labels:
    app.kubernetes.io/name: "example-app"
spec:
  descriptor:
    links:
      - description: '{{repl ConfigOption "a_templated_text"}}'
        url: "http://example-nginx"
`,
				},
				{
					Name: "replicated-app.yaml",
					Path: "replicated-app.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-slug
spec:
  title: App Name
  icon: icon-url
  statusInformers:
    - deployment/example-nginx
  releaseNotes: '{{repl ConfigOption "a_templated_text"}}'
`,
				},
				{
					Name: "support-bundle.yaml",
					Path: "support-bundle.yaml",
					Content: `apiVersion: troubleshoot.replicated.com/v1beta1
kind: Collector
metadata:
  name: collector-sample
spec:
  collectors:
    - clusterInfo: {}
    - clusterResources: {}
    - secret:
        key: '{{repl ConfigOption "a_templated_text"}}'
`,
				},
				{
					Name: "preflight.yaml",
					Path: "preflight.yaml",
					Content: `apiVersion: troubleshoot.replicated.com/v1beta1
kind: Preflight
metadata:
  name: example-preflight-checks
spec:
  analyzers:
  - clusterVersion:
      outcomes:
      - pass:
          message: repl{{ConfigOption "a_templated_text"}}
`,
				},
				{
					Name: "service.yaml",
					Path: "service.yaml",
					Content: `apiVersion: v1
kind: Service
metadata:
  labels:
    app: example
    component: nginx
  name: example-nginx
spec:
  ports:
  - port: 80
  selector:
    app: example
    component: repl{{ConfigOption "a_templated_text"}}
  type: LoadBalancer
`,
				},
			},
			want: SpecFiles{
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
        - name: a_templated_text
          title: a text field with a value provided by a template function
          type: text
          value: a templated value
        - name: try_to_template_me
          title: try to template me
          type: text
          value: 'a templated value'
`,
				},
				{
					Name: "application.yaml",
					Path: "application.yaml",
					Content: `apiVersion: app.k8s.io/v1beta1
kind: Application
metadata:
  name: "example-app"
  labels:
    app.kubernetes.io/name: "example-app"
spec:
  descriptor:
    links:
      - description: 'a templated value'
        url: "http://example-nginx"
`,
				},
				{
					Name: "replicated-app.yaml",
					Path: "replicated-app.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-slug
spec:
  title: App Name
  icon: icon-url
  statusInformers:
    - deployment/example-nginx
  releaseNotes: 'a templated value'
`,
				},
				{
					Name: "support-bundle.yaml",
					Path: "support-bundle.yaml",
					Content: `apiVersion: troubleshoot.replicated.com/v1beta1
kind: Collector
metadata:
  name: collector-sample
spec:
  collectors:
    - clusterInfo: {}
    - clusterResources: {}
    - secret:
        key: 'a templated value'
`,
				},
				{
					Name: "preflight.yaml",
					Path: "preflight.yaml",
					Content: `apiVersion: troubleshoot.replicated.com/v1beta1
kind: Preflight
metadata:
  name: example-preflight-checks
spec:
  analyzers:
  - clusterVersion:
      outcomes:
      - pass:
          message: a templated value
`,
				},
				{
					Name: "service.yaml",
					Path: "service.yaml",
					Content: `apiVersion: v1
kind: Service
metadata:
  labels:
    app: example
    component: nginx
  name: example-nginx
spec:
  ports:
  - port: 80
  selector:
    app: example
    component: a templated value
  type: LoadBalancer
`,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, path, err := tt.files.findAndValidateConfig()

			require.NoError(t, err)
			assert.Equal(t, path, tt.configPath)

			renderedFiles := SpecFiles{}
			for _, file := range tt.files {
				renderedContent, err := file.renderContent(config)
				require.NoError(t, err)
				file.Content = string(renderedContent)
				renderedFiles = append(renderedFiles, file)
			}

			require.NoError(t, err)
			assert.ElementsMatch(t, renderedFiles, tt.want)
		})
	}
}

func Test_renderInvalidTemplate(t *testing.T) {
	tests := []struct {
		name string
		file SpecFile
		want RenderTemplateError
	}{
		{
			name: "undefined function",
			file: SpecFile{
				Name: "undefined-function.yaml",
				Path: "undefined-function.yaml",
				Content: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: config-sample
spec:
  groups:
    - name: fake
      title: Fake
      items:
        - name: hash_func
          type: text
          value: '{{repl print "whatever" | sha256 }}'
`,
			},
			want: RenderTemplateError{
				message: `function "sha256" not defined`,
				line:    12,
			},
		},
		{
			name: "unterminated quotes",
			file: SpecFile{
				Name: "unterminated-quotes.yaml",
				Path: "unterminated-quotes.yaml",
				Content: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: config-sample
spec:
  groups:
    - name: fake
      title: Fake
      items:
        - name: my_func
          type: text
          default: my default value
          value: 'repl{{print "whatever }}'
`,
			},
			want: RenderTemplateError{
				message: `unterminated quoted string`,
				line:    13,
			},
		},
		{
			name: "expected more arguemnts",
			file: SpecFile{
				Name: "more-arguments.yaml",
				Path: "more-arguments.yaml",
				Content: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: config-sample
spec:
  groups:
    - name: fake
      items:
        - name: my_func
          type: text
          default: my default value
          value: 'repl{{ConfigOptionEquals "whatever" }}'
`,
			},
			want: RenderTemplateError{
				message: `at <ConfigOptionEquals>: wrong number of args for ConfigOptionEquals: want 2 got 1`,
				line:    12,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.file.renderContent(nil)

			renderTemplateError, ok := errors.Cause(err).(RenderTemplateError)
			assert.True(t, ok)

			assert.Equal(t, renderTemplateError.Error(), tt.want.Error())
			assert.Equal(t, renderTemplateError.Line(), tt.want.Line())
		})
	}
}
