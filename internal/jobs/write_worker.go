package jobs

import (
	"context"
	"fmt"
	"runtime"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
)

type WriteWorker struct {
	*WorkerPool
}

func NewWriteWorker(parent context.Context) *WriteWorker {
	return &WriteWorker{WorkerPool: NewWorkerPool(parent)}
}

func (wr *WriteWorker) StartWritePipeline(ctx context.Context, intent Intent) (err error) {
	var jobID string

	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			err = apperrors.NewWorkerJobErrorWithStack(jobID, "panic", fmt.Errorf("%v", r), string(buf[:n]))
		}
	}()

	pipelineCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobID = GenerateJobID(intent)
	_ = pipelineCtx

	return nil
}
