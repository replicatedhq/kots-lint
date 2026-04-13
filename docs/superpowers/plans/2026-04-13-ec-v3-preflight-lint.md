# EC v3 Preflight apiVersion Lint Rule Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Emit a lint error when an Embedded Cluster v3.x config is present and any Preflight spec in the same file set does not use `apiVersion: troubleshoot.sh/v1beta3`.

**Architecture:** Introduce a public `Lint` entry point in `pkg/ec/lint.go` that parses the EC version, delegates to the existing (now private) `lintVersion` for version validation, and conditionally calls a new `lintV3Preflight` when the version is v3.x. The call site in `pkg/kots/lint.go` is updated to use the new public name.

**Tech Stack:** Go, `gopkg.in/yaml.v2`, `github.com/replicatedhq/kots-lint/pkg/domain`, `strings` stdlib package.

---

## File Map

- **Modify:** `pkg/ec/lint.go` — add `Lint`, rename `LintEmbeddedClusterVersion` → `lintVersion`, add `lintV3Preflight`
- **Modify:** `pkg/ec/lint_test.go` — rename test, update call to `Lint`, add 4 new test cases
- **Modify:** `pkg/kots/lint.go:237` — update `ec.LintEmbeddedClusterVersion` → `ec.Lint`

---

### Task 1: Refactor `LintEmbeddedClusterVersion` into `Lint` + `lintVersion`

**Files:**
- Modify: `pkg/ec/lint.go`
- Modify: `pkg/ec/lint_test.go`
- Modify: `pkg/kots/lint.go`

- [ ] **Step 1: Rename `LintEmbeddedClusterVersion` → `lintVersion` in `pkg/ec/lint.go`**

Change the function signature from:
```go
func LintEmbeddedClusterVersion(specFiles domain.SpecFiles) ([]domain.LintExpression, error) {
```
to:
```go
func lintVersion(specFiles domain.SpecFiles) ([]domain.LintExpression, error) {
```
No other changes to the function body.

- [ ] **Step 2: Add `"strings"` to the import block in `pkg/ec/lint.go`**

The existing import block becomes:
```go
import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots-lint/pkg/domain"
	"gopkg.in/yaml.v2"
)
```

- [ ] **Step 3: Add the `Lint` function to `pkg/ec/lint.go` (just above `lintVersion`)**

```go
func Lint(specFiles domain.SpecFiles) ([]domain.LintExpression, error) {
	lintExpressions := []domain.LintExpression{}

	separatedSpecFiles, err := specFiles.Separate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to separate multi docs")
	}

	var ecVersion string
	for _, spec := range separatedSpecFiles {
		doc := map[string]interface{}{}
		if err := yaml.Unmarshal([]byte(spec.Content), &doc); err != nil {
			continue
		}
		if doc["apiVersion"] == "embeddedcluster.replicated.com/v1beta1" && doc["kind"] == "Config" {
			if specMap, ok := doc["spec"].(map[interface{}]interface{}); ok {
				if v, ok := specMap["version"].(string); ok {
					ecVersion = v
				}
			}
		}
	}

	versionExpressions, err := lintVersion(specFiles)
	if err != nil {
		return nil, err
	}
	lintExpressions = append(lintExpressions, versionExpressions...)

	if strings.HasPrefix(ecVersion, "v3.") {
		preflightExpressions, err := lintV3Preflight(specFiles)
		if err != nil {
			return nil, err
		}
		lintExpressions = append(lintExpressions, preflightExpressions...)
	}

	return lintExpressions, nil
}
```

- [ ] **Step 4: Add a stub `lintV3Preflight` so the package compiles**

Add this at the bottom of `pkg/ec/lint.go`:
```go
func lintV3Preflight(specFiles domain.SpecFiles) ([]domain.LintExpression, error) {
	return []domain.LintExpression{}, nil
}
```

- [ ] **Step 5: Update the call site in `pkg/kots/lint.go`**

Find line ~237:
```go
embeddedClusterLintExpressions, err := ec.LintEmbeddedClusterVersion(yamlFiles)
```
Change to:
```go
embeddedClusterLintExpressions, err := ec.Lint(yamlFiles)
```

- [ ] **Step 6: Update `pkg/ec/lint_test.go` — rename test function and update call**

Rename `Test_LintEmbeddedClusterVersion` → `Test_Lint` and change `LintEmbeddedClusterVersion` → `Lint` inside the test body:
```go
func Test_Lint(t *testing.T) {
    // ...existing test cases unchanged...
    
    for _, test := range tests {
        t.Run(test.name, func(t *testing.T) {
            // ...server setup unchanged...
            actual, err := Lint(test.specFiles)
            require.NoError(t, err)
            assert.ElementsMatch(t, actual, test.expect)
        })
    }
}
```

- [ ] **Step 7: Run tests to verify existing behaviour is preserved**

```
go test ./pkg/ec/... ./pkg/kots/...
```

