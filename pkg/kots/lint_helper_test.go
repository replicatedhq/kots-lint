package kots

var (
	validKotsAppSpec = SpecFile{
		Name: "kots-app.yaml",
		Path: "kots-app.yaml",
		Content: `apiVersion: kots.io/v1beta1
kind: Application
spec:
  icon: https://github.com/cncf/artwork/blob/master/projects/kubernetes/icon/color/kubernetes-icon-color.png
  statusInformers:
    - deployment/example-nginx`,
	}

	validPreflightSpec = SpecFile{
		Name: "app-preflight.yaml",
		Path: "app-preflight.yaml",
		Content: `apiVersion: troubleshoot.sh/v1beta2
kind: Preflight`,
	}

	validSupportBundleSpec = SpecFile{
		Name: "app-supportbundle.yaml",
		Path: "app-supportbundle.yaml",
		Content: `apiVersion: troubleshoot.sh/v1beta2
kind: SupportBundle`,
	}

	validConfigSpec = SpecFile{
		Name: "app-config.yaml",
		Path: "app-config.yaml",
		Content: `apiVersion: kots.io/v1beta1
kind: Config`,
	}

	validExampleNginxDeploymentSpecFile = SpecFile{
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
	}

	validRegexValidationConfigSpec = SpecFile{
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
        pattern: .*`,
	}
)
