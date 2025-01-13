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

### Debugging Rego files

Rego supports the use of `print()` function. To enable printing to stderr, add `EnablePrintStatements` and `PrintHook` to initialization code. For example:

```
	buildersQuery, err := rego.New(
		rego.Query("data.kots.spec.builders.lint"),
		rego.Module("builders-opa.rego", string(buildersRegoContent)),

		// The lines below allow using print() in the rego code to print to stderr
		rego.EnablePrintStatements(true),
		rego.PrintHook(topdown.NewPrintHook(os.Stderr)),
	).PrepareForEval(ctx)
```

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

## Updating specs

Generated specs can be copied from the source project directly.
File names should match, but since projects will change independently, care should be taken when copying.

### Troubleshoot

```
cp <troubleshoot root>/schemas/*.json <kots-lint root>/kubernetes_json_schema/schema/troubleshoot/
```

### KOTS

```
cp <kots root>/kotskinds/schemas/*.json <kots-lint root>/kubernetes_json_schema/schema/kots/
```

### Kubernetes

```
git clone git@github.com:yannh/kubernetes-json-schema.git
cp <kubernetes-json-schema>/<desired-version>-standalone-strict/*.json <kots-lint root>/kubernetes_json_schema/schema/<desired-version>-standalone-strict/
rm -rf <kots-lint root>/kubernetes_json_schema/schema/<previous-version>-standalone-strict/
```
