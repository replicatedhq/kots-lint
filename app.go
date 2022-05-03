package main

import (
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/replicatedhq/kots-lint/pkg/daemon"
	"github.com/replicatedhq/kots-lint/pkg/kots"
	"github.com/replicatedcom/saaskit/tracing/datadog"
	"github.com/replicatedhq/kots-line/pkg/version"
	log "github.com/sirupsen/logrus"
)

func main() {
	datadog.StartTracer("kots-lint", version.GitSHA())
	defer datadog.StopTracer()

	rand.Seed(time.Now().UTC().UnixNano())

	if err := kots.InitOPALinting("/rego"); err != nil {
		log.Errorf("failed to init opa linting: %v", err)
		os.Exit(1)
	}

	go daemon.Run()

	term := make(chan os.Signal)
	signal.Notify(term, syscall.SIGINT, syscall.SIGTERM)
	<-term
}
