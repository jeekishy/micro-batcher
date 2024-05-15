package processor

import (
	"fmt"
	"time"
)

type Processor interface {
	Process(jobs []any) chan string
}

type process struct {
}

func New() Processor {
	return &process{}
}

// Process is a dummy processor which will process a list of something
func (p *process) Process(jobs []any) chan string {
	jobResultChan := make(chan string)

	go func() {
		for _, j := range jobs {
			// ideally I would use a shared type, but I wanted to make this flexible
			// thus this would involve using type assertion to determine the job types
			// for example when the job is a string
			if data, ok := j.(string); ok {
				// simulate something being processed
				time.Sleep(100 * time.Millisecond)
				jobResultChan <- fmt.Sprintf("data %s successfully processed", data)
			}
		}

		close(jobResultChan)
	}()

	return jobResultChan
}
