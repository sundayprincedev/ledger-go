package main

import (
	"github.com/sundayprincedev/ledger/internal/data"
	"github.com/sundayprincedev/ledger/internal/jobs"
)

func buildIntents(operations []data.SeedOperation) ([]jobs.Intent, map[string]int) {
	intents := make([]jobs.Intent, 0, len(operations))
	skipped := make(map[string]int)

	for _, op := range operations {
		switch op.Type {
		case "transfer":
			intents = append(intents, jobs.Intent{
				Type:          jobs.OpSingleTransfer,
				FromAccountID: op.FromAccountID,
				ToAccountID:   op.ToAccountID,
				CurrencyCode:  op.CurrencyCode,
				Amount:        op.Amount,
				Note:          op.Note,
			})
		case "bulk_transfer":
			recipients := make([]jobs.Recipient, 0, len(op.Recipients))
			for _, r := range op.Recipients {
				recipients = append(recipients, jobs.Recipient{AccountID: r.AccountID, Weight: r.Weight})
			}
			intents = append(intents, jobs.Intent{
				Type:          jobs.OpBulkTransfer,
				FromAccountID: op.FromAccountID,
				CurrencyCode:  op.CurrencyCode,
				Amount:        op.Amount,
				Recipients:    recipients,
				Note:          op.Note,
			})
		default:
			skipped[op.Type]++
		}
	}

	return intents, skipped
}
