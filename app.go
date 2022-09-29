package main

import (
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	kjs "github.com/replicatedhq/kots-lint/kubernetes_json_schema"
	"github.com/replicatedhq/kots-lint/pkg/daemon"
	"github.com/replicatedhq/kots-lint/pkg/kots"
	log "github.com/sirupsen/logrus"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	schemaDir, err := kjs.InitKubernetesJsonSchemaDir()
	if err != nil {
		log.Errorf("failed to init kubernetes json schema dir: %v", err)
		os.Exit(1)
	}
	defer os.RemoveAll(schemaDir)

	if err := kots.InitOPALinting(); err != nil {
		log.Errorf("failed to init opa linting: %v", err)
		os.Exit(1)
	}

	go daemon.Run()

	term := make(chan os.Signal)
	signal.Notify(term, syscall.SIGINT, syscall.SIGTERM)
	<-term
}
