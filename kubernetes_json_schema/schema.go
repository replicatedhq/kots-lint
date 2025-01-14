package kubernetes_json_schema

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

const KUBERNETES_LINT_VERSION = "1.31.4"

//go:embed schema/**/*.json
var kubernetesJsonSchemaFS embed.FS

// the directory that holds the kubernetes json schema files
var KubernetesJsonSchemaDir string

func InitKubernetesJsonSchemaDir() (string, error) {
	return initKubernetesJsonSchemaDir(kubernetesJsonSchemaFS)
}

func initKubernetesJsonSchemaDir(schemaFS embed.FS) (string, error) {
	tempDir, err := os.MkdirTemp("", "kubernetesjsonschema")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp dir")
	}

	err = fs.WalkDir(schemaFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		data, err := schemaFS.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "failed to read file %s", path)
		}

		parts := strings.Split(path, string(os.PathSeparator))
		if len(parts) < 3 {
			return nil
		}
		destPath := filepath.Join(parts[2:]...) // trim root directory
		destPath = filepath.Join(fmt.Sprintf("v%s-standalone-strict", KUBERNETES_LINT_VERSION), destPath)

		destDir := filepath.Dir(filepath.Join(tempDir, destPath))
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return errors.Wrapf(err, "failed to create dir %s", destDir)
		}

		if err := os.WriteFile(filepath.Join(tempDir, destPath), data, 0755); err != nil {
			return errors.Wrap(err, "failed to write file")
		}

		return nil
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to walk kubernetes json schema dir")
	}

	KubernetesJsonSchemaDir = tempDir

	return tempDir, nil
}
