package main

import (
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/replicatedhq/kots-lint/pkg/daemon"
	"github.com/replicatedhq/kots-lint/pkg/kots"
	"github.com/replicatedhq/kots-lint/pkg/version"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	log "github.com/sirupsen/logrus"

)

func main() {
	tracer.Start(
		tracer.WithService("kots-lint"),
		tracer.WithServiceVersion(version.GitSHA),
		tracer.WithAgentAddr("dd-agent.internal:8126"),
	)
	defer tracer.Stop()

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
