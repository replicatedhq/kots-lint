## kots-lint is a service used for linting Replicated [KOTS](https://kots.io) yaml files.

### Linting with this service includes:

 - Detecting YAML syntax errors
 - Validating with [Kubeval](https://github.com/instrumenta/kubeval)
 - Utilizing [OPA](https://github.com/open-policy-agent/opa) to lint for best practices and some special errors and warnings

## Using the production API

```shell
$ tar cvf - path/to/folder | curl -XPOST --data-binary @- https://lint.replicated.com/v1/lint
```
To lint our example
```shell
$ tar cvf - example/files-to-lint | curl -XPOST --data-binary @- https://lint.replicated.com/v1/lint
```

## Development

Development for the applications in this project is done through [Okteto](https://replicated.okteto.dev).

## Setup

1. Install the Okteto CLI (`brew install okteto`)
2. Setup Okteto CLI (`okteto context create https://replicated.okteto.dev`)
3. Setup Okteto context in kubectl (`okteto context update-kubeconfig`)
4. Deploy your current branch. (from the Vandoor root directory: `okteto pipeline deploy`)

The project can also be run with Skaffold for local development and testing.
```shell
$ skaffold dev
```

Once skaffold runs successfully, the service can be reached at http://localhost:30082/v1/lint

## Run tests

Tests can be run manually with
```shell
$ make test
```
