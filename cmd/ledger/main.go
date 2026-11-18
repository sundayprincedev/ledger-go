package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sundayprincedev/ledger/internal/constants"
)

func main() {
	os.Exit(realMain())
}

func realMain() int {
	cfg := config{grace: constants.ShutdownGracePeriod}
	flag.StringVar(&cfg.seedPath, "seed", constants.DefaultSeedPath, "path to the seed file")
	flag.StringVar(&cfg.journalPath, "journal", constants.DefaultJournalPath, "path to the journal file")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		stop()
	}()

	if _, err := run(ctx, cfg, os.Stdout); err != nil {
		log.Println(err)
		return 1
	}
	return 0
}
