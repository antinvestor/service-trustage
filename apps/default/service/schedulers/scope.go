package schedulers

import (
	"context"
	"time"

	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/default/config"
	"github.com/antinvestor/service-trustage/apps/default/service/business"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
)

// ScopeScheduler reconciles running branch scopes and resumes/fails their parent executions.
type ScopeScheduler struct {
	scopeRepo repository.WorkflowScopeRunRepository
	engine    business.StateEngine
	cfg       *config.Config
}

// NewScopeScheduler creates a new ScopeScheduler.
func NewScopeScheduler(
	scopeRepo repository.WorkflowScopeRunRepository,
	engine business.StateEngine,
	cfg *config.Config,
) *ScopeScheduler {
	return &ScopeScheduler{
		scopeRepo: scopeRepo,
		engine:    engine,
		cfg:       cfg,
	}
}

// Start begins the scope reconciliation loop.
func (s *ScopeScheduler) Start(ctx context.Context) {
	log := util.Log(ctx)
	interval := time.Duration(s.cfg.ScopeIntervalSeconds) * time.Second

	log.Debug("scope scheduler started", "interval_seconds", s.cfg.ScopeIntervalSeconds)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			reconciled := s.RunOnce(ctx)
			if reconciled > 0 {
				log.Debug("scope scheduler completed", "reconciled", reconciled)
			}
		case <-ctx.Done():
			log.Debug("scope scheduler stopped")
			return
		}
	}
}

// RunOnce performs a single scope reconciliation sweep.
func (s *ScopeScheduler) RunOnce(ctx context.Context) int {
	log := util.Log(ctx)
	owner := "scope-scheduler"
	leaseUntil := time.Now().Add(time.Duration(s.cfg.ScopeClaimTTLSeconds) * time.Second)

	scopes, err := s.scopeRepo.ClaimRunning(ctx, s.cfg.ScopeBatchSize, owner, leaseUntil)
	if err != nil {
		log.WithError(err).Error("scope scheduler: failed to claim scopes")
		return 0
	}

	reconciled := 0
	for _, scope := range scopes {
		if reconcileErr := s.engine.ReconcileBranchScope(ctx, scope.ID); reconcileErr != nil {
			log.WithError(reconcileErr).Error("scope scheduler: reconcile failed", "scope_id", scope.ID)
			_ = s.scopeRepo.ReleaseClaim(ctx, scope.ID, owner)
			continue
		}

		_ = s.scopeRepo.ReleaseClaim(ctx, scope.ID, owner)
		reconciled++
	}

	return reconciled
}
