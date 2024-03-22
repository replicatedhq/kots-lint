package handlers

import (
	"embed"

	kjs "github.com/replicatedhq/kots-lint/kubernetes_json_schema"
	"github.com/replicatedhq/kots-lint/pkg/kots"
)

func init() {
	kjs.KubernetesJsonSchemaDir = "../../kubernetes_json_schema/schema"
	kots.InitOPALinting()
}

//go:embed test-data/*
var testdata embed.FS
