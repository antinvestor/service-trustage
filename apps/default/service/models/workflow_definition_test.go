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

package models_test

import (
	"testing"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

// TestTransitionTo_Idempotent verifies that transitioning a WorkflowDefinition
// to its current status is always a no-op. This matters for retry safety: if a
// two-transaction activation sequence crashes after Tx1 commits (workflow moves
// to ACTIVE) but before Tx2 completes, the retry will call
// TransitionTo(ACTIVE) on an already-ACTIVE record.  It must return nil so the
// retry can succeed without manual intervention.
func TestTransitionTo_Idempotent_DraftToDraft(t *testing.T) {
	t.Parallel()

	w := &models.WorkflowDefinition{}
	w.Status = models.WorkflowStatusDraft

	if err := w.TransitionTo(models.WorkflowStatusDraft); err != nil {
		t.Fatalf("TransitionTo(draft→draft) = %v, want nil", err)
	}

	if w.Status != models.WorkflowStatusDraft {
		t.Fatalf("status = %q, want %q", w.Status, models.WorkflowStatusDraft)
	}
}

func TestTransitionTo_Idempotent_ActiveToActive(t *testing.T) {
	t.Parallel()

	w := &models.WorkflowDefinition{}
	w.Status = models.WorkflowStatusActive

	if err := w.TransitionTo(models.WorkflowStatusActive); err != nil {
		t.Fatalf("TransitionTo(active→active) = %v, want nil", err)
	}

	if w.Status != models.WorkflowStatusActive {
		t.Fatalf("status = %q, want %q", w.Status, models.WorkflowStatusActive)
	}
}

func TestTransitionTo_Idempotent_ArchivedToArchived(t *testing.T) {
	t.Parallel()

	w := &models.WorkflowDefinition{}
	w.Status = models.WorkflowStatusArchived

	if err := w.TransitionTo(models.WorkflowStatusArchived); err != nil {
		t.Fatalf("TransitionTo(archived→archived) = %v, want nil", err)
	}

	if w.Status != models.WorkflowStatusArchived {
		t.Fatalf("status = %q, want %q", w.Status, models.WorkflowStatusArchived)
	}
}

// TestTransitionTo_ValidTransitions verifies that all normal forward transitions work.
func TestTransitionTo_ValidTransitions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		from models.WorkflowDefinitionStatus
		to   models.WorkflowDefinitionStatus
	}{
		{"draft→active", models.WorkflowStatusDraft, models.WorkflowStatusActive},
		{"draft→archived", models.WorkflowStatusDraft, models.WorkflowStatusArchived},
		{"active→archived", models.WorkflowStatusActive, models.WorkflowStatusArchived},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			w := &models.WorkflowDefinition{}
			w.Status = tc.from

			if err := w.TransitionTo(tc.to); err != nil {
				t.Fatalf("TransitionTo(%s→%s) = %v, want nil", tc.from, tc.to, err)
			}

			if w.Status != tc.to {
				t.Fatalf("status = %q, want %q", w.Status, tc.to)
			}
		})
	}
}

// TestTransitionTo_InvalidTransitions verifies that illegal transitions are rejected.
func TestTransitionTo_InvalidTransitions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		from models.WorkflowDefinitionStatus
		to   models.WorkflowDefinitionStatus
	}{
		{"active→draft", models.WorkflowStatusActive, models.WorkflowStatusDraft},
		{"archived→active", models.WorkflowStatusArchived, models.WorkflowStatusActive},
		{"archived→draft", models.WorkflowStatusArchived, models.WorkflowStatusDraft},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			w := &models.WorkflowDefinition{}
			w.Status = tc.from

			if err := w.TransitionTo(tc.to); err == nil {
				t.Fatalf("TransitionTo(%s→%s) = nil, want error", tc.from, tc.to)
			}
		})
	}
}
