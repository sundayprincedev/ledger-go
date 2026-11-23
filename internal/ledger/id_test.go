package ledger

import (
	"errors"
	"reflect"
	"testing"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
)

func inFlight(t *testing.T, r *IDRegistry, ids ...string) {
	t.Helper()
	for _, id := range ids {
		if res, err := r.Register(id); err != nil || !res.New {
			t.Fatalf("Register(%s): %+v, %v", id, res, err)
		}
	}
}

func stateOf(t *testing.T, r *IDRegistry, id string) JobState {
	t.Helper()
	res, err := r.LoadJob(id)
	if err != nil {
		t.Fatalf("LoadJob(%s): %v", id, err)
	}
	return res.State
}

func TestMarkPendingMovesAnInFlightJobAndBlocksANewClaim(t *testing.T) {
	r := NewIDRegistry()
	inFlight(t, r, "a")

	if err := r.MarkPending("a"); err != nil {
		t.Fatalf("MarkPending: %v", err)
	}

	if got := stateOf(t, r, "a"); got != StatePending {
		t.Fatalf("want pending, got %s", got)
	}
	res, err := r.Register("a")
	if err != nil || res.New || res.State != StatePending {
		t.Fatalf("a pending job must not be claimable and must report pending: %+v, %v", res, err)
	}
}

func TestMarkPendingOnlyAcceptsInFlightJobs(t *testing.T) {
	r := NewIDRegistry()
	inFlight(t, r, "a")
	if err := r.Complete("a"); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	if err := r.MarkPending("a"); !errors.Is(err, apperrors.ErrJobNotInFlight) {
		t.Fatalf("a completed job cannot become pending, got %v", err)
	}
	if err := r.MarkPending("missing"); !errors.Is(err, apperrors.ErrJobNotFound) {
		t.Fatalf("an unknown job cannot become pending, got %v", err)
	}
	if got := stateOf(t, r, "a"); got != StateCompleted {
		t.Fatalf("a rejected transition must not change state, got %s", got)
	}
}

func TestPendingIDsListsOnlyPendingJobsInOrder(t *testing.T) {
	r := NewIDRegistry()
	inFlight(t, r, "c", "a", "b", "d")
	for _, id := range []string{"c", "a", "b"} {
		if err := r.MarkPending(id); err != nil {
			t.Fatalf("MarkPending(%s): %v", id, err)
		}
	}
	if err := r.ResolveCompleted("b"); err != nil {
		t.Fatalf("ResolveCompleted: %v", err)
	}

	if got, want := r.PendingIDs(), []string{"a", "c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestResolveCompletedMakesThePendingJobCompleted(t *testing.T) {
	r := NewIDRegistry()
	inFlight(t, r, "a")
	_ = r.MarkPending("a")

	if err := r.ResolveCompleted("a"); err != nil {
		t.Fatalf("ResolveCompleted: %v", err)
	}
	if got := stateOf(t, r, "a"); got != StateCompleted {
		t.Fatalf("want completed, got %s", got)
	}
	if len(r.PendingIDs()) != 0 {
		t.Fatal("nothing should be pending any more")
	}
}

func TestResolveFailedFreesTheIDForARetry(t *testing.T) {
	r := NewIDRegistry()
	inFlight(t, r, "a")
	_ = r.MarkPending("a")

	if err := r.ResolveFailed("a"); err != nil {
		t.Fatalf("ResolveFailed: %v", err)
	}
	if res, err := r.Register("a"); err != nil || !res.New {
		t.Fatalf("a job that never landed must be claimable again: %+v, %v", res, err)
	}
}

func TestResolveRejectsJobsThatAreNotPending(t *testing.T) {
	r := NewIDRegistry()
	inFlight(t, r, "a")

	for name, resolve := range map[string]func(string) error{
		"ResolveCompleted": r.ResolveCompleted,
		"ResolveFailed":    r.ResolveFailed,
	} {
		if err := resolve("a"); !errors.Is(err, apperrors.ErrJobNotPending) {
			t.Fatalf("%s on an in-flight job: want ErrJobNotPending, got %v", name, err)
		}
		if err := resolve("missing"); !errors.Is(err, apperrors.ErrJobNotFound) {
			t.Fatalf("%s on an unknown job: want ErrJobNotFound, got %v", name, err)
		}
	}
	if got := stateOf(t, r, "a"); got != StateInFlight {
		t.Fatalf("a rejected resolve must not change state, got %s", got)
	}
}
