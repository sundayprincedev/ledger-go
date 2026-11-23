package jobs

import (
	"context"
	"time"

	apperrors "github.com/sundayprincedev/ledger/internal/errors"
	"github.com/sundayprincedev/ledger/internal/journal"
	"github.com/sundayprincedev/ledger/internal/ledger"
)

type ReplayResult struct {
	Recovered  int
	Registered int
	Duplicates []*apperrors.DuplicateJournalEntryError
}

func StartReadPipeline(ctx context.Context, journalPath string, registry *ledger.IDRegistry) (ReplayResult, error) {
	var result ReplayResult
	pipelineCtx := ctx

	if err := checkReadContext(pipelineCtx, -1); err != nil {
		return result, err
	}

	recoveryEntries, recoveryErr := journal.RecoverAndClean(journalPath)
	if recoveryErr != nil {
		return ReplayResult{}, &apperrors.ReadPipelineError{Step: apperrors.ReadStepRecover, Index: -1, Err: recoveryErr}
	}

	for i, raw := range recoveryEntries {
		if err := checkReadContext(pipelineCtx, i); err != nil {
			return result, err
		}

		t := time.Unix(0, raw.Timestamp)
		cleanEntry, entryfrompayloadErr := ledger.EntryFromPayload(raw.Payload, t)
		if entryfrompayloadErr != nil {
			return result, &apperrors.ReadPipelineError{Step: apperrors.ReadStepDecode, Index: i, Err: entryfrompayloadErr}
		}

		if registry.SeedCompleted(cleanEntry.ID) {
			result.Duplicates = append(result.Duplicates, &apperrors.DuplicateJournalEntryError{EntryID: cleanEntry.ID, Index: i})
			continue
		}

		result.Registered++
	}

	result.Recovered = len(recoveryEntries)

	return result, nil
}
