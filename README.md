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

The project can be run locally with
```shell
$ skaffold dev
```

Once skaffold runs successfully, the service can be reached at http://localhost:30082/v1/lint

### Adding new Kubernetes Versions

When adding new Kubernetes versions, the following non-standard schemas (anneed to be added to any new version folder:
* airgap-kots-v1beta1.json
* application-app-v1beta1.json
* application-kgrid-v1alpha1.json
* application-kots-v1beta1.json
* analyzer-troubleshoot-v1beta1.json
* analyzer-troubleshoot-v1beta2.json
* backup-velero-v1.json
* collector-troubleshoot-v1beta1.json
* collector-troubleshoot-v1beta2.json
* config-kots-v1beta1.json
* configvalues-kots-v1beta1.json
* grid-kgrid-v1alpha1.json
* identity-kots-v1beta1.json
* helmchart-kots-v1beta1.json
* installation-kots-v1beta1.json
* license-kots-v1beta1.json
* preflight-troubleshoot-v1beta1.json
* preflight-troubleshoot-v1beta1.json
* preflight-troubleshoot-v1beta2.json
* redactor-troubleshoot-v1beta1.json
* redactor-troubleshoot-v1beta2.json
* supportbundle-troubleshoot-v1beta1.json
* supportbundle-troubleshoot-v1beta2.json
* version-kgrid-v1alpha1.json

You can diff the schema folders with previous versions to verify the above list.

## Run tests

Tests can be run manually with
```shell
$ make test
```
