package schedulers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/antinvestor/service-trustage/apps/default/config"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
)

func TestSchedulers_ConstructorsAndStartStop(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CleanupIntervalHours:           1,
		RetentionDays:                  1,
		DispatchIntervalSeconds:        1,
		DispatchBatchSize:              1,
		DispatchMaxBatchesPerSweep:     1,
		OutboxIntervalSeconds:          1,
		OutboxBatchSize:                1,
		OutboxMaxBatchesPerSweep:       1,
		OutboxClaimTTLSeconds:          1,
		RetryIntervalSeconds:           1,
		RetryBatchSize:                 1,
		ScopeIntervalSeconds:           1,
		ScopeBatchSize:                 1,
		ScopeClaimTTLSeconds:           1,
		SignalIntervalSeconds:          1,
		SignalBatchSize:                1,
		SignalClaimTTLSeconds:          1,
		TimeoutIntervalSeconds:         1,
		TimeoutBatchSize:               1,
		DefaultExecutionTimeoutSeconds: 1,
		TimerIntervalSeconds:           1,
		TimerBatchSize:                 1,
		TimerClaimTTLSeconds:           1,
	}

	metrics := telemetry.NewMetrics()

	cases := []struct {
		name      string
		scheduler any
		start     func(context.Context)
	}{
		{
			name:      "cleanup",
			scheduler: NewCleanupScheduler(nil, nil, cfg),
			start:     NewCleanupScheduler(nil, nil, cfg).Start,
		},
		{
			name:      "cron",
			scheduler: NewCronScheduler(nil, nil, cfg),
			start:     NewCronScheduler(nil, nil, cfg).Start,
		},
		{
			name:      "dispatch",
			scheduler: NewDispatchScheduler(nil, nil, nil, cfg, metrics),
			start:     NewDispatchScheduler(nil, nil, nil, cfg, metrics).Start,
		},
		{
			name:      "outbox",
			scheduler: NewOutboxScheduler(nil, nil, cfg, metrics),
			start:     NewOutboxScheduler(nil, nil, cfg, metrics).Start,
		},
		{
			name:      "retry",
			scheduler: NewRetryScheduler(nil, nil, cfg, metrics),
			start:     NewRetryScheduler(nil, nil, cfg, metrics).Start,
		},
		{
			name:      "scope",
			scheduler: NewScopeScheduler(nil, nil, cfg),
			start:     NewScopeScheduler(nil, nil, cfg).Start,
		},
		{
			name:      "signal",
			scheduler: NewSignalScheduler(nil, nil, cfg),
			start:     NewSignalScheduler(nil, nil, cfg).Start,
		},
		{
			name:      "timeout",
			scheduler: NewTimeoutScheduler(nil, nil, nil, nil, cfg, metrics),
			start:     NewTimeoutScheduler(nil, nil, nil, nil, cfg, metrics).Start,
		},
		{
			name:      "timer",
			scheduler: NewTimerScheduler(nil, nil, cfg, metrics),
			start:     NewTimerScheduler(nil, nil, cfg, metrics).Start,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.NotNil(t, tc.scheduler)

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			done := make(chan struct{})
			go func() {
				defer close(done)
				tc.start(ctx)
			}()

			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("scheduler start did not stop after context cancellation")
			}
		})
	}
}
