package ec

import (
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots-lint/pkg/domain"
	"gopkg.in/yaml.v2"
)

var ecVersions map[string]bool
var rwMutex sync.RWMutex

func init() {
	ecVersions = make(map[string]bool)
}

func LintEmbeddedClusterVersion(specFiles domain.SpecFiles) ([]domain.LintExpression, error) {
	lintExpressions := []domain.LintExpression{}
	// separate multi docs because the manifest can be a part of a multi doc yaml file
	separatedSpecFiles, err := specFiles.Separate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to separate multi docs")
	}

	for _, spec := range separatedSpecFiles {
		var version string
		var versionExists bool
		doc := map[string]interface{}{}
		if err := yaml.Unmarshal([]byte(spec.Content), &doc); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal spec content")
		}
		if doc["apiVersion"] == "embeddedcluster.replicated.com/v1beta1" && doc["kind"] == "Config" {
			if spec, ok := doc["spec"].(map[interface{}]interface{}); ok {
				version, versionExists = spec["version"].(string)
			}
		}

		// if no version exists, continue to next file
		if !versionExists {
			continue
		} else {
			exists, err := checkIfECVersionExists(version)
			if err != nil {
				return nil, errors.Wrap(err, "failed to check if ec version exists")
			}
			if !exists {
				ecVersionlintExpression := domain.LintExpression{
					Rule:    "non-existent-ec-version",
					Type:    "error",
					Path:    spec.Path,
					Message: "Embedded Cluster version not found",
				}
				lintExpressions = append(lintExpressions, ecVersionlintExpression)
			}
		}
	}

	return lintExpressions, nil
}

func checkIfECVersionExists(version string) (bool, error) {
	url := "http://api.github.com/repos/replicatedhq/embedded-cluster/releases/tags/%s"
	token := os.Getenv("GITHUB_API_TOKEN")
	var bearer = "Bearer " + token

	rwMutex.RLock()
	verIsCached := ecVersions[version]
	rwMutex.RUnlock()

	if !verIsCached {
		req, err := http.NewRequest("GET", fmt.Sprintf(url, version), nil)
		if err != nil {
			return false, errors.Wrap(err, "failed to create new request")
		}
		req.Header.Set("Authorization", bearer)
		client := &http.Client{}
		resp, _ := client.Do(req)
		if resp.StatusCode == 404 {
			return false, nil
		} else if resp.StatusCode == 200 {
			rwMutex.Lock()
			ecVersions[version] = true
			rwMutex.Unlock()
		} else {
			return false, errors.New(fmt.Sprintf("received non 200 status code (%d) from GitHub API request", resp.StatusCode))
		}
	}

	return true, nil
}
