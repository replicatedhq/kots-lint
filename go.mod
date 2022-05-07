module github.com/replicatedhq/kots-lint

go 1.16

require (
	github.com/gin-gonic/gin v1.7.1
	github.com/gotestyourself/gotestyourself v2.2.0+incompatible // indirect
	github.com/instrumenta/kubeval v0.0.0-20190918223246-8d013ec9fc56
	github.com/mitchellh/mapstructure v1.4.2
	github.com/open-policy-agent/opa v0.34.2
	github.com/pkg/errors v0.9.1
	github.com/replicatedcom/saaskit v0.0.0-20220404234806-42d82ab45c31
	github.com/replicatedhq/kots v1.69.2-0.20220427202613-e7b5ac2fe43e
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.0
	github.com/tommy351/gin-cors v0.0.0-20150617141853-dc91dec6313a
	github.com/xeipuuv/gojsonschema v1.2.1-0.20200118195451-b537c054d4b4 // indirect
	go.undefinedlabs.com/scopeagent v0.1.15
	gopkg.in/yaml.v2 v2.4.0
	helm.sh/helm/v3 v3.7.1
	k8s.io/client-go v0.23.1
)

replace (
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20170817175659-5f6282db7d65
	github.com/docker/docker => github.com/docker/docker v0.0.0-20180522102801-da99009bbb11
	github.com/longhorn/longhorn-manager => github.com/replicatedhq/longhorn-manager v1.1.2-0.20210622201804-05b01947b99d
	github.com/nicksnyder/go-i18n => github.com/nicksnyder/go-i18n v1.10.1
	github.com/vmware-tanzu/velero => github.com/laverya/velero v1.4.1-0.20200618194205-ba7f18d4a7d8 // only until https://github.com/vmware-tanzu/velero/pull/2651 is merged
	k8s.io/api => k8s.io/api v0.23.1
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.23.1
	k8s.io/apimachinery => k8s.io/apimachinery v0.23.1
	k8s.io/apiserver => k8s.io/apiserver v0.23.1
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.23.1
	k8s.io/client-go => k8s.io/client-go v0.23.1
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.23.1
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.23.1
	k8s.io/code-generator => k8s.io/code-generator v0.23.1
	k8s.io/component-base => k8s.io/component-base v0.23.1
	k8s.io/component-helpers => k8s.io/component-helpers v0.23.1
	k8s.io/controller-manager => k8s.io/controller-manager v0.23.1
	k8s.io/cri-api => k8s.io/cri-api v0.23.1
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.23.1
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.23.1
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.23.1
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.23.1
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.23.1
	k8s.io/kubectl => k8s.io/kubectl v0.23.1
	k8s.io/kubelet => k8s.io/kubelet v0.23.1
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.23.1
	k8s.io/metrics => k8s.io/metrics v0.23.1
	k8s.io/mount-utils => k8s.io/mount-utils v0.23.1
	k8s.io/node-api => k8s.io/node-api v0.23.1
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.23.1
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.23.1
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.23.1
	k8s.io/sample-controller => k8s.io/sample-controller v0.23.1
)
