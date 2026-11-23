package main

import (
	"context"
	"errors"
	"io"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sundayprincedev/ledger/internal/data"
	apperrors "github.com/sundayprincedev/ledger/internal/errors"
	"github.com/sundayprincedev/ledger/internal/jobs"
	"github.com/sundayprincedev/ledger/internal/journal"
	"github.com/sundayprincedev/ledger/internal/ledger"
)

type config struct {
	seedPath    string
	journalPath string
	grace       time.Duration
}

type tally struct {
	succeeded      int
	alreadyDone    int
	failed         int
	outcomePending int
}

type summary struct {
	replay  jobs.ReplayResult
	sent    int
	tally   tally
	skipped map[string]int
	forced  bool
}

func run(ctx context.Context, cfg config, out io.Writer) (summary, error) {
	logger := log.New(out, "", 0)

	seed, err := data.Load(cfg.seedPath)
	if err != nil {
		return summary{}, err
	}

	store, err := buildStore(seed.Accounts)
	if err != nil {
		return summary{}, err
	}

	intents, skipped := buildIntents(seed.Operations)
	logger.Printf("loaded %d accounts and %d operations: %d intents, skipped %s",
		len(seed.Accounts), len(seed.Operations), len(intents), describeSkipped(skipped))

	registry := ledger.NewIDRegistry()
	balances := buildBalances(seed.Accounts)

	replay, err := jobs.StartReadPipeline(ctx, cfg.journalPath, registry, balances)
	if err != nil {
		return summary{}, err
	}
	logger.Printf("replayed %d journal entries: %d registered, %d duplicates",
		replay.Recovered, replay.Registered, len(replay.Duplicates))
	for _, duplicate := range replay.Duplicates {
		logger.Printf("warning  %v", duplicate)
	}

	file := journal.Open(cfg.journalPath)
	defer func() {
		if err := file.Close(); err != nil {
			logger.Printf("closing the journal: %v", err)
		}
	}()

	workCtx, cancelWork := context.WithCancel(context.WithoutCancel(ctx))
	defer cancelWork()

	worker := jobs.NewWriteWorker(workCtx)

	jobCh := make(chan *jobs.Job)
	errCh := make(chan error, 16)
	resultsCh := make(chan jobs.JobResult, 16)

	worker.Start(1, jobCh, errCh, resultsCh, worker.WriteWork(registry, store, balances, file))
	collected := collect(errCh, resultsCh, logger)

	sentCh := make(chan int, 1)
	go func() {
		sentCh <- feed(ctx, intents, jobCh)
	}()

	done := make(chan struct{})
	go func() {
		worker.Wg.Wait()
		close(done)
	}()

	forced := drain(ctx, done, cfg.grace, cancelWork)
	if forced {
		logger.Printf("grace period of %s expired: in-flight work was cancelled", cfg.grace)
	}

	close(errCh)
	close(resultsCh)

	result := summary{replay: replay, sent: <-sentCh, tally: collected(), skipped: skipped, forced: forced}
	for _, account := range seed.Accounts {
		if now := balances.Get(account.AccountID, account.CurrencyCode); now != account.Balance {
			logger.Printf("balance  %s %s %d -> %d", account.AccountID, account.CurrencyCode, account.Balance, now)
		}
	}
	logger.Printf("sent %d: %d ok, %d already done, %d failed, %d outcome pending",
		result.sent, result.tally.succeeded, result.tally.alreadyDone, result.tally.failed, result.tally.outcomePending)

	return result, nil
}

func feed(ctx context.Context, intents []jobs.Intent, jobCh chan<- *jobs.Job) int {
	defer close(jobCh)

	sent := 0
	for _, intent := range intents {
		if ctx.Err() != nil {
			return sent
		}
		select {
		case <-ctx.Done():
			return sent
		case jobCh <- jobs.NewWriteJob(intent, 0):
			sent++
		}
	}
	return sent
}

func drain(ctx context.Context, done <-chan struct{}, grace time.Duration, cancelWork context.CancelFunc) bool {
	select {
	case <-done:
		return false
	case <-ctx.Done():
	}

	timer := time.NewTimer(grace)
	defer timer.Stop()

	select {
	case <-done:
		return false
	case <-timer.C:
		cancelWork()
		<-done
		return true
	}
}

func collect(errCh <-chan error, resultsCh <-chan jobs.JobResult, logger *log.Logger) func() tally {
	var wg sync.WaitGroup
	var t tally

	wg.Add(2)
	go func() {
		defer wg.Done()
		for err := range errCh {
			logger.Printf("error    %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		for result := range resultsCh {
			switch {
			case result.Success && result.Ack != nil && result.Ack.Status == jobs.AckAlreadyCompleted:
				t.alreadyDone++
				logger.Printf("done     %s (already written, re-acked)", shortID(result.JobID))
			case result.Success:
				t.succeeded++
				logger.Printf("ok       %s", shortID(result.JobID))
			case errors.Is(result.Err, apperrors.ErrOutcomeUnknown):
				t.outcomePending++
				logger.Printf("pending  %s", shortID(result.JobID))
			default:
				t.failed++
				logger.Printf("failed   %s", shortID(result.JobID))
			}
		}
	}()

	return func() tally {
		wg.Wait()
		return t
	}
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func describeSkipped(skipped map[string]int) string {
	if len(skipped) == 0 {
		return "nothing"
	}

	types := make([]string, 0, len(skipped))
	for opType := range skipped {
		types = append(types, opType)
	}
	sort.Strings(types)

	var b strings.Builder
	for i, opType := range types {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(opType)
		b.WriteString(" x")
		b.WriteString(strconv.Itoa(skipped[opType]))
	}
	return b.String()
}
