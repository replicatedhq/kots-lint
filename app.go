package main

import (
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/replicatedhq/kots-lint/daemon"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	go daemon.Run()

	term := make(chan os.Signal)
	signal.Notify(term, syscall.SIGINT, syscall.SIGTERM)
	<-term
}
