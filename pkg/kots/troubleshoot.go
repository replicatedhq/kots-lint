package kots

import (
	"context"
	_ "embed"
	"fmt"
	"path"
	"strings"

	troubleshootscheme "github.com/replicatedhq/troubleshoot/pkg/client/troubleshootclientset/scheme"
	"github.com/replicatedhq/troubleshoot/pkg/constants"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var decoder runtime.Decoder

func init() {
	_ = v1.AddToScheme(troubleshootscheme.Scheme) // for secrets and configmaps
	decoder = troubleshootscheme.Codecs.UniversalDeserializer()
}

func GetEmbeddedTroubleshootSpecs(ctx context.Context, specsFiles SpecFiles) SpecFiles {
	tsSpecs := SpecFiles{}

	for _, specFile := range specsFiles {
		troubleshootSpecs := findTroubleshootSpecs(ctx, specFile.Content)
		for _, tsSpec := range troubleshootSpecs {
			tsSpecs = append(tsSpecs, SpecFile{
				Name:            path.Join(specFile.Name, tsSpec.Name),
				Path:            specFile.Name,
				Content:         tsSpec.Content,
				AllowDuplicates: tsSpec.AllowDuplicates,
			})
		}
	}

	return tsSpecs
}

// Extract troubleshoot specs from ConfigMap and Secret specs
func findTroubleshootSpecs(ctx context.Context, fileData string) SpecFiles {
	tsSpecs := SpecFiles{}

	srcDocs := strings.Split(fileData, "\n---\n")
	for _, srcDoc := range srcDocs {
		obj, _, err := decoder.Decode([]byte(srcDoc), nil, nil)
		if err != nil {
			log.Debugf("failed to decode raw spec: %s", srcDoc)
			continue
		}

		switch v := obj.(type) {
		case *v1.ConfigMap:
			specs := getSpecFromConfigMap(v, fmt.Sprintf("%s-", v.Name))
			tsSpecs = append(tsSpecs, specs...)
		case *v1.Secret:
			specs := getSpecFromSecret(v, fmt.Sprintf("%s-", v.Name))
			tsSpecs = append(tsSpecs, specs...)
		}
	}

	return tsSpecs
}

func getSpecFromConfigMap(cm *v1.ConfigMap, namePrefix string) SpecFiles {
	possibleKeys := []string{
		constants.SupportBundleKey,
		constants.RedactorKey,
		constants.PreflightKey,
		constants.PreflightKey2,
	}

	specs := SpecFiles{}
	for _, key := range possibleKeys {
		str, ok := cm.Data[key]
		if ok {
			specs = append(specs, SpecFile{
				Name:            namePrefix + key,
				Content:         str,
				AllowDuplicates: true,
			})
		}
	}

	return specs
}

func getSpecFromSecret(secret *v1.Secret, namePrefix string) SpecFiles {
	possibleKeys := []string{
		constants.SupportBundleKey,
		constants.RedactorKey,
		constants.PreflightKey,
		constants.PreflightKey2,
	}

	specs := SpecFiles{}
	for _, key := range possibleKeys {
		data, ok := secret.Data[key]
		if ok {
			specs = append(specs, SpecFile{
				Name:            namePrefix + key,
				Content:         string(data),
				AllowDuplicates: true,
			})
		}

		str, ok := secret.StringData[key]
		if ok {
			specs = append(specs, SpecFile{
				Name:            namePrefix + key,
				Content:         str,
				AllowDuplicates: true,
			})
		}
	}

	return specs
}
