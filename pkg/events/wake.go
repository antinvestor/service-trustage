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

import "time"

// Wake kinds for sched-* Frame queue messages.
const (
	WakeKindDispatch  = "dispatch"
	WakeKindRetry     = "retry"
	WakeKindTimer     = "timer"
	WakeKindTimeout   = "timeout"
	WakeKindSignal    = "signal"
	WakeKindReconcile = "reconcile"
	WakeKindCron      = "cron"
	WakeKindCleanup   = "cleanup"
	WakeKindOutbox    = "outbox"
	WakeKindScope     = "scope"
)

// WakeMessage is the payload for delayed wakes and external scheduler triggers.
type WakeMessage struct {
	Kind        string    `json:"kind"`
	ExecutionID string    `json:"execution_id,omitempty"`
	TimerID     string    `json:"timer_id,omitempty"`
	WaitID      string    `json:"wait_id,omitempty"`
	ScopeID     string    `json:"scope_id,omitempty"`
	EventID     string    `json:"event_id,omitempty"`
	DueAt       time.Time `json:"due_at,omitempty"`
	// Jobs lists reconcile job names when Kind is reconcile (empty = all standard jobs).
	Jobs []string `json:"jobs,omitempty"`
}
