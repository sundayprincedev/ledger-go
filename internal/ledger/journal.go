package ledger

import "github.com/sundayprincedev/ledger/internal/journal"

type Journal interface {
	Append(payload []byte) error
	Recover() ([]journal.RawEntry, error)
	Close() error
	Reset()
	Poison()
}
