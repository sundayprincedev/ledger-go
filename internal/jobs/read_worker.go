package jobs

import "context"

type ReadWorker struct {
	*WorkerPool
}

func NewReadWorker(parent context.Context) *ReadWorker {
	return &ReadWorker{WorkerPool: NewWorkerPool(parent)}
}
