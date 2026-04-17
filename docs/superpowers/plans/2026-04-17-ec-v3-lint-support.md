# EC v3 Lint Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix three linting failures that occur when a release uses Embedded Cluster v3: spurious version-not-found errors, YAML parse failures from Helm template syntax in v1beta3 Preflight files, and "function not defined" errors for EC v3-only template functions.

**Architecture:** A new `pkg/ec/version.go` exports `IsECV3Version` so both `pkg/ec` and `pkg/kots` can share the v3 detection logic without circular imports. Each fix is a small, targeted guard added to an existing function; no new abstractions are introduced.

**Tech Stack:** Go, `gopkg.in/yaml.v2`, `github.com/stretchr/testify`

---

## File Map

| File | Action | Purpose |
|---|---|---|
| `pkg/ec/version.go` | Create | Export `IsECV3Version` helper |
| `pkg/ec/version_test.go` | Create | Unit tests for `IsECV3Version` |
| `pkg/ec/lint.go` | Modify | Skip GitHub API check for v3 versions |
| `pkg/ec/lint_test.go` | Modify | Add EC v3 test case to `Test_Lint` |
| `pkg/kots/lint.go` | Modify | Skip YAML validation for v1beta3 Preflight; suppress EC v3 function errors |
| `pkg/kots/lint_test.go` | Modify | Add tests for the two kots fixes |

---

## Task 1: Extract IsECV3Version to pkg/ec/version.go

**Files:**
- Create: `pkg/ec/version.go`
- Create: `pkg/ec/version_test.go`

- [ ] **Step 1: Write the failing test**

Create `pkg/ec/version_test.go`:

```go
package ec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_IsECV3Version(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"3.0.0+k8s-1.34", true},
		{"v3.0.0+k8s-1.34", true},
		{"3.1.2-beta+k8s-1.35", true},
		{"v3.9.9", true},
		{"2.9.9+k8s-1.29", false},
		{"v2.0.0+k8s-1.29", false},
		{"v1.2.2+k8s-1.29", false},
		{"30.0.0", false},
		{"v30.0.0", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			assert.Equal(t, tt.want, IsECV3Version(tt.version))
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/ethan/go/src/github.com/replicatedhq/kots-lint/.claude/worktrees/hazy-forging-rabin
go test ./pkg/ec/... -run Test_IsECV3Version -v
```

Expected: compile error — `IsECV3Version` undefined.

- [ ] **Step 3: Create pkg/ec/version.go**

```go
package ec

import "strings"

// IsECV3Version reports whether version represents an Embedded Cluster v3 release.
// It accepts versions with or without a leading "v" (e.g. "3.0.0+k8s-1.34" or "v3.0.0+k8s-1.34").
func IsECV3Version(version string) bool {
	return strings.HasPrefix(version, "3.") || strings.HasPrefix(version, "v3.")
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./pkg/ec/... -run Test_IsECV3Version -v
```

