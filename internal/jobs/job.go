package jobs

import (
	"sync/atomic"
	"time"
)

type JobState int32

const (
	StatePending JobState = iota
	StateRunning
	StateCommitted
	StateCompleted
	StateFailed
	StateCanceled
)

func (s JobState) String() string {
	switch s {
	case StatePending:
		return "pending"
	case StateRunning:
		return "running"
	case StateCommitted:
		return "committed"
	case StateCompleted:
		return "completed"
	case StateFailed:
		return "failed"
	case StateCanceled:
		return "canceled"
	default:
		return "unknown"
	}
}

type Job struct {
	ID        string
	Payload   any
	Timeout   time.Duration
	State     atomic.Int32
	Result    []byte
	Err       error
	CreatedAt time.Time
	UpdatedAt atomic.Int64
}

type JobResult struct {
	JobID   string
	Success bool
	Err     error
}

func (s JobState) isTerminal() bool {
	return s == StateCompleted || s == StateFailed || s == StateCanceled
}

func (j *Job) Transition(to JobState) bool {
	for {
		current := JobState(j.State.Load())
		if current.isTerminal() || current >= to {
			return false
		}
		if j.State.CompareAndSwap(int32(current), int32(to)) {
			j.UpdatedAt.Store(time.Now().UnixNano())
			return true
		}
	}
}
