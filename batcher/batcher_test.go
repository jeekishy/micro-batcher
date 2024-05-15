package batcher

import (
	"context"
	"testing"
	"time"

	"github.com/jeekishy/micro-batcher/processor"
	"github.com/stretchr/testify/assert"
)

func TestBatch_AddJob(t *testing.T) {
	b := mockBatcher(5, 10*time.Second)

	b.AddJob("Job 1")
	b.AddJob("Job 2")
	b.AddJob("Job 3")

	assert.Equal(t, 3, len(b.jobs), "unexpected number of jobs")
	b.ShutDown()
}

func TestBatch_ShutDown(t *testing.T) {
	b := mockBatcher(5, 10*time.Second)

	// shutdown batch then add job
	b.ShutDown()
	b.AddJob("Job 1")
	b.AddJob("Job 2")
	b.AddJob("Job 3")

	assert.Equal(t, 0, len(b.jobs), "unexpected number of jobs")
}

func TestBatch_MonitorJobResult(t *testing.T) {
	b := mockBatcher(2, 500*time.Millisecond)
	go b.monitorBatching(context.Background())

	// capture processed result
	results := make([]string, 0)

	// add upto batch limit
	b.AddJob("Job 1")
	b.AddJob("Job 2")
	b.AddJob("Job 3")
	b.ShutDown()

	// listen for processed job results
	jobResult := b.MonitorJobResult()
	for r := range jobResult {
		results = append(results, r)
	}

	// check all jobs has been processed
	assert.Equal(t, 3, len(results))
}

func mockBatcher(size int, frequency time.Duration) *batch {
	return &batch{
		size:           size,
		frequency:      frequency,
		enable:         true,
		jobs:           make([]any, 0),
		result:         make(chan string),
		triggerBatch:   make(chan struct{}, 1),
		stopBatch:      make(chan struct{}, 1),
		lockChan:       make(chan struct{}, 1),
		ticker:         time.NewTicker(frequency),
		batchProcessor: processor.New(),
	}
}
