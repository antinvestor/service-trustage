package dsl_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/antinvestor/service-trustage/dsl"
)

func TestJitterFor_Deterministic(t *testing.T) {
	sched, err := dsl.ParseCron("*/5 * * * *")
	require.NoError(t, err)

	base := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	nominal := sched.Next(base)

	j1 := dsl.JitterFor("s-1", sched, nominal)
	j2 := dsl.JitterFor("s-1", sched, nominal)
	require.Equal(t, j1, j2)
}

func TestJitterFor_RespectsCap(t *testing.T) {
	sched, err := dsl.ParseCron("*/5 * * * *")
	require.NoError(t, err)

	base := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	nominal := sched.Next(base)

	for i := range 100 {
		j := dsl.JitterFor(fmt.Sprintf("s-%d", i), sched, nominal)
		require.True(
			t,
			j >= 0 && j < dsl.CronMaxJitter,
			"jitter %v out of [0, %v)",
			j,
			dsl.CronMaxJitter,
		)
	}
}
