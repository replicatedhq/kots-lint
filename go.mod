module github.com/replicatedhq/kots-lint

go 1.14

require (
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/gin-gonic/gin v1.6.1
	github.com/instrumenta/kubeval v0.0.0-20190918223246-8d013ec9fc56
	github.com/johntdyer/slack-go v0.0.0-20180213144715-95fac1160b22 // indirect
	github.com/mitchellh/mapstructure v1.2.2
	github.com/newrelic/go-agent v2.13.0+incompatible
	github.com/open-policy-agent/opa v0.18.0
	github.com/pkg/errors v0.9.1
	github.com/replicatedcom/saaskit v0.0.0-20180719175845-249c0d6c71b2
	github.com/replicatedhq/kots v1.17.2
	github.com/stretchr/testify v1.5.1
	github.com/tommy351/gin-cors v0.0.0-20150617141853-dc91dec6313a
	github.com/xeipuuv/gojsonschema v1.2.1-0.20200118195451-b537c054d4b4 // indirect
	go.undefinedlabs.com/scopeagent v0.1.12
	gopkg.in/yaml.v2 v2.3.0
	helm.sh/helm/v3 v3.2.4
	k8s.io/client-go v0.18.3
)

replace github.com/nicksnyder/go-i18n => github.com/nicksnyder/go-i18n v1.10.1

replace github.com/docker/docker => github.com/docker/docker v0.0.0-20180522102801-da99009bbb11
