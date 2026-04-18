package worker

import (
	"context"
	"errors"
	"log"
	"time"
)

// Job is a unit of work queued for flash-sale spikes.
// Each job should be quick to enqueue and safe to run in a goroutine.
type Job struct {
	Name   string
	Do     func(context.Context) error
	Result chan error
}

type WorkerPool struct {
	numWorkers int
	jobs       chan Job // Buffered channel to absorb spikes.
	stop       chan struct{}
}

func NewWorkerPool(numWorkers, bufferSize int) *WorkerPool {
	if numWorkers <= 0 {
		numWorkers = 1
	}
	if bufferSize <= 0 {
		bufferSize = 1000
	}
	return &WorkerPool{
		numWorkers: numWorkers,
		jobs:       make(chan Job, bufferSize),
		stop:       make(chan struct{}),
	}
}

func (wp *WorkerPool) Start() {
	for i := 0; i < wp.numWorkers; i++ {
		go wp.worker(i)
	}
	log.Printf("Worker pool started with %d workers (buffer=%d)\n", wp.numWorkers, cap(wp.jobs))
}

func (wp *WorkerPool) Stop() {
	close(wp.stop)
}

// Submit enqueues a job. If the queue is full, it returns an error immediately.
func (wp *WorkerPool) Submit(job Job) error {
	select {
	case wp.jobs <- job:
		return nil
	default:
		return errors.New("payment queue is full, please retry")
	}
}

func (wp *WorkerPool) worker(id int) {
	log.Printf("Worker %d started\n", id)
	defer log.Printf("Worker %d stopped\n", id)

	for {
		select {
		case <-wp.stop:
			return
		case job := <-wp.jobs:
			if job.Do == nil {
				if job.Result != nil {
					job.Result <- nil
				}
				continue
			}
			// Each job gets a bounded execution window.
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			err := job.Do(ctx)
			cancel()

			if job.Result != nil {
				select {
				case job.Result <- err:
				default:
				}
			}
		}
	}
}
