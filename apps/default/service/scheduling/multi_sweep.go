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

package scheduling

import (
	"context"
	"time"

	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/default/config"
)

// SweepFunc is a single RunOnce-style job.
type SweepFunc func(ctx context.Context) int

// MultiSweepRunner runs all due-work sweeps on one ticker (non-Cloud-Tasks / local).
type MultiSweepRunner struct {
	cfg    *config.Config
	sweeps []namedSweep
}

type namedSweep struct {
	name string
	fn   SweepFunc
}

// NewMultiSweepRunner builds a multi-sweep runner. Pass nil-safe RunOnce wrappers.
func NewMultiSweepRunner(cfg *config.Config, sweeps map[string]SweepFunc) *MultiSweepRunner {
	r := &MultiSweepRunner{cfg: cfg}
	// Stable order for observability.
	order := []string{
		"dispatch", "outbox", "retry", "timer", "timeout", "signal", "scope", "cron",
	}
	for _, name := range order {
		if fn, ok := sweeps[name]; ok && fn != nil {
			r.sweeps = append(r.sweeps, namedSweep{name: name, fn: fn})
		}
	}
	// Any extra jobs.
	for name, fn := range sweeps {
		if fn == nil {
			continue
		}
		found := false
		for _, s := range r.sweeps {
			if s.name == name {
				found = true
				break
			}
		}
		if !found {
			r.sweeps = append(r.sweeps, namedSweep{name: name, fn: fn})
		}
	}
	return r
}

// Start blocks until ctx is cancelled.
func (r *MultiSweepRunner) Start(ctx context.Context) {
	log := util.Log(ctx)
	interval := time.Duration(r.cfg.ReconcilerMultiSweepIntervalS) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}

	log.Info("multi-sweep reconciler started",
		"interval_seconds", int(interval.Seconds()),
		"jobs", len(r.sweeps),
	)

	// Run once immediately so cold start doesn't wait a full interval.
	r.runAll(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.runAll(ctx)
		case <-ctx.Done():
			log.Debug("multi-sweep reconciler stopped")
			return
		}
	}
}

func (r *MultiSweepRunner) runAll(ctx context.Context) {
	log := util.Log(ctx)
	for _, s := range r.sweeps {
		if ctx.Err() != nil {
			return
		}
		n := s.fn(ctx)
		if n > 0 {
			log.Debug("multi-sweep job", "job", s.name, "processed", n)
		}
	}
}

// RunJobsOnce runs the named jobs (empty = all). Returns total processed count.
func (r *MultiSweepRunner) RunJobsOnce(ctx context.Context, jobs []string) int {
	want := map[string]struct{}{}
	for _, j := range jobs {
		want[j] = struct{}{}
	}
	total := 0
	for _, s := range r.sweeps {
		if len(want) > 0 {
			if _, ok := want[s.name]; !ok {
				continue
			}
		}
		total += s.fn(ctx)
	}
	return total
}
