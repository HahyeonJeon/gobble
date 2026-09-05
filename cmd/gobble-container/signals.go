package main

import (
	"context"
	"os"
	"os/signal"
	"time"
)

func forwardInterrupt(container string) func() {
	interrupts := make(chan os.Signal, 1)
	done := make(chan struct{})
	signal.Notify(interrupts, os.Interrupt)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-interrupts:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				_, _ = dockerOutput(ctx, "kill", "--signal=SIGINT", container)
				cancel()
			}
		}
	}()
	return func() { signal.Stop(interrupts); close(done) }
}
