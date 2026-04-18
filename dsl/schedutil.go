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
	return time.Duration(int64(h.Sum64() % uint64(maxDur)))
}
