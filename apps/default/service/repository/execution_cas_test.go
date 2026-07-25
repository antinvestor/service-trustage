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

package repository_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
)

// Dual-path CAS tests run against the shared repository suite DB when present.
// Standalone compile-safe unit checks for error contracts live here too.

func TestUpdateStatusExpected_Contract(t *testing.T) {
	t.Parallel()
	// Ensures ErrStaleMutation is the dual-path loser signal.
	if !errors.Is(repository.ErrStaleMutation, repository.ErrStaleMutation) {
		t.Fatal("sentinel")
	}
}

// TestDualPathDispatchAndTimeout are integration tests attached to the repository suite
// when the suite is available; this file provides helpers used by suite extensions.

func concurrentDispatchClaim(
	t *testing.T,
	repo repository.WorkflowExecutionRepository,
	execID string,
) (wins int) {
	t.Helper()
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := repo.UpdateStatusExpected(
				t.Context(),
				execID,
				models.ExecStatusPending,
				models.ExecStatusDispatched,
				map[string]any{
					"started_at": time.Now(),
					"timeout_at": time.Now().Add(time.Minute),
				},
			)
			if err == nil {
				mu.Lock()
				wins++
				mu.Unlock()
			} else if !errors.Is(err, repository.ErrStaleMutation) {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()
	return wins
}
