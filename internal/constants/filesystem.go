package constants

import "time"

const (
	DefaultSeedPath    = "internal/data/seed.json"
	DefaultJournalPath = "internal/data/ledger.log"

	ShutdownGracePeriod = 4 * time.Second
)
