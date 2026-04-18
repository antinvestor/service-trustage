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

// Package events defines canonical event payload shapes emitted by this service.
package events

import (
	"encoding/json"
	"fmt"
)

// ScheduleFiredType is the event_type value for schedule-fired events.
const ScheduleFiredType = "schedule.fired"

// ScheduleFiredPayload is the JSON shape of a schedule.fired event.
// System fields are typed (and therefore cannot be shadowed by user input).
// User-supplied data is namespaced under Input.
type ScheduleFiredPayload struct {
	ScheduleID   string         `json:"schedule_id"`
	ScheduleName string         `json:"schedule_name"`
	FiredAt      string         `json:"fired_at"` // RFC3339
	Input        map[string]any `json:"input,omitempty"`
}

// BuildScheduleFiredPayload constructs the payload with system fields set on
// the struct (immune to user shadowing) and user input preserved under Input.
func BuildScheduleFiredPayload(scheduleID, scheduleName, firedAtRFC3339 string, userInput map[string]any) ScheduleFiredPayload {
	return ScheduleFiredPayload{
		ScheduleID:   scheduleID,
		ScheduleName: scheduleName,
		FiredAt:      firedAtRFC3339,
		Input:        userInput,
	}
}

// ToJSON serialises the payload — convenience for callers that just need a string.
func (p ScheduleFiredPayload) ToJSON() (string, error) {
	raw, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("marshal schedule fired payload: %w", err)
	}
	return string(raw), nil
}
