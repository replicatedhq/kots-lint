package domain

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_renderSpecFiles(t *testing.T) {
	tests := []struct {
		name  string
		files SpecFiles
		want  SpecFiles
	}{
		{
			name: "basic",
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
		{
			name: "private-ca-cert",
			files: SpecFiles{
				{
					Name: "secret.yaml",
					Path: "secret.yaml",
					Content: `apiVersion: v1
kind: Secret
metadata:
  name: ca-certificate
  namespace: default
type: Opaque
data:
  ca.crt: '{{repl PrivateCACert }}'
`,
				},
			},
			want: SpecFiles{
				{
					Name: "secret.yaml",
					Path: "secret.yaml",
					Content: `apiVersion: v1
kind: Secret
metadata:
  name: ca-certificate
  namespace: default
type: Opaque
data:
  ca.crt: ''
`,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderedFiles, err := tt.files.Render()
			require.NoError(t, err)
			assert.ElementsMatch(t, renderedFiles, tt.want)
		})
	}
}

func Test_renderContent(t *testing.T) {
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
			config, path, err := tt.files.FindAndValidateConfig()

			require.NoError(t, err)
			assert.Equal(t, path, tt.configPath)

			builder, err := GetTemplateBuilder(config)
			require.NoError(t, err)

			renderedFiles := SpecFiles{}
			for _, file := range tt.files {
				renderedContent, err := file.RenderContent(builder)
				require.NoError(t, err)
				file.Content = string(renderedContent)
				renderedFiles = append(renderedFiles, file)
			}

			require.NoError(t, err)
			assert.ElementsMatch(t, renderedFiles, tt.want)
		})
	}
}

func Test_findAndValidateConfig(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		wantErr    string
		files      SpecFiles
	}{
		{
			name:       "invalid config",
			configPath: "config.yaml",
			wantErr:    "failed to render config: failed to template config objects: failed to template config: failed to render config template: failed to get template: template: config:20: bad character U+0022 '\"'",
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
          when: '{{repl ConfigOption a_templated_text"}}'
`,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, path, err := tt.files.FindAndValidateConfig()
			assert.Equal(t, err.Error(), tt.wantErr)
			assert.Equal(t, path, tt.configPath)
			assert.NotNil(t, config)
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
				Content: `apiVersion: troubleshoot.replicated.com/v1beta1
kind: Collector
metadata:
  name: collector-sample
spec:
  collectors:
    - clusterInfo: {}
    - clusterResources: {}
    - secret:
        key: '{{repl print "whatever" | sha256 }}'
`,
			},
			want: RenderTemplateError{
				message: `function "sha256" not defined`,
				match:   `        key: '{{repl print "whatever" | sha256 }}'`,
			},
		},
		{
			name: "unterminated quotes",
			file: SpecFile{
				Name: "unterminated-quotes.yaml",
				Path: "unterminated-quotes.yaml",
				Content: `apiVersion: troubleshoot.replicated.com/v1beta1
kind: Collector
metadata:
  name: collector-sample
spec:
  collectors:
    - clusterInfo: {}
    - secret:
        key: '{{repl print "whatever }}'
    - clusterResources: {}
`,
			},
			want: RenderTemplateError{
				message: `unterminated quoted string`,
				match:   `        key: '{{repl print "whatever }}'`,
			},
		},
		{
			name: "expected more arguemnts",
			file: SpecFile{
				Name: "more-arguments.yaml",
				Path: "more-arguments.yaml",
				Content: `apiVersion: troubleshoot.replicated.com/v1beta1
kind: Collector
metadata:
  name: collector-sample
spec:
  collectors:
    - clusterInfo: {}
    - clusterResources: {}
    - secret:
        key: 'repl{{ConfigOptionEquals "whatever" }}'
`,
			},
			want: RenderTemplateError{
				message: `at <ConfigOptionEquals>: wrong number of args for ConfigOptionEquals: want 2 got 1`,
				match:   `        key: 'repl{{ConfigOptionEquals "whatever" }}'`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder, err := GetTemplateBuilder(nil)
			require.NoError(t, err)

			_, err = tt.file.RenderContent(builder)

			renderTemplateError, ok := errors.Cause(err).(RenderTemplateError)
			assert.True(t, ok)

			assert.Equal(t, renderTemplateError.Error(), tt.want.Error())
			assert.Equal(t, renderTemplateError.Match(), tt.want.Match())
		})
	}
}
