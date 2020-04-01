package main

import (
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/replicatedcom/saaskit/log"
	"github.com/replicatedhq/kots-lint/pkg/daemon"
	"github.com/replicatedhq/kots-lint/pkg/instrument"
	"github.com/replicatedhq/kots-lint/pkg/kots"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	newrelicApp, err := instrument.GetNewRelicApp()
	if err != nil {
		log.Errorf("failed to configure newrelic: %v", err)
		os.Exit(1)
	}

	err = kots.InitOPALinting("/rego/kots-spec-default.rego")
	if err != nil {
		log.Errorf("failed to init opa linting: %v", err)
		os.Exit(1)
	}

	go daemon.Run(newrelicApp)

	term := make(chan os.Signal)
	signal.Notify(term, syscall.SIGINT, syscall.SIGTERM)
	<-term
}
