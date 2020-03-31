package main

import (
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/replicatedcom/saaskit/log"
	"github.com/replicatedhq/kots-lint/newrelic"
	"github.com/replicatedhq/kots-lint/pkg/daemon"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	newrelicApp, err := newrelic.GetNewRelicApp()
	if err != nil {
		log.Errorf("Failed to configure newrelic: %v", err)
		os.Exit(1)
	}

	go daemon.Run(newrelicApp)

	term := make(chan os.Signal)
	signal.Notify(term, syscall.SIGINT, syscall.SIGTERM)
	<-term
}
