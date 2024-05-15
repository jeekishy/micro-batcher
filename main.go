package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jeekishy/micro-batcher/batcher"
	"github.com/jeekishy/micro-batcher/processor"
)

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	dummyProcessor := processor.New()
	b := batcher.New(ctx, 5, 5*time.Second, dummyProcessor)

	// these should process in a batch and triggered by size limit
	b.AddJob("Job 1")
	b.AddJob("Job 2")
	b.AddJob("Job 3")
	b.AddJob("Job 4")
	b.AddJob("Job 5")

	// these should be process in another batch, triggered by frequency
	b.AddJob("Job 6")
	b.AddJob("Job 7")
	b.AddJob("Job 8")

	b.ShutDown()

	// monitor batching results
	go func() {
		jobResult := b.MonitorJobResult()

		for {
			select {
			case r := <-jobResult:
				if len(r) == 0 {
					return
				}
				fmt.Printf("%s at %s\n", r, time.Now().Format(time.RFC3339))
			case <-ctx.Done():
				return
			}
		}
	}()

	// system call shutdown
	<-ctx.Done()
	fmt.Println("shutting down")
}
