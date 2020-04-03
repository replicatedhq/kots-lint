## kots-lint is a service used for linting Replicated [KOTS](https://kots.io) yaml files.

### Linting with this service includes:

 - Detecting YAML syntax errors
 - Validating with Kubeval
 - Utilizing [OPA](https://github.com/open-policy-agent/opa) to lint for best practices and some special errors and warnings

## Development

The project can be run locally with
```shell
$ skaffold dev
```

Once skaffold runs successfully, the service can be reached at http://localhost:30082/v1/lint

## Run tests

Tests can be run manually with
```shell
$ make test
```
