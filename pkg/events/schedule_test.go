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

package events

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScheduleFiredPayload_JSONShape(t *testing.T) {
	p := ScheduleFiredPayload{
		ScheduleID: "sched-1", ScheduleName: "nightly",
		FiredAt: "2026-04-18T00:00:00Z",
		Input:   map[string]any{"amount": 100.0},
	}
	raw, err := json.Marshal(p)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	require.Equal(t, "sched-1", m["schedule_id"])
	require.Equal(t, "nightly", m["schedule_name"])
	require.Equal(t, "2026-04-18T00:00:00Z", m["fired_at"])
	require.Equal(t, map[string]any{"amount": 100.0}, m["input"])
}

func TestScheduleFiredPayload_OmitsInputWhenNil(t *testing.T) {
	p := ScheduleFiredPayload{ScheduleID: "s", ScheduleName: "n", FiredAt: "2026-04-18T00:00:00Z"}
	raw, err := json.Marshal(p)
	require.NoError(t, err)
	require.NotContains(t, string(raw), `"input"`)
}

func TestBuildScheduleFiredPayload_SystemFieldsWin(t *testing.T) {
	userInput := map[string]any{
		"schedule_id":   "HIJACK",
		"schedule_name": "HIJACK",
		"fired_at":      "1970-01-01T00:00:00Z",
		"safe":          "stays",
	}
	p := BuildScheduleFiredPayload("real-id", "real-name", "2026-04-18T00:00:00Z", userInput)

	require.Equal(t, "real-id", p.ScheduleID)
	require.Equal(t, "real-name", p.ScheduleName)
	require.Equal(t, "2026-04-18T00:00:00Z", p.FiredAt)
	require.Equal(t, userInput, p.Input)
}

func TestScheduleFiredType(t *testing.T) {
	require.Equal(t, "schedule.fired", ScheduleFiredType)
}

func TestScheduleFiredPayload_ToJSON(t *testing.T) {
	p := ScheduleFiredPayload{ScheduleID: "s", ScheduleName: "n", FiredAt: "2026-04-18T00:00:00Z"}
	raw, err := p.ToJSON()
	require.NoError(t, err)
	require.Contains(t, raw, `"schedule_id":"s"`)
}
