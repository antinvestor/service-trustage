package dsl

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestJitterFor_Deterministic(t *testing.T) {
	sched, err := ParseCron("*/5 * * * *")
	require.NoError(t, err)

	base := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	nominal := sched.Next(base)

	require.Equal(t, JitterFor("s-1", sched, nominal), JitterFor("s-1", sched, nominal))
}

func TestJitterFor_RespectsCap(t *testing.T) {
	sched, err := ParseCron("*/5 * * * *")
	require.NoError(t, err)

	base := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	nominal := sched.Next(base)

	for i := 0; i < 100; i++ {
		j := JitterFor(fmt.Sprintf("s-%d", i), sched, nominal)
		require.True(t, j >= 0 && j < CronMaxJitter, "jitter %v out of [0, %v)", j, CronMaxJitter)
	}
}
