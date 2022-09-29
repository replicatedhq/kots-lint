package kubernetes_json_schema

import (
	"embed"
	"io/ioutil"
	"path/filepath"
	"testing"
)

//go:embed test-schema/**/*.json
var testSchemaFS embed.FS

func TestInitKubernetesJsonSchemaDir(t *testing.T) {
	schemaDir, err := initKubernetesJsonSchemaDir(testSchemaFS)
	if err != nil {
		t.Errorf("InitKubernetesJsonSchemaDir() error = %v", err)
		return
	}

	content, err := ioutil.ReadFile(filepath.Join(schemaDir, "v1.23.6-standalone-strict", "configmap.json"))
	if err != nil {
		t.Errorf("InitKubernetesJsonSchemaDir() failed to read configmap.json")
		return
	}
	if len(content) == 0 {
		t.Errorf("InitKubernetesJsonSchemaDir() configmap.json is empty")
		return
	}

	content, err = ioutil.ReadFile(filepath.Join(schemaDir, "v1.23.6-standalone-strict", "configmap-v1.json"))
	if err != nil {
		t.Errorf("InitKubernetesJsonSchemaDir() failed to read configmap-v1.json")
		return
	}
	if len(content) == 0 {
		t.Errorf("InitKubernetesJsonSchemaDir() configmap-v1.json is empty")
		return
	}
}
