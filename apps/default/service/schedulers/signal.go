package schedulers

import (
	"context"
	"errors"
	"time"

	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/default/config"
	"github.com/antinvestor/service-trustage/apps/default/service/business"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
)

// SignalScheduler handles signal wait timeout expiry.
type SignalScheduler struct {
	waitRepo repository.WorkflowSignalWaitRepository
	engine   business.StateEngine
	cfg      *config.Config
}

// NewSignalScheduler creates a new SignalScheduler.
func NewSignalScheduler(
	waitRepo repository.WorkflowSignalWaitRepository,
	engine business.StateEngine,
	cfg *config.Config,
) *SignalScheduler {
	return &SignalScheduler{
		waitRepo: waitRepo,
		engine:   engine,
		cfg:      cfg,
	}
}

// Start begins the signal timeout sweep loop.
func (s *SignalScheduler) Start(ctx context.Context) {
	log := util.Log(ctx)
	interval := time.Duration(s.cfg.SignalIntervalSeconds) * time.Second

	log.Info("signal scheduler started", "interval_seconds", s.cfg.SignalIntervalSeconds)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			timedOut := s.RunOnce(ctx)
			if timedOut > 0 {
				log.Info("signal scheduler completed", "timed_out", timedOut)
			}
		case <-ctx.Done():
			log.Info("signal scheduler stopped")
			return
		}
	}
}

// RunOnce performs a single signal timeout sweep.
func (s *SignalScheduler) RunOnce(ctx context.Context) int {
	log := util.Log(ctx)
	now := time.Now()
	owner := "signal-scheduler"
	leaseUntil := now.Add(time.Duration(s.cfg.SignalClaimTTLSeconds) * time.Second)

	waits, err := s.waitRepo.ClaimTimedOut(ctx, now, s.cfg.SignalBatchSize, owner, leaseUntil)
	if err != nil {
		log.WithError(err).Error("signal scheduler: failed to claim timed out waits")
		return 0
	}

	timedOut := 0
	for _, wait := range waits {
		failErr := s.engine.FailWaitingExecution(
			ctx,
			wait.ExecutionID,
			models.ExecStatusTimedOut,
			&business.CommitError{
				Class:   "timeout",
				Code:    "signal_wait_timeout",
				Message: "signal wait timed out before a matching signal was delivered",
			},
		)
		if failErr != nil && !errors.Is(failErr, business.ErrStaleExecution) {
			log.WithError(failErr).Error("signal scheduler: failed to time out waiting execution",
				"wait_id", wait.ID,
				"execution_id", wait.ExecutionID,
			)
			_ = s.waitRepo.ReleaseClaim(ctx, wait.ID, owner)
			continue
		}

		if markErr := s.waitRepo.MarkTimedOutByOwner(ctx, wait.ID, owner, time.Now()); markErr != nil {
			log.WithError(markErr).Error("signal scheduler: failed to mark wait timed out",
				"wait_id", wait.ID,
				"execution_id", wait.ExecutionID,
			)
			continue
		}

		timedOut++
	}

	return timedOut
}
