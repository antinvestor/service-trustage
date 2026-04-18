package dsl

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseCron_ValidExpression(t *testing.T) {
	s, err := ParseCron("*/5 * * * *")
	require.NoError(t, err)
	require.Equal(t, "*/5 * * * *", s.Expr())
}

func TestParseCron_RejectsSixField(t *testing.T) {
	_, err := ParseCron("0 */5 * * * *")
	require.Error(t, err)
}

func TestParseCron_RejectsDescriptor(t *testing.T) {
	_, err := ParseCron("@hourly")
	require.Error(t, err)
}

func TestParseCron_RejectsEmpty(t *testing.T) {
	_, err := ParseCron("")
	require.Error(t, err)
}

func TestCronSchedule_NextMonotonic(t *testing.T) {
	s, err := ParseCron("*/10 * * * *")
	require.NoError(t, err)

	base := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	n1 := s.Next(base)
	n2 := s.Next(n1)
	n3 := s.Next(n2)

	require.True(t, n2.After(n1))
	require.True(t, n3.After(n2))
	require.Equal(t, 10*time.Minute, n2.Sub(n1))
}
