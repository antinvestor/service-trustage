// Copyright 2023-2026 Ant Investor Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//nolint:testpackage // package-local scheduler constructor smoke tests
package schedulers

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/antinvestor/service-trustage/apps/default/config"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
)

func TestSchedulers_Constructors(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CleanupIntervalHours:           1,
		RetentionDays:                  1,
		DispatchBatchSize:              1,
		DispatchMaxBatchesPerSweep:     1,
		OutboxBatchSize:                1,
		OutboxMaxBatchesPerSweep:       1,
		OutboxClaimTTLSeconds:          1,
		RetryBatchSize:                 1,
		ScopeBatchSize:                 1,
		ScopeClaimTTLSeconds:           1,
		SignalBatchSize:                1,
		SignalClaimTTLSeconds:          1,
		TimeoutBatchSize:               1,
		DefaultExecutionTimeoutSeconds: 1,
		TimerBatchSize:                 1,
		TimerClaimTTLSeconds:           1,
	}

	metrics := telemetry.NewMetrics()

	require.NotNil(t, NewCleanupScheduler(nil, nil, cfg))
	require.NotNil(t, NewCronScheduler(nil, cfg, nil))
	require.NotNil(t, NewDispatchScheduler(nil, nil, nil, cfg, metrics))
	require.NotNil(t, NewOutboxScheduler(nil, nil, cfg, metrics))
	require.NotNil(t, NewRetryScheduler(nil, nil, cfg, metrics))
	require.NotNil(t, NewScopeScheduler(nil, nil, cfg))
	require.NotNil(t, NewSignalScheduler(nil, nil, cfg))
	require.NotNil(t, NewTimeoutScheduler(nil, nil, nil, nil, cfg, metrics))
	require.NotNil(t, NewTimerScheduler(nil, nil, cfg, metrics))
}
