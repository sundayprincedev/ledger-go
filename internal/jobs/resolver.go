package jobs

import (
	"context"
	"errors"
	"time"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
	"github.com/sundayprincedev/ledger/internal/ledger"
)

type ResolverResult struct {
	Completed []string
	Failed    []string
}

func StartResolverPipeline(ctx context.Context, journal ledger.Journal, registry *ledger.IDRegistry, balances *ledger.Balances) (ResolverResult, error) {
	var result ResolverResult

	pendingIDs := registry.PendingIDs()
	if len(pendingIDs) == 0 {
		return result, nil
	}

	unresolved := make(map[string]struct{}, len(pendingIDs))
	for _, id := range pendingIDs {
		unresolved[id] = struct{}{}
	}

	if err := checkResolveContext(ctx, -1); err != nil {
		return result, err
	}

	journalEntries, err := journal.Recover()
	if err != nil {
		return result, &apperrors.ResolvePipelineError{Step: apperrors.ResolveStepRecover, Index: -1, Err: err}
	}

	for i, raw := range journalEntries {
		if err := checkResolveContext(ctx, i); err != nil {
			return result, err
		}

		cleanEntry, decodeErr := ledger.EntryFromPayload(raw.Payload, time.Unix(0, raw.Timestamp))
		if decodeErr != nil {
			return result, &apperrors.ResolvePipelineError{Step: apperrors.ResolveStepDecode, Index: i, Err: decodeErr}
		}

		lookup, lookupErr := registry.LoadJob(cleanEntry.ID)
		if errors.Is(lookupErr, apperrors.ErrJobNotFound) {
			continue
		}
		if lookupErr != nil {
			return result, &apperrors.ResolvePipelineError{Step: apperrors.ResolveStepComplete, Index: i, EntryID: cleanEntry.ID, Err: lookupErr}
		}

		switch lookup.State {
		case ledger.StateCompleted:
			continue
		case ledger.StatePending:
		default:
			return result, &apperrors.ResolvePipelineError{
				Step:    apperrors.ResolveStepComplete,
				Index:   i,
				EntryID: cleanEntry.ID,
				Err:     &apperrors.UnexpectedJobStateError{JobID: cleanEntry.ID, State: string(lookup.State)},
			}
		}

		if applyErr := balances.Apply(cleanEntry); applyErr != nil {
			return result, &apperrors.ResolvePipelineError{Step: apperrors.ResolveStepApply, Index: i, EntryID: cleanEntry.ID, Err: applyErr}
		}
		if completeErr := registry.ResolveCompleted(cleanEntry.ID); completeErr != nil {
			return result, &apperrors.ResolvePipelineError{Step: apperrors.ResolveStepComplete, Index: i, EntryID: cleanEntry.ID, Err: completeErr}
		}

		delete(unresolved, cleanEntry.ID)
		result.Completed = append(result.Completed, cleanEntry.ID)
	}

	for _, id := range pendingIDs {
		if _, stillPending := unresolved[id]; !stillPending {
			continue
		}
		if failErr := registry.ResolveFailed(id); failErr != nil {
			return result, &apperrors.ResolvePipelineError{Step: apperrors.ResolveStepFail, Index: -1, EntryID: id, Err: failErr}
		}
		result.Failed = append(result.Failed, id)
	}

	return result, nil
}
