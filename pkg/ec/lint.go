package ec

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots-lint/pkg/domain"
	"gopkg.in/yaml.v2"
)

var ecVersions map[string]EmbeddedClusterVersion
var rwMutex sync.RWMutex
var githubAPIURL = "http://api.github.com"

type EmbeddedClusterVersion struct {
	PreRelease bool `json:"prerelease"`
}

func init() {
	ecVersions = make(map[string]EmbeddedClusterVersion)
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
			// if no version is defined, return error version is required
			if !versionExists {
				ecVersionlintExpression := domain.LintExpression{
					Rule:    "ec-version-required",
					Type:    "error",
					Path:    spec.Path,
					Message: "Embedded Cluster version is required",
				}
				lintExpressions = append(lintExpressions, ecVersionlintExpression)
			} else {
				// version is defined, check if it is valid.
				ecVersion, exists, err := checkIfECVersionExists(version)
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
				} else if ecVersion.PreRelease {
					ecVersionlintExpression := domain.LintExpression{
						Rule:    "non-existent-ec-version",
						Type:    "error",
						Path:    spec.Path,
						Message: "Embedded Cluster version is a pre-release",
					}
					lintExpressions = append(lintExpressions, ecVersionlintExpression)
				}
			}
		}
	}

	return lintExpressions, nil
}

func checkIfECVersionExists(version string) (*EmbeddedClusterVersion, bool, error) {
	url := githubAPIURL + "/repos/replicatedhq/embedded-cluster/releases/tags/%s"
	token := os.Getenv("GITHUB_API_TOKEN")
	var bearer = "Bearer " + token

	rwMutex.RLock()
	ecVersion, found := ecVersions[version]
	rwMutex.RUnlock()

	if found {
		return &ecVersion, true, nil
	}

	req, err := http.NewRequest("GET", fmt.Sprintf(url, version), nil)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to create new request")
	}
	req.Header.Set("Authorization", bearer)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to make http request")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, false, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, false, errors.New(fmt.Sprintf("received non 200 status code (%d) from GitHub API request", resp.StatusCode))
	}

	var newVersion EmbeddedClusterVersion
	if err := json.NewDecoder(resp.Body).Decode(&newVersion); err != nil {
		return nil, false, errors.Wrap(err, "failed to decode embedded cluster version json")
	}

	if newVersion.PreRelease {
		return &newVersion, true, nil
	}

	rwMutex.Lock()
	ecVersions[version] = newVersion
	rwMutex.Unlock()

	return &newVersion, true, nil
}
