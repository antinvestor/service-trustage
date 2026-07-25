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

package scheduling_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/scheduling"
)

type stubDefRepo struct {
	blob string
	err  error
}

func (s stubDefRepo) GetByNameAndVersion(_ context.Context, _ string, _ int) (*models.WorkflowDefinition, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &models.WorkflowDefinition{DSLBlob: s.blob}, nil
}

func TestTimeoutResolver_StepWins(t *testing.T) {
	t.Parallel()
	dslBlob := `{
  "version": "1.0",
  "name": "t",
  "timeout": "5m",
  "steps": [{"id": "s1", "type": "call", "timeout": "45s", "call": {"action": "log.entry", "input": {}}}]
}`
	r := scheduling.NewTimeoutResolver(stubDefRepo{blob: dslBlob}, 300*time.Second, 300*time.Second)
	now := time.Now()
	at, err := r.ResolveTimeoutAt(context.Background(), "t", 1, "s1", now)
	if err != nil {
		t.Fatal(err)
	}
	delta := at.Sub(now)
	if delta < 44*time.Second || delta > 46*time.Second {
		t.Fatalf("expected ~45s, got %v", delta)
	}
}

func TestTimeoutResolver_Cap(t *testing.T) {
	t.Parallel()
	dslBlob := `{
  "version": "1.0",
  "name": "t",
  "timeout": "1h",
  "steps": [{"id": "s1", "type": "call", "call": {"action": "log.entry", "input": {}}}]
}`
	r := scheduling.NewTimeoutResolver(stubDefRepo{blob: dslBlob}, 300*time.Second, 120*time.Second)
	now := time.Now()
	at, err := r.ResolveTimeoutAt(context.Background(), "t", 1, "s1", now)
	if err != nil {
		t.Fatal(err)
	}
	delta := at.Sub(now)
	if delta < 119*time.Second || delta > 121*time.Second {
		t.Fatalf("expected cap 120s, got %v", delta)
	}
}

func TestTimeoutResolver_DefaultOnMissingDef(t *testing.T) {
	t.Parallel()
	r := scheduling.NewTimeoutResolver(stubDefRepo{err: errors.New("missing")}, 90*time.Second, 300*time.Second)
	now := time.Now()
	at, err := r.ResolveTimeoutAt(context.Background(), "t", 1, "s1", now)
	if err != nil {
		t.Fatal(err)
	}
	if at.Sub(now) < 89*time.Second || at.Sub(now) > 91*time.Second {
		t.Fatalf("default 90s, got %v", at.Sub(now))
	}
}
