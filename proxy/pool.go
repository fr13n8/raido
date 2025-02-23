package proxy

import (
	"sync"
	"sync/atomic"
	"time"
)

// WorkerPool manages a dynamic pool of workers.
type WorkerPool struct {
	taskQueue     chan func()    // Queue for tasks
	maxWorkers    int            // Maximum number of workers
	minWorkers    int            // Minimum number of workers
	idleTimeout   time.Duration  // Time after which idle workers stop
	activeWorkers int32          // Current number of active workers
	workerWG      sync.WaitGroup // Wait group for worker shutdown
	stop          chan struct{}  // Signal to stop the pool
}

// NewWorkerPool creates a new WorkerPool.
func NewWorkerPool(minWorkers, maxWorkers int, idleTimeout time.Duration) *WorkerPool {
	return &WorkerPool{
		taskQueue:   make(chan func(), 1000),
		maxWorkers:  maxWorkers,
		minWorkers:  minWorkers,
		idleTimeout: idleTimeout,
		stop:        make(chan struct{}),
	}
}

// Start initializes the worker pool with the minimum number of workers.
func (wp *WorkerPool) Start() {
	for range wp.minWorkers {
		wp.startWorker()
	}
}

// Stop shuts down the worker pool gracefully.
func (wp *WorkerPool) Stop() {
	close(wp.stop)
	wp.workerWG.Wait()
}

// Submit adds a task to the worker pool.
func (wp *WorkerPool) Submit(task func()) {
	select {
	case wp.taskQueue <- task:
		// Task submitted successfully
	default:
		// Queue is full; start a new worker if below maxWorkers
		if int(atomic.LoadInt32(&wp.activeWorkers)) < wp.maxWorkers {
			wp.startWorker()
			wp.taskQueue <- task
		} else {
			// If at max workers, block until a slot is available
			wp.taskQueue <- task
		}
	}
}

// startWorker launches a new worker goroutine.
func (wp *WorkerPool) startWorker() {
	atomic.AddInt32(&wp.activeWorkers, 1)
	wp.workerWG.Add(1)
	go func() {
		defer wp.workerWG.Done()
		defer atomic.AddInt32(&wp.activeWorkers, -1)
		for {
			select {
			case task, ok := <-wp.taskQueue:
				if !ok {
					return
				}
				task()
			case <-time.After(wp.idleTimeout):
				// Stop worker if idle too long and above minWorkers
				if int(atomic.LoadInt32(&wp.activeWorkers)) > wp.minWorkers {
					return
				}
			case <-wp.stop:
				return
			}
		}
	}()
}