Expected: all existing tests pass.

- [ ] **Step 8: Commit**

```bash
git add pkg/ec/lint.go pkg/ec/lint_test.go pkg/kots/lint.go
git commit -m "refactor(ec): introduce Lint entry point, make lintVersion private"
```

---

### Task 2: Add failing tests for `lintV3Preflight`

**Files:**
- Modify: `pkg/ec/lint_test.go`

- [ ] **Step 1: Add four new test cases to the `tests` slice in `Test_Lint`**

Add these after the existing `"pre-release version"` case:
```go
{
    name: "v3 preflight with correct v1beta3 apiVersion",
    specFiles: domain.SpecFiles{
        {
            Path: "ec-config.yaml",
            Content: `apiVersion: embeddedcluster.replicated.com/v1beta1
kind: Config
spec:
  version: "v3.0.0+k8s-1.29"`,
        },
        {
            Path: "preflight.yaml",
            Content: `apiVersion: troubleshoot.sh/v1beta3
kind: Preflight
metadata:
  name: preflight-sample
spec:
  analyzers: []`,
        },
    },
    expect:    []domain.LintExpression{},
    apiResult: []byte(`{}`),
},
{
    name: "v3 preflight with wrong v1beta2 apiVersion",
    specFiles: domain.SpecFiles{
        {
            Path: "ec-config.yaml",
            Content: `apiVersion: embeddedcluster.replicated.com/v1beta1
kind: Config
spec:
  version: "v3.0.0+k8s-1.29"`,
        },
        {
            Path: "preflight.yaml",
            Content: `apiVersion: troubleshoot.sh/v1beta2
kind: Preflight
metadata:
  name: preflight-sample
spec:
  analyzers: []`,
        },
    },
    expect: []domain.LintExpression{
        {
            Rule:    "ec-v3-preflight-api-version",
            Type:    "error",
            Path:    "preflight.yaml",
            Message: "Preflight spec must use apiVersion troubleshoot.sh/v1beta3 with Embedded Cluster v3",
        },
    },
    apiResult: []byte(`{}`),
},
{
    name: "v3 with no preflight",
    specFiles: domain.SpecFiles{
        {
            Path: "ec-config.yaml",
            Content: `apiVersion: embeddedcluster.replicated.com/v1beta1
kind: Config
spec:
  version: "v3.0.0+k8s-1.29"`,
        },
    },
    expect:    []domain.LintExpression{},
    apiResult: []byte(`{}`),
},
{
    name: "v2 preflight with v1beta2 apiVersion no error",
    specFiles: domain.SpecFiles{
        {
            Path: "ec-config.yaml",
            Content: `apiVersion: embeddedcluster.replicated.com/v1beta1
kind: Config
spec:
  version: "v2.0.0+k8s-1.29"`,
        },
        {
            Path: "preflight.yaml",
            Content: `apiVersion: troubleshoot.sh/v1beta2
kind: Preflight
metadata:
  name: preflight-sample
spec:
  analyzers: []`,
        },
    },
    expect:    []domain.LintExpression{},
    apiResult: []byte(`{}`),
},
```

- [ ] **Step 2: Run tests to verify the new cases fail**

```
go test ./pkg/ec/... -run Test_Lint -v
```

Expected: the `"v3 preflight with wrong v1beta2 apiVersion"` case fails with no lint expression returned (stub returns empty). The other three new cases pass.

---

### Task 3: Implement `lintV3Preflight`

**Files:**
- Modify: `pkg/ec/lint.go`

- [ ] **Step 1: Replace the stub `lintV3Preflight` with the real implementation**

```go
func lintV3Preflight(specFiles domain.SpecFiles) ([]domain.LintExpression, error) {
	lintExpressions := []domain.LintExpression{}

	separatedSpecFiles, err := specFiles.Separate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to separate multi docs")
	}

	for _, spec := range separatedSpecFiles {
		doc := map[string]interface{}{}
		if err := yaml.Unmarshal([]byte(spec.Content), &doc); err != nil {
			continue
		}
		if doc["kind"] == "Preflight" {
			apiVersion, _ := doc["apiVersion"].(string)
			if apiVersion != "troubleshoot.sh/v1beta3" {
				lintExpressions = append(lintExpressions, domain.LintExpression{
					Rule:    "ec-v3-preflight-api-version",
					Type:    "error",
					Path:    spec.Path,
					Message: "Preflight spec must use apiVersion troubleshoot.sh/v1beta3 with Embedded Cluster v3",
				})
			}
		}
	}

	return lintExpressions, nil
}
```

- [ ] **Step 2: Run all tests to verify everything passes**

```
go test ./pkg/ec/... ./pkg/kots/...
```

Expected: all tests pass.

- [ ] **Step 3: Commit**

```bash
git add pkg/ec/lint.go pkg/ec/lint_test.go
git commit -m "feat(ec): lint error when v3 preflight uses wrong apiVersion"
```
