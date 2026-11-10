package ledger

import "github.com/sundayprincedev/ledger/internal/journal"

type Journal interface {
	Append(payload []byte) error
	Recover() ([]journal.RawEntry, error)
	Close() error
}

// var _ Journal = (*journal.FileJournal)(nil) Move to main later
