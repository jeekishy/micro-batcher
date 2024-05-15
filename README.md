# Micro-Batching
Micro-batcher which allows batching of jobs and setting a frequency and size limit.

## To  Micro-batcher
```go
   
// initialise micro-batcher with limit, frequency and a JobProcessor
b := batcher.New(context.Background(), 5, 5*time.Second, processor.New())

// add any type of job which your job processor can handle
b.AddJob("Job 1")
b.AddJob("Job 2")
	
// listen for processed job results
go func() {
    jobResult := b.MonitorJobResult()
    for r := range jobResult {
        results = append(results, r)
    }
}
	
// to stop batcher
b.ShutDown()
```