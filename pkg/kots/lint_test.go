package kots

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	kurllint "github.com/replicatedhq/kurlkinds/pkg/lint"
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

func Test_lintTargetMinKotsVersions(t *testing.T) {

	tests := []struct {
		name      string
		specFiles SpecFiles
		expect    []LintExpression
	}{
		{
			name: "valid target and min version with 'v' prefix",
			specFiles: SpecFiles{
				{
					Path: "",
					Content: `apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: validVersions
spec:
  targetKotsVersion: "v1.64.0"
  minKotsVersion: "v1.59.0"`,
				},
			},
			expect: []LintExpression{},
		},
		{
			name: "valid target and min version without 'v' prefix",
			specFiles: SpecFiles{
				{
					Path: "",
					Content: `apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: validVersions
spec:
  targetKotsVersion: "1.64.0"
  minKotsVersion: "1.59.0"`,
				},
			},
			expect: []LintExpression{},
		},
		{
			name: "valid spec without target nor min versions defined",
			specFiles: SpecFiles{
				{
					Path: "test.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Application
metadata:
name: app-slug
spec:
title: App Name
kustomizeVersion: "3.5.4"
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
			name: "invalid target version",
			specFiles: SpecFiles{
				{
					Path: "",
					Content: `apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: invalidTargetVersion
spec:
  targetKotsVersion: "1000.0.0"
  minKotsVersion: "1.60.0"
`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "non-existent-target-kots-version",
					Type:    "error",
					Message: "Target KOTS version not found",
				},
			},
		},
		{
			name: "invalid min version",
			specFiles: SpecFiles{
				{
					Path: "",
					Content: `apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: invalidTargetVersion
spec:
  targetKotsVersion: "1.64.0"
  minKotsVersion: "1000.0.0"
`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "non-existent-min-kots-version",
					Type:    "error",
					Message: "Minimum KOTS version not found",
				},
			},
		},
		{
			name: "invalid target and min version",
			specFiles: SpecFiles{
				{
					Path: "",
					Content: `apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: invalidTargetVersion
spec:
  targetKotsVersion: "1000.0.0"
  minKotsVersion: "1000.0.0"
`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "non-existent-target-kots-version",
					Type:    "error",
					Message: "Target KOTS version not found",
				}, {
					Rule:    "non-existent-min-kots-version",
					Type:    "error",
					Message: "Minimum KOTS version not found",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := lintTargetMinKotsVersions(test.specFiles)
			require.NoError(t, err)
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
		{
			name: "detects helmchart crd that has invalid schema until rendered",
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
      - name: helm_values
        type: textarea
        title: Helm Values
        value: |
          key1: value1
          key2: value2
      - name: helm_values_optional
        type: textarea
        title: Optional Helm Values
        value: |
          optkey1: optvalue1
          optkey2: optvalue2
`,
				},
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
    chartVersion: 10.3.5

  # values are used in the customer environment, as a pre-render step
  # these values will be supplied to helm template
  values: repl{{ConfigOption "helm_values" | nindent 4}}

  optionalValues:
  - when: "true"
    recursiveMerge: true
    values: repl{{ConfigOption "helm_values_optional" | nindent 8}}

  # builder values provide a way to render the chart with all images
  # and manifests. this is used in replicated to create airgap packages
  builder: {}
`,
				},
			},
			expect: []LintExpression{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tarGzFiles := SpecFiles{}
			yamlFiles := SpecFiles{}
			for _, file := range test.specFiles {
				if file.isTarGz() {
					tarGzFiles = append(tarGzFiles, file)
				}
				if file.isYAML() {
					yamlFiles = append(yamlFiles, file)
				}
			}

			renderedFiles, err := yamlFiles.render()
			require.NoError(t, err)

			actual, err := lintHelmCharts(renderedFiles, tarGzFiles)
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
		{
			name: "kubeval basic k8s kinds no errors",
			specFiles: SpecFiles{
				{
					Name: "deployment.yaml",
					Path: "deployment.yaml",
					Content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80`,
				},
				{
					Name: "service.yaml",
					Path: "service.yaml",
					Content: `apiVersion: v1
kind: Service
metadata:
  name: nginx-service
spec:
  selector:
    app: nginx
  ports:
  - name: nginx-port
    protocol: TCP
    port: 80
    targetPort: 80`,
				},
				{
					Name: "statefulset.yaml",
					Path: "statefulset.yaml",
					Content: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: web
spec:
  selector:
    matchLabels:
      app: nginx # has to match .spec.template.metadata.labels
  serviceName: "nginx"
  replicas: 3 # by default is 1
  minReadySeconds: 10 # by default is 0
  template:
    metadata:
      labels:
        app: nginx # has to match .spec.selector.matchLabels
    spec:
      terminationGracePeriodSeconds: 10
      containers:
      - name: nginx
        image: k8s.gcr.io/nginx-slim:0.8
        ports:
        - containerPort: 80
          name: web
        volumeMounts:
        - name: www
          mountPath: /usr/share/nginx/html
  volumeClaimTemplates:
  - metadata:
      name: www
    spec:
      accessModes: [ "ReadWriteOnce" ]
      storageClassName: "my-storage-class"
      resources:
        requests:
          storage: 1Gi`,
				},
				{
					Name: "job.yaml",
					Path: "job.yaml",
					Content: `apiVersion: batch/v1
kind: Job
metadata:
  name: pi
spec:
  template:
    spec:
      containers:
      - name: pi
        image: perl
        command: ["perl",  "-Mbignum=bpi", "-wle", "print bpi(2000)"]
      restartPolicy: Never
  backoffLimit: 4`,
				},
				{
					Name: "cronjob.yaml",
					Path: "cronjob.yaml",
					Content: `apiVersion: batch/v1beta1
kind: CronJob
metadata:
  name: hello
  labels:
    app: example
    component: cronjob
spec:
  schedule: "* */1 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: hello
            image: busybox
            args:
            - /bin/sh
            - -c
            - date; echo Hello
          restartPolicy: OnFailure`,
				},
				{
					Name: "serviceaccount.yaml",
					Path: "serviceaccount.yaml",
					Content: `apiVersion: v1
kind: ServiceAccount
metadata:
  name: qakots-backup
  annotations:
    key: val
  labels:
    app.kubernetes.io/name: qakots-backup`,
				},
				{
					Name: "role.yaml",
					Path: "role.yaml",
					Content: `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: qakots-backup
rules:
- apiGroups: ["apps"]
  resources: ["deployments"]
  verbs: ["create", "update", "patch", "get", "list", "watch"]`,
				},
				{
					Name: "rolebinding.yaml",
					Path: "rolebinding.yaml",
					Content: `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: qakots-backup-binding
subjects:
- kind: ServiceAccount
  name: qakots-backup
roleRef:
  kind: Role
  name: qakots-backup
  apiGroup: rbac.authorization.k8s.io`,
				},
				{
					Name: "namespace.yaml",
					Path: "namespace.yaml",
					Content: `apiVersion: v1
kind: Namespace
metadata:
  name: test
spec: {}`,
				},
				{
					Name: "daemonset.yaml",
					Path: "daemonset.yaml",
					Content: `apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: fluentd-elasticsearch
  namespace: kube-system
  labels:
    k8s-app: fluentd-logging
spec:
  selector:
    matchLabels:
      name: fluentd-elasticsearch
  template:
    metadata:
      labels:
        name: fluentd-elasticsearch
    spec:
      tolerations:
      # this toleration is to have the daemonset runnable on master nodes
      # remove it if your masters can't run pods
      - key: node-role.kubernetes.io/master
        operator: Exists
        effect: NoSchedule
      containers:
      - name: fluentd-elasticsearch
        image: quay.io/fluentd_elasticsearch/fluentd:v2.5.2
        resources:
          limits:
            memory: 200Mi
          requests:
            cpu: 100m
            memory: 200Mi
        volumeMounts:
        - name: varlog
          mountPath: /var/log
        - name: varlibdockercontainers
          mountPath: /var/lib/docker/containers
          readOnly: true
      terminationGracePeriodSeconds: 30
      volumes:
      - name: varlog
        hostPath:
          path: /var/log
      - name: varlibdockercontainers
        hostPath:
          path: /var/lib/docker/containers`,
				},
				{
					Name: "pvc.yaml",
					Path: "pvc.yaml",
					Content: `apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pvc
  labels:
    app: pvc
spec:
  accessModes:
    - "ReadWriteOnce"
  resources:
    requests:
      storage: "100Gi"`,
				},
				{
					Name: "configmap.yaml",
					Path: "configmap.yaml",
					Content: `apiVersion: v1
kind: ConfigMap
metadata:
  name: config
  labels:
    app: config
data:
  hello: world`,
				},
				{
					Name: "secret.yaml",
					Path: "secret.yaml",
					Content: `apiVersion: v1
kind: Secret
metadata:
  name: secret
  labels:
    app: secret
type: Opaque
data:
  sentry-secret: secret123
  smtp-password: password_secret
  user-password: password123`,
				},
				{
					Name: "ingress-extensions-v1beta1.yaml",
					Path: "ingress-extensions-v1beta1.yaml",
					Content: `apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: example-nginx-ingress
spec:
  rules:
  - http:
      paths:
        - path: /
          backend:
            serviceName: example-nginx
            servicePort: 80`,
				},
				{
					Name: "ingress-networking-v1beta1.yaml",
					Path: "ingress-networking-v1beta1.yaml",
					Content: `apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  name: example-nginx-ingress
spec:
  rules:
  - http:
      paths:
        - path: /
          backend:
            serviceName: example-nginx
            servicePort: 80`,
				},
				{
					Name: "ingress-networking-v1.yaml",
					Path: "ingress-networking-v1.yaml",
					Content: `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example-nginx-ingress
spec:
  rules:
  - http:
      paths:
        - path: /
          pathType: Prefix
          backend:
            service:
              name: example-nginx
              port:
                number: 80`,
				},
			},
			expect: []LintExpression{},
		},
		{
			name: "kubeval basic replicated kinds no errors",
			specFiles: SpecFiles{
				{
					Name: "replicated-app.yaml",
					Path: "replicated-app.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: my-application
spec:
  title: "My Application"
  icon: https://support.io/img/logo.png
  releaseNotes: These are our release notes
  allowRollback: false
  kubectlVersion: latest
  kustomizeVersion: latest
  targetKotsVersion: "1.60.0"
  minKotsVersion: "1.40.0"
  requireMinimalRBACPrivileges: false
  additionalImages:
    - jenkins/jenkins:lts
  additionalNamespaces:
    - "*"
  ports:
    - serviceName: web
      servicePort: 9000
      localPort: 9000
      applicationUrl: "http://web"
  statusInformers:
    - deployment/my-web-svc
    - deployment/my-worker
  graphs:
    - title: User Signups
      query: 'sum(user_signup_events_total)'`,
				},
				{
					Name: "application.yaml",
					Path: "application.yaml",
					Content: `apiVersion: app.k8s.io/v1beta1
kind: Application
metadata:
  name: "my-app"
  labels:
    app.kubernetes.io/name: "my-app"
    app.kubernetes.io/version: "9.1.1"
spec:
  selector:
    matchLabels:
     app.kubernetes.io/name: "my-app"
  componentKinds: []
  descriptor:
    version: "9.1.1"
    description: "Open-source error tracking with full stacktraces & asynchronous context."
    icons:
      - src: "https://sentry-brand.storage.googleapis.com/sentry-glyph-black.png"
        type: "image/png"
    type: "sentry"
    links:
      - description: Open Sentry Enterprise
        url: "http://sentry"`,
				},
				{
					Name: "config.yaml",
					Path: "config.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: my-config
spec:
  groups:
  - name: authentication
    title: Authentication
    description: Configure application authentication below.
    items:
    - name: email-address
      title: Email Address
      type: text
    - name: password_text
      title: Password Text
      type: password
      value: "{{repl RandomString 10}}"`,
				},
				{
					Name: "preflight.yaml",
					Path: "preflight.yaml",
					Content: `apiVersion: troubleshoot.replicated.com/v1beta1
kind: Preflight
metadata:
  name: preflight
spec:
  analyzers:
    - imagePullSecret:
        checkName: Access to index.docker.io
        registryName: index.docker.io
        outcomes:
          - fail:
              message: Could not find index.docker.io imagePullSecret
          - pass:
              message: Found credentials to pull private images from index.docker.io
`,
				},
				{
					Name: "redactor.yaml",
					Path: "redactor.yaml",
					Content: `apiVersion: troubleshoot.sh/v1beta2
kind: Redactor
metadata:
  name: my-redactor-name
spec:
  redactors:
  - name: replace password # names are not used internally, but are useful for recordkeeping
    fileSelector:
      file: data/my-password-dump # this targets a single file
    removals:
      values:
      - abc123 # this value is my password, and should never appear in a support bundle
  - name: all files # as no file is specified, this redactor will run against all files
    removals:
      regex:
      - redactor: (another)(?P<mask>.*)(here)
      - selector: 'S3_ENDPOINT' # remove the value in lines following those that contain the string S3_ENDPOINT
        redactor: '("value": ").*(")'
      yamlPath:
      - "abc.xyz.*" # redact all items in the array at key xyz within key abc in yaml documents`,
				},
				{
					Name: "supportbundle.yaml",
					Path: "supportbundle.yaml",
					Content: `apiVersion: troubleshoot.sh/v1beta2
kind: SupportBundle
metadata:
  name: support-bundle
spec:
  collectors:
    - secret:
        name: myapp-postgres
        key: uri
        includeValue: false
  analyzers:
    - imagePullSecret:
        checkName: Access to index.docker.io
        registryName: index.docker.io
        outcomes:
          - fail:
              message: Could not find index.docker.io imagePullSecret
          - pass:
              message: Found credentials to pull private images from index.docker.io
`,
				},
				{
					Name: "helmchart.yaml",
					Path: "helmchart.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: HelmChart
metadata:
  name: samplechart
spec:
  chart:
    name: samplechart
    chartVersion: 3.1.7
  exclude: "false"
  helmVersion: v2
  useHelmInstall: true
  values:
    postgresql:
      enabled: true
  namespace: samplechart-namespace
  builder:
    postgresql:
      enabled: true`,
				},
				{
					Name: "backup.yaml",
					Path: "backup.yaml",
					Content: `apiVersion: velero.io/v1
kind: Backup
metadata:
  name: backup
spec: {}`,
				},
				{
					Name: "identity.yaml",
					Path: "identity.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Identity
metadata:
  name: my-application
spec:
    identityIssuerURL: https://{{repl ConfigOption "ingress_hostname"}}/dex
    oidcRedirectUris:
      - https://{{repl ConfigOption "ingress_hostname"}}/oidc/login/callback
    supportedProviders: [ oidc ]
    requireIdentityProvider: true
    roles:
      - id: member
        name: Member
        description: Can see every member and non-secret team in the organization.
      - id: owner
        name: Owner
        description: Has full administrative access to the entire organization.
    oauth2AlwaysShowLoginScreen: false
    signingKeysExpiration: 6h
    idTokensExpiration: 24h
    webConfig:
      title: My App
      theme:
        logoUrl: data:image/png;base64,<encoded_base64_stream>
        logoBase64: <base64 encoded png file>
        styleCssBase64: <base64 encoded [styles.css](https://github.com/dexidp/dex/blob/v2.27.0/web/themes/coreos/styles.css) file>
        faviconBase64: <base64 encoded png file>
`,
				},
			},
			expect: []LintExpression{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			separatedSpecFiles, err := test.specFiles.separate()
			require.NoError(t, err)

			renderedFiles, err := separatedSpecFiles.render()
			require.NoError(t, err)

			actual, err := lintWithKubevalSchema(renderedFiles, test.specFiles, "file://../../kubernetes_json_schema/schema")
			require.NoError(t, err)
			assert.ElementsMatch(t, actual, test.expect)
		})
	}
}

func Test_lintRenderContent(t *testing.T) {
	tests := []struct {
		name          string
		specFiles     SpecFiles
		renderedFiles SpecFiles
		expect        []LintExpression
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
			renderedFiles: SpecFiles{
				{
					Name:     "config.yaml",
					Path:     "config.yaml",
					DocIndex: 0,
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
					Name:     "test.yaml",
					Path:     "test.yaml",
					DocIndex: 0,
					Content: `apiVersion: v1
kind: ConfigMap
metadata:
  name: example-config
data:
    ENV_VAR_1: "fake"
    ENV_VAR_2: 'value'`,
				},
				{
					Name:     "test.yaml",
					Path:     "test.yaml",
					DocIndex: 1,
					Content: `apiVersion: v1
kind: ConfigMap
metadata:
  name: example-config-2
data:
    ENV_VAR_1: "fake"
    ENV_VAR_2: 'value'`,
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
			renderedFiles: SpecFiles{
				{
					Name:     "config.yaml",
					Path:     "config.yaml",
					DocIndex: 0,
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
					Name:     "test.yaml",
					Path:     "test.yaml",
					DocIndex: 0,
					Content: `apiVersion: v1
kind: ConfigMap
metadata:
  name: example-config
data:
    ENV_VAR_1: "fake"
    ENV_VAR_2: ''`,
				},
				{
					Name:     "test.yaml",
					Path:     "test.yaml",
					DocIndex: 1,
					Content: `apiVersion: v1
kind: ConfigMap
metadata:
  name: example-config-2
data:
    ENV_VAR_1: "fake"
    ENV_VAR_2: ''`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "config-is-invalid",
					Type:    "error",
					Path:    "config.yaml",
					Message: `failed to decode config content: json: cannot unmarshal number into Go struct field ConfigItem.spec.groups.items.title of type string`,
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
			renderedFiles: SpecFiles{
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
			renderedFiles: SpecFiles{
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
			actual, renderedFiles, err := lintRenderContent(test.specFiles)
			require.NoError(t, err)
			assert.ElementsMatch(t, actual, test.expect)
			assert.ElementsMatch(t, renderedFiles, test.renderedFiles)
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
			name: "repeatable config options",
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
        - name: repeatable_value
          title: a text field with a value provided by a template function
          type: text
          repeatable: true
          templates:
          - name: example-config
          valuesByGroup:
            example_settings:
              key: value
        - name: non_repeatable_value
          title: a text field with a value provided by a template function
          type: text
          value: hello
          default: goodbye`,
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
    ENV_VAR_2: '{{repl ConfigOption "[[repl .repeatable_value ]]" }}'
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: example-config-2
data:
    ENV_VAR_1: '{{repl ConfigOption "[[repl .non_existing_item ]]" }}'
    ENV_VAR_2: '{{repl ConfigOption "[[repl .non_repeatable_value ]]" }}'`,
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
					Rule:    "config-option-not-repeatable",
					Type:    "error",
					Path:    "test.yaml",
					Message: "Config option \"non_repeatable_value\" not repeatable",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 15,
							},
						},
					},
				},
				{
					Rule:    "config-option-not-repeatable",
					Type:    "error",
					Path:    "test.yaml",
					Message: "Config option \"non_existing_item\" not repeatable",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 14,
							},
						},
					},
				},
				{
					Rule:    "config-option-not-found",
					Type:    "warn",
					Path:    "test.yaml",
					Message: "Config option \"non_existing_item\" not found",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 14,
							},
						},
					},
				},
			},
		},
		{
			name: "repeatable config spec",
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
        - name: good_repeatable_entry
          title: a text field with a value provided by a template function
          type: text
          repeatable: true
          templates:
          - name: example-config
            yamlPath: this.is.fine[0]
          valuesByGroup:
            example_settings:
              key: value
        - name: missing_template
          title: a text field with a value provided by a template function
          type: text
          repeatable: true
          valuesByGroup:
            example_settings:
              key: value
        - name: missing_valuesByGroup
          title: a text field with a value provided by a template function
          type: text
          repeatable: true
          templates:
            name: example-config
        - name: bad_yamlPath
          title: a text field with a value provided by a template function
          type: text
          repeatable: true
          templates:
          - name: example-config
            yamlPath: this.is[0].missing[1].ending.array
          valuesByGroup:
            example_settings:
              key: value`,
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
					Rule:    "repeat-option-missing-template",
					Type:    "error",
					Path:    "config.yaml",
					Message: "Repeatable Config option \"missing_template\" has an incomplete template target",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 23,
							},
						},
					},
				},
				{
					Rule:    "repeat-option-missing-valuesByGroup",
					Type:    "error",
					Path:    "config.yaml",
					Message: "Repeatable Config option \"missing_valuesByGroup\" has an incomplete valuesByGroup",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 30,
							},
						},
					},
				},
				{
					Rule:    "repeat-option-malformed-yamlpath",
					Type:    "error",
					Path:    "config.yaml",
					Message: "Repeatable Config option \"bad_yamlPath\" yamlPath does not end with an array",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 36,
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
		{
			name: "leading v is valid in target/min kots versions",
			specFiles: SpecFiles{
				validExampleNginxDeploymentSpecFile,
				{
					Name: "replicated-app.yaml",
					Path: "replicated-app.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-slug
spec:
  title: App Name
  icon: https://github.com/cncf/artwork/blob/master/projects/kubernetes/icon/color/kubernetes-icon-color.png
  minKotsVersion: v1.50.0
  targetKotsVersion: v1.60.0
  statusInformers:
    - deployment/example-nginx
`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "preflight-spec",
					Type:    "warn",
					Message: "Missing preflight spec",
				},
				{
					Rule:    "config-spec",
					Type:    "warn",
					Message: "Missing config spec",
				},
				{
					Rule:    "troubleshoot-spec",
					Type:    "warn",
					Message: "Missing troubleshoot spec",
				},
			},
		},
		{
			name: "target kots version must be a valid semver",
			specFiles: SpecFiles{
				validExampleNginxDeploymentSpecFile,
				{
					Name: "replicated-app.yaml",
					Path: "replicated-app.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-slug
spec:
  title: App Name
  icon: https://github.com/cncf/artwork/blob/master/projects/kubernetes/icon/color/kubernetes-icon-color.png
  targetKotsVersion: vv1.50.0
  statusInformers:
    - deployment/example-nginx	
`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "preflight-spec",
					Type:    "warn",
					Message: "Missing preflight spec",
				},
				{
					Rule:    "config-spec",
					Type:    "warn",
					Message: "Missing config spec",
				},
				{
					Rule:    "troubleshoot-spec",
					Type:    "warn",
					Message: "Missing troubleshoot spec",
				},
				{
					Rule:    "invalid-target-kots-version",
					Type:    "error",
					Message: "Target KOTS version must be a valid semver",
					Path:    "replicated-app.yaml",
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
			name: "min kots version must be a valid semver",
			specFiles: SpecFiles{
				validExampleNginxDeploymentSpecFile,
				{
					Name: "replicated-app.yaml",
					Path: "replicated-app.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-slug
spec:
  title: App Name
  icon: https://github.com/cncf/artwork/blob/master/projects/kubernetes/icon/color/kubernetes-icon-color.png
  minKotsVersion: vv1.4s0.0
  statusInformers:
    - deployment/example-nginx
`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "preflight-spec",
					Type:    "warn",
					Message: "Missing preflight spec",
				},
				{
					Rule:    "config-spec",
					Type:    "warn",
					Message: "Missing config spec",
				},
				{
					Rule:    "troubleshoot-spec",
					Type:    "warn",
					Message: "Missing troubleshoot spec",
				},
				{
					Rule:    "invalid-min-kots-version",
					Type:    "error",
					Message: "Minimum KOTS version must be a valid semver",
					Path:    "replicated-app.yaml",
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
			name: "invalid helm release name - contains a space",
			specFiles: SpecFiles{
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
    releaseName: invalid name

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
					Rule:    "preflight-spec",
					Type:    "warn",
					Message: "Missing preflight spec",
				},
				{
					Rule:    "config-spec",
					Type:    "warn",
					Message: "Missing config spec",
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
					Rule:    "invalid-helm-release-name",
					Path:    "redis-10.3.5.tar.gz/redis.yaml",
					Type:    "error",
					Message: "Invalid Helm release name, must match regex ^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$ and the length must not be longer than 53",
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
			name: "invalid helm release name - no capital letters",
			specFiles: SpecFiles{
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
    releaseName: invalid-Name

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
					Rule:    "preflight-spec",
					Type:    "warn",
					Message: "Missing preflight spec",
				},
				{
					Rule:    "config-spec",
					Type:    "warn",
					Message: "Missing config spec",
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
					Rule:    "invalid-helm-release-name",
					Path:    "redis-10.3.5.tar.gz/redis.yaml",
					Type:    "error",
					Message: "Invalid Helm release name, must match regex ^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$ and the length must not be longer than 53",
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
			name: "invalid helm release name - invalid characters",
			specFiles: SpecFiles{
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
    releaseName: inval$$id-name

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
					Rule:    "preflight-spec",
					Type:    "warn",
					Message: "Missing preflight spec",
				},
				{
					Rule:    "config-spec",
					Type:    "warn",
					Message: "Missing config spec",
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
					Rule:    "invalid-helm-release-name",
					Path:    "redis-10.3.5.tar.gz/redis.yaml",
					Type:    "error",
					Message: "Invalid Helm release name, must match regex ^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$ and the length must not be longer than 53",
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
			name: "duplicate helm release name",
			specFiles: SpecFiles{
				{
					Name: "redis-release-1.yaml",
					Path: "redis-10.3.5.tar.gz/redis-release-1.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: HelmChart
metadata:
  name: redis-release-1
spec:
  # chart identifies a matching chart from a .tgz
  chart:
    name: redis
    chartVersion: 10.3.5
    releaseName: redis

  # values are used in the customer environment, as a pre-render step
  # these values will be supplied to helm template
  values: {}

  # builder values provide a way to render the chart with all images
  # and manifests. this is used in replicated to create airgap packages
  builder: {}
`,
				},
				{
					Name: "redis-release-2.yaml",
					Path: "redis-10.3.5.tar.gz/redis-release-2.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: HelmChart
metadata:
  name: redis-release-2
spec:
  # chart identifies a matching chart from a .tgz
  chart:
    name: redis
    chartVersion: 10.3.5
    releaseName: redis

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
					Rule:    "preflight-spec",
					Type:    "warn",
					Message: "Missing preflight spec",
				},
				{
					Rule:    "config-spec",
					Type:    "warn",
					Message: "Missing config spec",
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
					Rule:    "duplicate-helm-release-name",
					Path:    "redis-10.3.5.tar.gz/redis-release-1.yaml",
					Type:    "error",
					Message: "Release name is already used in redis-10.3.5.tar.gz/redis-release-2.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 10,
							},
						},
					},
				},
				{
					Rule:    "duplicate-helm-release-name",
					Path:    "redis-10.3.5.tar.gz/redis-release-2.yaml",
					Type:    "error",
					Message: "Release name is already used in redis-10.3.5.tar.gz/redis-release-1.yaml",
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
			name: "unique helm release names - no errors",
			specFiles: SpecFiles{
				{
					Name: "redis-release-1.yaml",
					Path: "redis-10.3.5.tar.gz/redis-release-1.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: HelmChart
metadata:
  name: redis-release-1
spec:
  # chart identifies a matching chart from a .tgz
  chart:
    name: redis
    chartVersion: 10.3.5
    releaseName: redis-release-1

  # values are used in the customer environment, as a pre-render step
  # these values will be supplied to helm template
  values: {}

  # builder values provide a way to render the chart with all images
  # and manifests. this is used in replicated to create airgap packages
  builder: {}
`,
				},
				{
					Name: "redis-release-2.yaml",
					Path: "redis-10.3.5.tar.gz/redis-release-2.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: HelmChart
metadata:
  name: redis-release-2
spec:
  # chart identifies a matching chart from a .tgz
  chart:
    name: redis
    chartVersion: 10.3.5
    releaseName: redis-release-2

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
					Rule:    "preflight-spec",
					Type:    "warn",
					Message: "Missing preflight spec",
				},
				{
					Rule:    "config-spec",
					Type:    "warn",
					Message: "Missing config spec",
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
			name: "up-to-date and valid kubernetes installer",
			specFiles: SpecFiles{
				{
					Name: "installer.yaml",
					Path: "installer.yaml",
					Content: `apiVersion: cluster.kurl.sh/v1beta1
kind: Installer
metadata:
  name: my-installer
spec:
  contour:
    version: 0.14.0
  kotsadm:
    version: 1.70.0
  kubernetes:
    version: 1.23.5
  registry:
    version: 2.7.1
  rook:
    version: 1.4.3
  weave:
    version: 2.5.2
`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "preflight-spec",
					Type:    "warn",
					Message: "Missing preflight spec",
				},
				{
					Rule:    "config-spec",
					Type:    "warn",
					Message: "Missing config spec",
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
			name: "deprecated kubernetes installer api version",
			specFiles: SpecFiles{
				{
					Name: "installer.yaml",
					Path: "installer.yaml",
					Content: `apiVersion: kurl.sh/v1beta1
kind: Installer
metadata:
  name: my-installer
spec:
  contour:
    version: 0.14.0
  kotsadm:
    version: 1.70.0
  kubernetes:
    version: 1.23.5
  registry:
    version: 2.7.1
  rook:
    version: 1.4.3
  weave:
    version: 2.5.2
`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "preflight-spec",
					Type:    "warn",
					Message: "Missing preflight spec",
				},
				{
					Rule:    "config-spec",
					Type:    "warn",
					Message: "Missing config spec",
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
					Rule:    "deprecated-kubernetes-installer-version",
					Path:    "installer.yaml",
					Type:    "warn",
					Message: "API version 'kurl.sh/v1beta1' is deprecated. Use 'cluster.kurl.sh/v1beta1' instead.",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
			},
		},
		{
			name: "invalid kubernetes installer in release",
			specFiles: SpecFiles{
				{
					Name: "installer.yaml",
					Path: "installer.yaml",
					Content: `apiVersion: cluster.kurl.sh/v1beta1
kind: Installer
metadata:
  name: my-installer
spec:
  contour:
    version: 0.14.0
  kotsadm:
    version: 1.70.0
  kubernetes:
    version: 1.23.x
  registry:
    version: 2.7.1
  rook:
    version: latest
  weave:
    version: 2.5.2
`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "preflight-spec",
					Type:    "warn",
					Message: "Missing preflight spec",
				},
				{
					Rule:    "config-spec",
					Type:    "warn",
					Message: "Missing config spec",
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
					Rule:    "invalid-kubernetes-installer",
					Path:    "installer.yaml",
					Type:    "error",
					Message: "Add-ons included in the Kubernetes installer must pin specific versions rather than 'latest' or x-ranges (e.g., 1.2.x).",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 11,
							},
						},
					},
				},
				{
					Rule:    "invalid-kubernetes-installer",
					Path:    "installer.yaml",
					Type:    "error",
					Message: "Add-ons included in the Kubernetes installer must pin specific versions rather than 'latest' or x-ranges (e.g., 1.2.x).",
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
			name: "duplicate kots kinds in release",
			specFiles: SpecFiles{
				validExampleNginxDeploymentSpecFile,
				{
					Name: "config-1.yaml",
					Path: "config-1.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Config
`,
				},
				{
					Name: "config-2.yaml",
					Path: "config-2.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Config
`,
				},
				{
					Name: "replicated-app-1.yaml",
					Path: "replicated-app-1.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Application
spec:
  icon: https://github.com/cncf/artwork/blob/master/projects/kubernetes/icon/color/kubernetes-icon-color.png
  statusInformers:
    - deployment/example-nginx
`,
				},
				{
					Name: "replicated-app-2.yaml",
					Path: "replicated-app-2.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Application
spec:
  icon: https://github.com/cncf/artwork/blob/master/projects/kubernetes/icon/color/kubernetes-icon-color.png
  statusInformers:
    - deployment/example-nginx
`,
				},
				{
					Name: "identity-1.yaml",
					Path: "identity-1.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Identity
`,
				},
				{
					Name: "identity-2.yaml",
					Path: "identity-2.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Identity
`,
				},
				{
					Name: "collector-1.yaml",
					Path: "collector-1.yaml",
					Content: `apiVersion: troubleshoot.sh/v1beta2
kind: Collector
`,
				},
				{
					Name: "collector-2.yaml",
					Path: "collector-2.yaml",
					Content: `apiVersion: troubleshoot.sh/v1beta2
kind: Collector
`,
				},
				{
					Name: "analyzer-1.yaml",
					Path: "analyzer-1.yaml",
					Content: `apiVersion: troubleshoot.sh/v1beta2
kind: Analyzer
`,
				},
				{
					Name: "analyzer-2.yaml",
					Path: "analyzer-2.yaml",
					Content: `apiVersion: troubleshoot.sh/v1beta2
kind: Analyzer
`,
				},
				{
					Name: "support-bundle-1.yaml",
					Path: "support-bundle-1.yaml",
					Content: `apiVersion: troubleshoot.sh/v1beta2
kind: SupportBundle
`,
				},
				{
					Name: "support-bundle-2.yaml",
					Path: "support-bundle-2.yaml",
					Content: `apiVersion: troubleshoot.sh/v1beta2
kind: SupportBundle
`,
				},
				{
					Name: "redactor-1.yaml",
					Path: "redactor-1.yaml",
					Content: `apiVersion: troubleshoot.sh/v1beta2
kind: Redactor
`,
				},
				{
					Name: "redactor-2.yaml",
					Path: "redactor-2.yaml",
					Content: `apiVersion: troubleshoot.sh/v1beta2
kind: Redactor
`,
				},
				{
					Name: "preflight-1.yaml",
					Path: "preflight-1.yaml",
					Content: `apiVersion: troubleshoot.sh/v1beta2
kind: Preflight
`,
				},
				{
					Name: "preflight-2.yaml",
					Path: "preflight-2.yaml",
					Content: `apiVersion: troubleshoot.sh/v1beta2
kind: Preflight
`,
				},
				{
					Name: "backup-1.yaml",
					Path: "backup-1.yaml",
					Content: `apiVersion: velero.io/v1
kind: Backup
`,
				},
				{
					Name: "backup-2.yaml",
					Path: "backup-2.yaml",
					Content: `apiVersion: velero.io/v1
kind: Backup
`,
				},
				{
					Name: "k8s-app-1.yaml",
					Path: "k8s-app-1.yaml",
					Content: `apiVersion: app.k8s.io/v1beta1
kind: Application
`,
				},
				{
					Name: "k8s-app-2.yaml",
					Path: "k8s-app-2.yaml",
					Content: `apiVersion: app.k8s.io/v1beta1
kind: Application
`,
				},
				{
					Name: "installer-1.yaml",
					Path: "installer-1.yaml",
					Content: `apiVersion: cluster.kurl.sh/v1beta1
kind: Installer
`,
				},
				{
					Name: "installer-2.yaml",
					Path: "installer-2.yaml",
					Content: `apiVersion: cluster.kurl.sh/v1beta1
kind: Installer
`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "duplicate-kots-kind",
					Path:    "config-1.yaml",
					Type:    "error",
					Message: "A release can only include one 'Config' resource, but another 'Config' resource was found in config-2.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "config-2.yaml",
					Type:    "error",
					Message: "A release can only include one 'Config' resource, but another 'Config' resource was found in config-1.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "replicated-app-1.yaml",
					Type:    "error",
					Message: "A release can only include one 'Application' resource, but another 'Application' resource was found in replicated-app-2.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "replicated-app-2.yaml",
					Type:    "error",
					Message: "A release can only include one 'Application' resource, but another 'Application' resource was found in replicated-app-1.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "identity-1.yaml",
					Type:    "error",
					Message: "A release can only include one 'Identity' resource, but another 'Identity' resource was found in identity-2.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "identity-2.yaml",
					Type:    "error",
					Message: "A release can only include one 'Identity' resource, but another 'Identity' resource was found in identity-1.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "collector-1.yaml",
					Type:    "error",
					Message: "A release can only include one 'Collector' resource, but another 'Collector' resource was found in collector-2.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "collector-2.yaml",
					Type:    "error",
					Message: "A release can only include one 'Collector' resource, but another 'Collector' resource was found in collector-1.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "analyzer-1.yaml",
					Type:    "error",
					Message: "A release can only include one 'Analyzer' resource, but another 'Analyzer' resource was found in analyzer-2.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "analyzer-2.yaml",
					Type:    "error",
					Message: "A release can only include one 'Analyzer' resource, but another 'Analyzer' resource was found in analyzer-1.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "support-bundle-1.yaml",
					Type:    "error",
					Message: "A release can only include one 'SupportBundle' resource, but another 'SupportBundle' resource was found in support-bundle-2.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "support-bundle-2.yaml",
					Type:    "error",
					Message: "A release can only include one 'SupportBundle' resource, but another 'SupportBundle' resource was found in support-bundle-1.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "redactor-1.yaml",
					Type:    "error",
					Message: "A release can only include one 'Redactor' resource, but another 'Redactor' resource was found in redactor-2.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "redactor-2.yaml",
					Type:    "error",
					Message: "A release can only include one 'Redactor' resource, but another 'Redactor' resource was found in redactor-1.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "preflight-1.yaml",
					Type:    "error",
					Message: "A release can only include one 'Preflight' resource, but another 'Preflight' resource was found in preflight-2.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "preflight-2.yaml",
					Type:    "error",
					Message: "A release can only include one 'Preflight' resource, but another 'Preflight' resource was found in preflight-1.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "backup-1.yaml",
					Type:    "error",
					Message: "A release can only include one 'Backup' resource, but another 'Backup' resource was found in backup-2.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "backup-2.yaml",
					Type:    "error",
					Message: "A release can only include one 'Backup' resource, but another 'Backup' resource was found in backup-1.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "k8s-app-1.yaml",
					Type:    "error",
					Message: "A release can only include one 'Application' resource, but another 'Application' resource was found in k8s-app-2.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "k8s-app-2.yaml",
					Type:    "error",
					Message: "A release can only include one 'Application' resource, but another 'Application' resource was found in k8s-app-1.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "installer-1.yaml",
					Type:    "error",
					Message: "A release can only include one 'Installer' resource, but another 'Installer' resource was found in installer-2.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "installer-2.yaml",
					Type:    "error",
					Message: "A release can only include one 'Installer' resource, but another 'Installer' resource was found in installer-1.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
			},
		},
		{
			name: "duplicate kots kinds in release with different api versions",
			specFiles: SpecFiles{
				{
					Name: "collector-1.yaml",
					Path: "collector-1.yaml",
					Content: `apiVersion: troubleshoot.replicated.com/v1beta1
kind: Collector
`,
				},
				{
					Name: "collector-2.yaml",
					Path: "collector-2.yaml",
					Content: `apiVersion: troubleshoot.sh/v1beta2
kind: Collector
`,
				},
				{
					Name: "analyzer-1.yaml",
					Path: "analyzer-1.yaml",
					Content: `apiVersion: troubleshoot.replicated.com/v1beta1
kind: Analyzer
`,
				},
				{
					Name: "analyzer-2.yaml",
					Path: "analyzer-2.yaml",
					Content: `apiVersion: troubleshoot.sh/v1beta2
kind: Analyzer
`,
				},
				{
					Name: "support-bundle-1.yaml",
					Path: "support-bundle-1.yaml",
					Content: `apiVersion: troubleshoot.replicated.com/v1beta1
kind: SupportBundle
`,
				},
				{
					Name: "support-bundle-2.yaml",
					Path: "support-bundle-2.yaml",
					Content: `apiVersion: troubleshoot.sh/v1beta2
kind: SupportBundle
`,
				},
				{
					Name: "redactor-1.yaml",
					Path: "redactor-1.yaml",
					Content: `apiVersion: troubleshoot.replicated.com/v1beta1
kind: Redactor
`,
				},
				{
					Name: "redactor-2.yaml",
					Path: "redactor-2.yaml",
					Content: `apiVersion: troubleshoot.sh/v1beta2
kind: Redactor
`,
				},
				{
					Name: "preflight-1.yaml",
					Path: "preflight-1.yaml",
					Content: `apiVersion: troubleshoot.replicated.com/v1beta1
kind: Preflight
`,
				},
				{
					Name: "preflight-2.yaml",
					Path: "preflight-2.yaml",
					Content: `apiVersion: troubleshoot.sh/v1beta2
kind: Preflight
`,
				},
				{
					Name: "installer-1.yaml",
					Path: "installer-1.yaml",
					Content: `apiVersion: kurl.sh/v1beta1
kind: Installer
`,
				},
				{
					Name: "installer-2.yaml",
					Path: "installer-2.yaml",
					Content: `apiVersion: cluster.kurl.sh/v1beta1
kind: Installer
`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "config-spec",
					Type:    "warn",
					Message: "Missing config spec",
				},
				{
					Rule:    "application-spec",
					Type:    "warn",
					Message: "Missing application spec",
				},
				{
					Rule:    "deprecated-kubernetes-installer-version",
					Path:    "installer-1.yaml",
					Type:    "warn",
					Message: "API version 'kurl.sh/v1beta1' is deprecated. Use 'cluster.kurl.sh/v1beta1' instead.",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "collector-1.yaml",
					Type:    "error",
					Message: "A release can only include one 'Collector' resource, but another 'Collector' resource was found in collector-2.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "collector-2.yaml",
					Type:    "error",
					Message: "A release can only include one 'Collector' resource, but another 'Collector' resource was found in collector-1.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "analyzer-1.yaml",
					Type:    "error",
					Message: "A release can only include one 'Analyzer' resource, but another 'Analyzer' resource was found in analyzer-2.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "analyzer-2.yaml",
					Type:    "error",
					Message: "A release can only include one 'Analyzer' resource, but another 'Analyzer' resource was found in analyzer-1.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "support-bundle-1.yaml",
					Type:    "error",
					Message: "A release can only include one 'SupportBundle' resource, but another 'SupportBundle' resource was found in support-bundle-2.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "support-bundle-2.yaml",
					Type:    "error",
					Message: "A release can only include one 'SupportBundle' resource, but another 'SupportBundle' resource was found in support-bundle-1.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "redactor-1.yaml",
					Type:    "error",
					Message: "A release can only include one 'Redactor' resource, but another 'Redactor' resource was found in redactor-2.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "redactor-2.yaml",
					Type:    "error",
					Message: "A release can only include one 'Redactor' resource, but another 'Redactor' resource was found in redactor-1.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "preflight-1.yaml",
					Type:    "error",
					Message: "A release can only include one 'Preflight' resource, but another 'Preflight' resource was found in preflight-2.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "preflight-2.yaml",
					Type:    "error",
					Message: "A release can only include one 'Preflight' resource, but another 'Preflight' resource was found in preflight-1.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "installer-1.yaml",
					Type:    "error",
					Message: "A release can only include one 'Installer' resource, but another 'Installer' resource was found in installer-2.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
				{
					Rule:    "duplicate-kots-kind",
					Path:    "installer-2.yaml",
					Type:    "error",
					Message: "A release can only include one 'Installer' resource, but another 'Installer' resource was found in installer-1.yaml",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 1,
							},
						},
					},
				},
			},
		},
		{
			name: "validate missing icon, status informer, config-spec, preflight-spec, troubleshoot-spec",
			specFiles: SpecFiles{
				{
					Name: "kots-app.yaml",
					Path: "kots-app.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app
  spec:
    title: App Name`,
				},
				{
					Name: "app-preflight.yaml",
					Path: "app-preflight.yaml",
					Content: `apiVersion: troubleshoot.sh/v1beta2
kind: Preflight`,
				},
				{
					Name: "app-supportbundle.yaml",
					Path: "app-supportbundle.yaml",
					Content: `apiVersion: troubleshoot.sh/v1beta2
kind: SupportBundle`,
				},
				{
					Name: "app-config.yaml",
					Path: "app-config.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Config`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "application-icon",
					Type:    "warn",
					Message: "Missing application icon",
					Path:    "kots-app.yaml",
				},
				{
					Rule:    "application-statusInformers",
					Type:    "warn",
					Message: "Missing application statusInformers",
					Path:    "kots-app.yaml",
				},
			},
		}, {
			name: "config-option-invalid-regex-validator",
			specFiles: SpecFiles{
				validKotsAppSpec,
				validPreflightSpec,
				validSupportBundleSpec,
				{
					Name: "app-config.yaml",
					Path: "app-config.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Config
spec:
  groups:
    - name: test
      title: Test
      items:
      - name: test
        title: Test
        type: text
        validation:
          regex:
            pattern: abc[`},
			},
			expect: []LintExpression{
				{
					Rule:    "config-option-invalid-regex-validator",
					Type:    "error",
					Path:    "app-config.yaml",
					Message: "Config option regex validator pattern \"abc[\" is invalid",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 13,
							},
						},
					},
				},
			},
		}, {
			name: "config-option-regex-validator-invalid-type",
			specFiles: SpecFiles{
				validKotsAppSpec,
				validPreflightSpec,
				validSupportBundleSpec,
				{
					Name: "app-config.yaml",
					Path: "app-config.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Config
spec:
  groups:
    - name: test
      title: Test
      items:
      - name: test
        title: Test
        type: bool
        validation:
          regex:
            pattern: .*`},
			},
			expect: []LintExpression{
				{
					Rule:    "config-option-regex-validator-invalid-type",
					Type:    "error",
					Path:    "app-config.yaml",
					Message: "Config option type should be one of [text|textarea|password|file] with regex validator",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 10,
							},
						},
					},
				},
			},
		}, {
			name: "valid regex pattern",
			specFiles: SpecFiles{
				validKotsAppSpec,
				validPreflightSpec,
				validSupportBundleSpec,
				validRegexdConfigSpec,
			},
			expect: []LintExpression{},
		},
	}

	InitOPALinting()

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
  kustomizeVersion: "3.5.4"
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
  kustomizeVersion: "latest"
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
								Line: 10,
							},
						},
					},
				},
				{
					Rule:    "nonexistent-status-informer-object",
					Type:    "warn",
					Path:    "test.yaml",
					Message: "Status informer points to a nonexistent kubernetes object. If this is a Helm resource, this warning can be ignored.",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 11,
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
					Message: "Status informer points to a nonexistent kubernetes object. If this is a Helm resource, this warning can be ignored.",
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
					Message: "Status informer points to a nonexistent kubernetes object. If this is a Helm resource, this warning can be ignored.",
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
		{
			name: "unsupported kustomize version",
			specFiles: SpecFiles{
				{
					Name: "kots-app.yaml",
					Path: "kots-app.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-slug
spec:
  kustomizeVersion: "2.0.0"`,
				},
			},
			expect: []LintExpression{
				{
					Rule:    "kustomize-version",
					Type:    "warn",
					Path:    "kots-app.yaml",
					Message: "Unsupported kustomize version, 3.5.4 will be used instead",
					Positions: []LintExpressionItemPosition{
						{
							Start: LintExpressionItemLinePosition{
								Line: 6,
							},
						},
					},
				},
			},
		},
		{
			name: "ignore nonexistent-status-informer-object rule",
			specFiles: SpecFiles{
				{
					Name: "app.yaml",
					Path: "app.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: app-slug
spec:
  statusInformers:
    - service/example-nginx`,
				},
				{
					Name: "lint-config.yaml",
					Path: "lint-config.yaml",
					Content: `apiVersion: kots.io/v1beta1
kind: LintConfig
metadata:
  name: lint-config
spec:
  rules:
    - name: nonexistent-status-informer-object
      level: "off"`,
				},
			},
			expect: []LintExpression{},
		},
	}

	InitOPALinting()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			separatedSpecFiles, err := test.specFiles.separate()
			require.NoError(t, err)

			renderedFiles, err := separatedSpecFiles.render()
			require.NoError(t, err)

			actual, err := lintWithOPARendered(renderedFiles, test.specFiles)
			require.NoError(t, err)
			assert.ElementsMatch(t, actual, test.expect)
		})
	}
}

func Test_lintKurlInstaller(t *testing.T) {
	tests := []struct {
		name      string
		specFiles SpecFiles
		expect    []LintExpression
	}{
		{
			name: "deployment file",
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
			specFiles: SpecFiles{
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
			specFiles: SpecFiles{
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
			expect: []LintExpression{
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
			specFiles: SpecFiles{
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
			expect: []LintExpression{
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
			specFiles: SpecFiles{
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
			expect: []LintExpression{
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := lintKurlInstaller(linter, test.specFiles)
			require.NoError(t, err)
			assert.ElementsMatch(t, actual, test.expect)
		})
	}
}
