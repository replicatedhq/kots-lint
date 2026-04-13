# EC v3 Preflight apiVersion Lint Rule

**Date:** 2026-04-13

## Summary

Add a lint error when an Embedded Cluster v3.x version is specified and a troubleshoot Preflight spec in the same spec files does not use `apiVersion: troubleshoot.sh/v1beta3`.

## Changes

### `pkg/ec/lint.go`

**`Lint(specFiles domain.SpecFiles) ([]domain.LintExpression, error)`** — new public entry point, replaces `LintEmbeddedClusterVersion`. Responsibilities:
1. Separate multi-doc YAML files.
2. Parse the EC version string from any `embeddedcluster.replicated.com/v1beta1` Config doc.
3. Call `lintVersion(specFiles)` unconditionally and collect results.
4. If the parsed EC version starts with `v3.`, call `lintV3Preflight(specFiles)` and collect results.
5. Return all aggregated lint expressions.

**`lintVersion(specFiles domain.SpecFiles) ([]domain.LintExpression, error)`** — rename of existing `LintEmbeddedClusterVersion`, made private. Logic unchanged: checks version is present, exists on GitHub, and is not a pre-release.

**`lintV3Preflight(specFiles domain.SpecFiles) ([]domain.LintExpression, error)`** — new private function. Iterates separated spec files; for each doc where `kind == "Preflight"` and `apiVersion != "troubleshoot.sh/v1beta3"`, emits:
```go
domain.LintExpression{
    Rule:    "ec-v3-preflight-api-version",
    Type:    "error",
    Path:    spec.Path,
    Message: "Preflight spec must use apiVersion troubleshoot.sh/v1beta3 with Embedded Cluster v3",
}
```

### `pkg/kots/lint.go`

Rename the call from `ec.LintEmbeddedClusterVersion(yamlFiles)` → `ec.Lint(yamlFiles)`.

## Version Detection

EC version is parsed from `spec.version` in `embeddedcluster.replicated.com/v1beta1` Config docs. A version is considered v3.x if it starts with `"v3."`.

## Error Rule Name

`ec-v3-preflight-api-version`

## Tests

- `pkg/ec/lint_test.go`: add cases to `Test_LintEmbeddedClusterVersion` (or rename test) covering:
  - EC v3.x + Preflight with `troubleshoot.sh/v1beta3` → no error
  - EC v3.x + Preflight with `troubleshoot.sh/v1beta2` → error on preflight path
  - EC v3.x + no Preflight → no error
  - EC v2.x + Preflight with `troubleshoot.sh/v1beta2` → no error
- Existing tests remain passing; only the call site in `lint.go` changes.
