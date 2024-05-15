package batcher

import (
	"context"
	"fmt"
	"time"
)

// Batcher provides a structured way to interact with the micro-batcher
type Batcher interface {
	AddJob(j any)
	MonitorJobResult() <-chan string
	ShutDown()
}

// BatchProcessor allows dependency inversion thus allowing us to use any
// batch processor which implements similar interface
type BatchProcessor interface {
	Process(jobs []any) chan string
}

type batch struct {
	size           int // batch limit
	frequency      time.Duration
	enable         bool // controls if batching is allowed or not
	jobs           []any
	result         chan string // to send out processed results once received
	batchProcessor BatchProcessor
	triggerBatch   chan struct{} // channel to get a signal when to batch next lot of jobs
	stopBatch      chan struct{} // channel to send stop signal to stop batching
	lockChan       chan struct{} // channel used as a lock
	ticker         *time.Ticker  // to manage frequency
}

type JobResult struct {
	ID         string
	CompleteAt time.Time
}

func New(ctx context.Context, size int, frequency time.Duration, bProcessor BatchProcessor) Batcher {
	b := &batch{
		size:           size,
		frequency:      frequency,
		enable:         true,
		jobs:           make([]any, 0),
		result:         make(chan string),
		triggerBatch:   make(chan struct{}, 1),
		stopBatch:      make(chan struct{}, 1),
		lockChan:       make(chan struct{}, 1),
		batchProcessor: bProcessor,
		ticker:         time.NewTicker(frequency),
	}

	// start batcher
	go b.monitorBatching(ctx)

	return b
}

func (b *batch) AddJob(j any) {
	// do not accept any job once a shutdown is triggered
	if !b.enable {
		return
	}

	b.lock()
	b.jobs = append(b.jobs, j)
	fmt.Printf("job: %s added\n", j)
	b.unlock()

	// check we have reach the batch limit
	// send a signal in the channel that job can be batched and processed
	if len(b.jobs) == b.size {
		b.triggerBatch <- struct{}{}
	}

	// reset ticker when first job is batched,
	// this should auto trigger batching once frequency is reached
	if len(b.jobs) == 1 {
		b.ticker.Reset(b.frequency)
	}
}

// monitorBatching listens to two different channels for when we are ready to batch
func (b *batch) monitorBatching(ctx context.Context) {
	for {
		select {
		case <-b.stopBatch:
			fmt.Println("stopping batching from shutdown method")
			return

		case <-ctx.Done():
			fmt.Println("stopping batching from syscall")
			return

		case <-b.ticker.C:
			b.sendBatchedJobsForProcessing()

		case <-b.triggerBatch:
			b.sendBatchedJobsForProcessing()
		}
	}
}

// sendBatchedJobsForProcessing - send current jobs in the list to be batched process
func (b *batch) sendBatchedJobsForProcessing() {
	b.lock()

	// stop our ticker so as we do not keep on getting further batching signals
	b.ticker.Stop()

	// even if we have reach our time frequency
	// we want to keep within our batch limit, so only batch upto the limit
	var jobsToBatch []any
	if len(b.jobs) >= b.size {
		jobsToBatch, b.jobs = b.jobs[:b.size], b.jobs[b.size:]
	} else {
		jobsToBatch = b.jobs
		b.jobs = make([]any, 0)
	}

	// send jobs to be batched and clear jobs sent for batching
	// I could have also used a channel as a queue to store the jobs but
	//this one as it is a simpler implementation
	result := b.batchProcessor.Process(jobsToBatch)

	// we want to keep the ticker going until we do not have any jobs to batch
	if len(b.jobs) > 0 {
		b.ticker.Reset(b.frequency)
	}

	b.unlock()

	// handle job result from job processor
	// this will terminate all job is processed and channel is closed
	go func() {
		for r := range result {
			b.result <- r
		}

		// once processing is completed, check if we need to stop batching
		if len(b.jobs) == 0 && !b.enable {
			b.stopBatch <- struct{}{}
			close(b.result)
		}
	}()
}

// MonitorJobResult - provides a way to get batched job result
func (b *batch) MonitorJobResult() <-chan string {
	return b.result
}

// ShutDown - will set a flag which gets check before adding a job and after batching is processed
func (b *batch) ShutDown() {
	b.lock()
	defer b.unlock()

	b.enable = false
}

// lock - simulates a block by implementing a buffered channel of 1
// could have used a sync mutex but prefer the simplicity of channels to obtain similar effect
func (b *batch) lock() {
	b.lockChan <- struct{}{}
}

// unlock - frees up the channel so as new lock can be assigned
func (b *batch) unlock() {
	<-b.lockChan
}
