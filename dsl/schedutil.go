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

package dsl

import (
	"hash/fnv"
	"time"
)

// CronMaxJitter caps per-schedule jitter so spread is <= one poll interval.
const CronMaxJitter = 30 * time.Second

const cronJitterPeriodDivisor = 10

// JitterFor returns a deterministic per-schedule offset in the range
// [0, min(period/10, CronMaxJitter)). Stable across restarts.
func JitterFor(scheduleID string, cronSched CronSchedule, nominal time.Time) time.Duration {
	following := cronSched.Next(nominal)
	period := following.Sub(nominal)
	if period <= 0 {
		return 0
	}

	maxDur := period / cronJitterPeriodDivisor
	if maxDur > CronMaxJitter {
		maxDur = CronMaxJitter
	}
	if maxDur <= 0 {
		return 0
	}

	h := fnv.New64a()
	_, _ = h.Write([]byte(scheduleID))
	//nolint:gosec // maxDur is always in [1, 30s]; modulo result is always < maxDur, safe to convert
	return time.Duration(int64(h.Sum64() % uint64(maxDur)))
}
