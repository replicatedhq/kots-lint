package main

import (
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/replicatedhq/kots-lint/pkg/daemon"
	"github.com/replicatedhq/kots-lint/pkg/instrument"
	"github.com/replicatedhq/kots-lint/pkg/kots"
	log "github.com/sirupsen/logrus"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	newrelicApp, err := instrument.GetNewRelicApp()
	if err != nil {
		log.Errorf("failed to configure newrelic: %v", err)
		os.Exit(1)
	}

	if err := kots.InitOPALinting("/rego"); err != nil {
		log.Errorf("failed to init opa linting: %v", err)
		os.Exit(1)
	}

	go daemon.Run(newrelicApp)

	term := make(chan os.Signal)
	signal.Notify(term, syscall.SIGINT, syscall.SIGTERM)
	<-term
}
