# EC v3 Lint Support Design

Date: 2026-04-17

## Overview

Three targeted fixes to make kots-lint support Embedded Cluster (EC) v3 releases. The core issue is that the linter was written before EC v3 existed and fails in three distinct ways when v3 features are present.

## Fix 1 — EC v3 Version Validation Skip

**File:** `pkg/ec/lint.go`

**Problem:** `lintVersion` calls `checkIfECVersionExists` for all EC config versions. EC v3 versions (e.g. `3.0.0-alpha-31+k8s-1.34`) are not present in the `replicatedhq/embedded-cluster` GitHub releases API, causing a spurious "Embedded Cluster version not found" error.

**Fix:** Add a private `isECV3Version(version string) bool` helper that returns true when `version` has prefix `"3."` or `"v3."` (mirrors the OPA rule already in `kots-spec-opa-nonrendered.rego`). In `lintVersion`, call this before `checkIfECVersionExists`; if it returns true, skip the GitHub API check and produce no lint expression. The `apiVersion`/`kind` detection and the `ec-version-required` check (missing version field) are unchanged.

## Fix 2 — Skip YAML Validation for v1beta3 Preflight Files with Helm Templating

**File:** `pkg/kots/lint.go`

**Problem:** `lintIsValidYAML` runs before template rendering. A `troubleshoot.sh/v1beta3` Preflight file that contains Helm template syntax (`{{- if .Values.foo }}`) fails YAML parsing with "could not find expected ':'", causing linting to abort early.

**Fix:** At the top of `lintFileHasValidYAML`, before attempting any YAML decode, check:
```
strings.Contains(file.Content, "apiVersion: troubleshoot.sh/v1beta3") &&
strings.Contains(file.Content, "kind: Preflight")
```
If both match, return an empty slice immediately. String matching is used because the file may be unparseable YAML; `apiVersion` and `kind` appear at the top level and are unaffected by Helm blocks deeper in the spec.

## Fix 3 — Suppress Specific Undefined Function Errors for EC v3 Releases

**File:** `pkg/kots/lint.go`

**Problem:** `ReplicatedImageName` and `ReplicatedImageRegistry` are EC v3 template functions not registered in the kots template builder. When a cluster-config.yaml uses them, `RenderContent` returns a "function not defined" error, blocking linting.

**Fix:** In `lintRenderContent`, before processing render errors:
1. Determine whether the release is an EC v3 release by scanning `yamlFiles` for any file where `apiVersion == "embeddedcluster.replicated.com/v1beta1"`, `kind == "Config"`, and `spec.version` passes `isECV3Version`. Extract `isECV3Version` to a shared location (e.g. `pkg/ec/version.go`) so both `pkg/ec/lint.go` and `pkg/kots/lint.go` can use it.
2. If the release is EC v3 AND the render error message is exactly `function "ReplicatedImageName" not defined` or `function "ReplicatedImageRegistry" not defined`, omit the lint expression entirely.
3. All other render errors are emitted as before. Non-EC-v3 releases are unaffected.

## Testing

Each fix gets a dedicated test case:

- **Fix 1:** Add test to `pkg/ec/lint_test.go` with an EC v3 version (e.g. `"3.0.0+k8s-1.34"`) pointing at the mock HTTP server; assert no lint expression is produced without any GitHub API call.
- **Fix 2:** Add test to `pkg/kots/lint_test.go` with a `troubleshoot.sh/v1beta3` Preflight file containing a `{{- if ... }}` block; assert no `invalid-yaml` error.
- **Fix 3:** Add test to `pkg/kots/lint_test.go` with an EC v3 cluster-config that references `ReplicatedImageName` and `ReplicatedImageRegistry`; assert no lint expressions. Also add a counter-test showing non-EC-v3 releases still get errors for these functions.

## Out of Scope

- Registering `ReplicatedImageName`/`ReplicatedImageRegistry` as real stub functions that return values.
- Validating EC v3 versions against any alternative release source.
- Handling Helm template syntax in non-Preflight or non-v1beta3 files.
