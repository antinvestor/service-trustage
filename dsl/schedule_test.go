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

package dsl_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/antinvestor/service-trustage/dsl"
)

func TestParseCron_ValidExpression(t *testing.T) {
	s, err := dsl.ParseCron("*/5 * * * *")
	require.NoError(t, err)
	require.Equal(t, "*/5 * * * *", s.Expr())
}

func TestParseCron_RejectsSixField(t *testing.T) {
	_, err := dsl.ParseCron("0 */5 * * * *")
	require.Error(t, err)
}

func TestParseCron_RejectsDescriptor(t *testing.T) {
	_, err := dsl.ParseCron("@hourly")
	require.Error(t, err)
}

func TestParseCron_RejectsEmpty(t *testing.T) {
	_, err := dsl.ParseCron("")
	require.Error(t, err)
}

func TestCronSchedule_NextMonotonic(t *testing.T) {
	s, err := dsl.ParseCron("*/10 * * * *")
	require.NoError(t, err)

	base := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	n1 := s.Next(base)
	n2 := s.Next(n1)
	n3 := s.Next(n2)

	require.True(t, n2.After(n1))
	require.True(t, n3.After(n2))
	require.Equal(t, 10*time.Minute, n2.Sub(n1))
}

func TestCronSchedule_NextInZone(t *testing.T) {
	s, err := dsl.ParseCron("0 2 * * *") // 02:00 daily
	require.NoError(t, err)

	// 05:30 UTC is 01:30 EDT on 2026-04-18. Next should be 02:00 EDT = 06:00 UTC.
	baseUTC := time.Date(2026, 4, 18, 5, 30, 0, 0, time.UTC)
	next, err := s.NextInZone(baseUTC, "America/New_York")
	require.NoError(t, err)
	require.Equal(t, 6, next.UTC().Hour())
}

func TestCronSchedule_NextInZone_InvalidZone(t *testing.T) {
	s, err := dsl.ParseCron("*/5 * * * *")
	require.NoError(t, err)

	_, err = s.NextInZone(time.Now(), "Not/A/Zone")
	require.Error(t, err)
}

func TestCronSchedule_NextInZone_UTCEquivalentToNext(t *testing.T) {
	s, err := dsl.ParseCron("0 2 * * *")
	require.NoError(t, err)

	baseUTC := time.Date(2026, 4, 18, 0, 0, 0, 0, time.UTC)
	want := s.Next(baseUTC)
	got, err := s.NextInZone(baseUTC, "UTC")
	require.NoError(t, err)
	require.True(t, want.Equal(got))
}
