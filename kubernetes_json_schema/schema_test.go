package kubernetes_json_schema

import (
	"embed"
	"os"
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

	content, err := os.ReadFile(filepath.Join(schemaDir, "v1.33.3-standalone-strict", "configmap.json"))
	if err != nil {
		t.Errorf("InitKubernetesJsonSchemaDir() failed to read configmap.json")
		return
	}
	if len(content) == 0 {
		t.Errorf("InitKubernetesJsonSchemaDir() configmap.json is empty")
		return
	}

	content, err = os.ReadFile(filepath.Join(schemaDir, "v1.33.3-standalone-strict", "configmap-v1.json"))
	if err != nil {
		t.Errorf("InitKubernetesJsonSchemaDir() failed to read configmap-v1.json")
		return
	}
	if len(content) == 0 {
		t.Errorf("InitKubernetesJsonSchemaDir() configmap-v1.json is empty")
		return
	}

	content, err = os.ReadFile(filepath.Join(schemaDir, "v1.33.3-standalone-strict", "airgap-kots-v1beta1.json"))
	if err != nil {
		t.Errorf("InitKubernetesJsonSchemaDir() failed to read airgap-kots-v1beta1.json")
		return
	}
	if len(content) == 0 {
		t.Errorf("InitKubernetesJsonSchemaDir() airgap-kots-v1beta1.json is empty")
		return
	}
}
