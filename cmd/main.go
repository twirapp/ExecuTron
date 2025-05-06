package main

import (
	"github.com/twirapp/executron/internal/server"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	s := server.New()
	s.Run()

	defer s.Stop()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	<-signals
}
