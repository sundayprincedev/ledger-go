package ledger

import (
	"sync"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
)

type JobState string

const (
	StateInFlight  JobState = "in_flight"
	StateCompleted JobState = "completed"
	StatePending   JobState = "pending"
)

type JobIDLookupResult struct {
	State JobState
	New   bool
}

type IDRegistry struct {
	mu sync.Mutex
	m  map[string]JobState
}

func NewIDRegistry() *IDRegistry {
	return &IDRegistry{m: make(map[string]JobState)}
}

func (r *IDRegistry) Register(id string) (JobIDLookupResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if state, exists := r.m[id]; exists {
		return JobIDLookupResult{State: state, New: false}, nil
	}
	r.m[id] = StateInFlight
	return JobIDLookupResult{State: StateInFlight, New: true}, nil
}

func (r *IDRegistry) LoadJob(id string) (JobIDLookupResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, ok := r.m[id]
	if !ok {
		return JobIDLookupResult{}, &apperrors.JobError{JobID: id, Err: apperrors.ErrJobNotFound}
	}
	return JobIDLookupResult{State: state}, nil
}

func (r *IDRegistry) Complete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, ok := r.m[id]
	if !ok {
		return &apperrors.JobError{JobID: id, Err: apperrors.ErrJobNotFound}
	}
	if state != StateInFlight {
		return &apperrors.JobNotPendingError{JobID: id, State: string(state)}
	}
	r.m[id] = StateCompleted
	return nil
}

func (r *IDRegistry) SeedCompleted(id string) (duplicate bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.m[id]; exists {
		return true
	}

	r.m[id] = StateCompleted
	return false
}

func (r *IDRegistry) Release(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.m[id]; !ok {
		return &apperrors.JobError{JobID: id, Err: apperrors.ErrJobNotFound}
	}
	delete(r.m, id)
	return nil
}