Expected: all 9 cases PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/ec/version.go pkg/ec/version_test.go
git commit -m "feat(ec): add IsECV3Version helper"
```

---

## Task 2: Skip version validation for EC v3 in pkg/ec/lint.go

**Files:**
- Modify: `pkg/ec/lint.go:59-83` (inside the `else` branch of `versionExists`)
- Modify: `pkg/ec/lint_test.go` (add EC v3 case to `Test_Lint`)

- [ ] **Step 1: Write the failing test**

In `pkg/ec/lint_test.go`, add this case inside the `tests` slice in `Test_Lint`, after the existing "valid version" case:

```go
{
    name: "ec v3 version skips github check",
    specFiles: domain.SpecFiles{
        {
            Path: "cluster-config.yaml",
            Content: `apiVersion: embeddedcluster.replicated.com/v1beta1
kind: Config
spec:
  version: "3.0.0+k8s-1.34"`,
        },
    },
    expect:    []domain.LintExpression{},
    apiResult: nil, // server returns 404; must not be reached
},
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./pkg/ec/... -run Test_Lint/ec_v3_version_skips_github_check -v
```

Expected: FAIL — the mock server returns 404, so the linter emits `non-existent-ec-version`.

- [ ] **Step 3: Add the v3 guard in lintVersion**

In `pkg/ec/lint.go`, inside the `else` branch (line ~59), add an early-return before calling `checkIfECVersionExists`:

Current code (lines 59–83):
```go
		} else {
			// version is defined, check if it is valid.
			ecVersion, exists, err := checkIfECVersionExists(version)
```

Replace with:
```go
		} else {
			if IsECV3Version(version) {
				continue
			}
			// version is defined, check if it is valid.
			ecVersion, exists, err := checkIfECVersionExists(version)
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./pkg/ec/... -run Test_Lint -v
```

Expected: all cases PASS including the new EC v3 case.

- [ ] **Step 5: Commit**

```bash
git add pkg/ec/lint.go pkg/ec/lint_test.go
git commit -m "feat(ec): skip version validation for EC v3 releases"
```

---

## Task 3: Skip YAML validation for v1beta3 Preflight files with Helm templates

**Files:**
- Modify: `pkg/kots/lint.go:803` (`lintFileHasValidYAML`)
- Modify: `pkg/kots/lint_test.go` (add case to `Test_lintFileHasValidYAML`)

- [ ] **Step 1: Write the failing test**

In `pkg/kots/lint_test.go`, add this case inside the `tests` slice in `Test_lintFileHasValidYAML`:

```go
{
    name: "v1beta3 preflight with helm template syntax is skipped",
    specFile: domain.SpecFile{
        Path: "preflight.yaml",
        Content: `apiVersion: troubleshoot.sh/v1beta3
kind: Preflight
metadata:
  name: vendor-app-preflight
spec:
  analyzers:
    {{- if .Values.client.enabled }}
    - nodeResources:
        checkName: Cluster Memory
        outcomes:
          - fail:
              when: "sum(memoryCapacity) < 2Gi"
              message: The cluster requires at least 2Gi of memory.
          - pass:
              message: The cluster has sufficient memory.
    {{- end }}`,
    },
    expect: []domain.LintExpression{},
},
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./pkg/kots/... -run Test_lintFileHasValidYAML/v1beta3_preflight_with_helm_template_syntax_is_skipped -v
```

Expected: FAIL — emits `invalid-yaml` error at line 7.

- [ ] **Step 3: Add the guard to lintFileHasValidYAML**

In `pkg/kots/lint.go`, at the top of `lintFileHasValidYAML` (before `lintExpressions := []domain.LintExpression{}`), add:

```go
func lintFileHasValidYAML(file domain.SpecFile) []domain.LintExpression {
	// v1beta3 Preflight files may contain Helm template syntax which is not valid YAML.
	// They will be rendered later; skip static YAML validation for them.
	if strings.Contains(file.Content, "apiVersion: troubleshoot.sh/v1beta3") &&
		strings.Contains(file.Content, "kind: Preflight") {
		return []domain.LintExpression{}
	}

	lintExpressions := []domain.LintExpression{}
	// ... rest of function unchanged
```

Note: `strings` is already imported in `pkg/kots/lint.go`.

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./pkg/kots/... -run Test_lintFileHasValidYAML -v
```

Expected: all cases PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/kots/lint.go pkg/kots/lint_test.go
git commit -m "feat(kots): skip YAML validation for v1beta3 Preflight files with Helm templates"
```

---

## Task 4: Suppress EC v3 template function errors in lintRenderContent

**Files:**
- Modify: `pkg/kots/lint.go` — add `isReleaseECV3` and `isECV3IgnoredFunctionError` helpers; update `lintRenderContent`
- Modify: `pkg/kots/lint_test.go` — add cases to `Test_lintRenderContent`

- [ ] **Step 1: Write the failing tests**

In `pkg/kots/lint_test.go`, add these two cases inside the `tests` slice in `Test_lintRenderContent`.

The test harness (line ~2144) does:
```go
actual, renderedFiles, err := lintRenderContent(test.specFiles)
assert.ElementsMatch(t, actual, test.expect)
assert.ElementsMatch(t, renderedFiles, test.renderedFiles)
```

For both cases the file fails to render so `renderedFiles` is empty. For the non-EC v3 case use a flat 5-line file so the line position is predictable (line 5, match `  name: '{{repl ReplicatedImageName "myapp" }}'`):

```go
{
    name: "ec v3 release ignores ReplicatedImageName and ReplicatedImageRegistry errors",
    specFiles: domain.SpecFiles{
        {
            Name: "cluster-config.yaml",
            Path: "cluster-config.yaml",
            Content: `apiVersion: embeddedcluster.replicated.com/v1beta1
kind: Config
spec:
  version: "3.0.0+k8s-1.34"
  extensions:
    helm:
      charts:
        - chartname: myapp
          values: |
            image: '{{repl ReplicatedImageName "myapp" }}'
            registry: '{{repl ReplicatedImageRegistry "myapp" }}'`,
        },
    },
    renderedFiles: domain.SpecFiles{},
    expect:        []domain.LintExpression{},
},
{
    name: "non-ec-v3 release still errors on ReplicatedImageName",
    specFiles: domain.SpecFiles{
        {
            Name: "ec-config.yaml",
            Path: "ec-config.yaml",
            Content: `apiVersion: embeddedcluster.replicated.com/v1beta1
kind: Config
spec:
  version: "2.0.0+k8s-1.29"
  name: '{{repl ReplicatedImageName "myapp" }}'`,
        },
    },
    renderedFiles: domain.SpecFiles{},
    expect: []domain.LintExpression{
        {
            Rule:    "unable-to-render",
            Type:    "error",
            Path:    "ec-config.yaml",
            Message: `function "ReplicatedImageName" not defined`,
            Positions: []domain.LintExpressionItemPosition{
                {
                    Start: domain.LintExpressionItemLinePosition{
                        Line: 5,
                    },
                },
            },
        },
    },
},
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./pkg/kots/... -run "Test_lintRenderContent/ec_v3_release_ignores|Test_lintRenderContent/non-ec-v3_release_still_errors" -v
```

Expected: both FAIL — EC v3 case emits an `unable-to-render` error; non-v3 case also emits an error (same behavior, both wrong for now).

- [ ] **Step 3: Add helpers and update lintRenderContent**

In `pkg/kots/lint.go`, add these two private helpers anywhere before `lintRenderContent`:

```go
// isReleaseECV3 returns true if specFiles contains an Embedded Cluster v3 Config resource.
func isReleaseECV3(specFiles domain.SpecFiles) bool {
	for _, file := range specFiles {
		doc := map[string]interface{}{}
		if err := yaml.Unmarshal([]byte(file.Content), &doc); err != nil {
			continue
		}
		if doc["apiVersion"] != "embeddedcluster.replicated.com/v1beta1" || doc["kind"] != "Config" {
			continue
		}
		spec, ok := doc["spec"].(map[interface{}]interface{})
		if !ok {
			continue
		}
		version, ok := spec["version"].(string)
		if !ok {
			continue
		}
		if ec.IsECV3Version(version) {
			return true
		}
	}
	return false
}

// isECV3IgnoredFunctionError returns true when err is a "not defined" error for a
// template function that only exists in EC v3 runtime contexts.
func isECV3IgnoredFunctionError(err string) bool {
	for _, fn := range []string{"ReplicatedImageName", "ReplicatedImageRegistry"} {
		if strings.Contains(err, fmt.Sprintf(`function "%s" not defined`, fn)) {
			return true
		}
	}
	return false
}
```

Then, in `lintRenderContent`, add one line after the opening of the function body (after `separatedSpecFiles` is obtained but before the render loop) and update the `RenderTemplateError` handling block:

After line `separatedSpecFiles, err := specFiles.Separate()` and its error check, add:
```go
releaseIsECV3 := isReleaseECV3(specFiles)
```

Then inside the render loop, in the `if renderErr, ok := errors.Cause(err).(domain.RenderTemplateError); ok {` block, add a guard at the very top of that block:

```go
		if renderErr, ok := errors.Cause(err).(domain.RenderTemplateError); ok {
			if releaseIsECV3 && isECV3IgnoredFunctionError(renderErr.Error()) {
				continue
			}
			lintExpression := domain.LintExpression{
				// ... rest of existing code unchanged
```

The full updated block looks like:

```go
		// check if the error is coming from kots RenderTemplate function
		if renderErr, ok := errors.Cause(err).(domain.RenderTemplateError); ok {
			if releaseIsECV3 && isECV3IgnoredFunctionError(renderErr.Error()) {
				continue
			}
			lintExpression := domain.LintExpression{
				Rule:    "unable-to-render",
				Type:    "error",
				Path:    file.Path,
				Message: renderErr.Error(),
			}

			if renderErr.Match() != "" {
				// we need to get the line number for the original file content not the separated document
				foundSpecFile, err := specFiles.GetFile(file.Path)
				if err != nil {
					lintExpressions = append(lintExpressions, lintExpression)
					continue
				}

				line, err := util.GetLineNumberFromMatch(foundSpecFile.Content, renderErr.Match(), file.DocIndex)
				if err != nil || line == -1 {
					lintExpressions = append(lintExpressions, lintExpression)
					continue
				}
				lintExpression.Positions = []domain.LintExpressionItemPosition{
					{
						Start: domain.LintExpressionItemLinePosition{
							Line: line,
						},
					},
				}
			}

			lintExpressions = append(lintExpressions, lintExpression)
			continue
		}
```

Note: `fmt` is already imported in `pkg/kots/lint.go`.

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./pkg/kots/... -run Test_lintRenderContent -v
```

Expected: all cases PASS.

- [ ] **Step 5: Run the full test suite**

```bash
go test ./... -v 2>&1 | tail -40
```

Expected: all tests PASS with no regressions.

- [ ] **Step 6: Commit**

```bash
git add pkg/kots/lint.go pkg/kots/lint_test.go
git commit -m "feat(kots): suppress EC v3 template function errors for ReplicatedImageName and ReplicatedImageRegistry"
```
