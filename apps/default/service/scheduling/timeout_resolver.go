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

package scheduling

import (
	"context"
	"fmt"
	"time"

	"github.com/antinvestor/service-trustage/apps/default/service/business"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/dsl"
)

// DefinitionLoader loads workflow DSL by name/version (subset of definition repository).
type DefinitionLoader interface {
	GetByNameAndVersion(ctx context.Context, name string, version int) (*models.WorkflowDefinition, error)
}

// TimeoutResolver resolves execution deadlines from DSL step/workflow timeouts.
type TimeoutResolver struct {
	defRepo        DefinitionLoader
	defaultTimeout time.Duration
	maxTimeout     time.Duration
}

// NewTimeoutResolver creates a TimeoutResolver.
// defRepo may be repository.WorkflowDefinitionRepository or any DefinitionLoader.
func NewTimeoutResolver(
	defRepo DefinitionLoader,
	defaultTimeout, maxTimeout time.Duration,
) *TimeoutResolver {
	if defaultTimeout <= 0 {
		defaultTimeout = 300 * time.Second
	}
	if maxTimeout <= 0 {
		maxTimeout = 300 * time.Second
	}
	return &TimeoutResolver{
		defRepo:        defRepo,
		defaultTimeout: defaultTimeout,
		maxTimeout:     maxTimeout,
	}
}

// Compile-time check: concrete repos implement DefinitionLoader.
var _ DefinitionLoader = repository.WorkflowDefinitionRepository(nil)

// Ensure TimeoutResolver implements business.ExecutionTimeoutResolver.
var _ business.ExecutionTimeoutResolver = (*TimeoutResolver)(nil)

// ResolveTimeoutAt returns now + resolved duration (step → workflow → default), capped at max.
func (r *TimeoutResolver) ResolveTimeoutAt(
	ctx context.Context,
	workflowName string,
	workflowVersion int,
	state string,
	now time.Time,
) (time.Time, error) {
	dur := r.defaultTimeout

	def, err := r.defRepo.GetByNameAndVersion(ctx, workflowName, workflowVersion)
	if err != nil {
		// Fall back to default when definition missing.
		return now.Add(minDuration(dur, r.maxTimeout)), nil
	}

	spec, parseErr := dsl.Parse([]byte(def.DSLBlob))
	if parseErr != nil {
		return now.Add(minDuration(dur, r.maxTimeout)), nil
	}

	if step := dsl.FindStep(spec, state); step != nil && step.Timeout.Duration > 0 {
		dur = step.Timeout.Duration
	} else if spec.Timeout.Duration > 0 {
		dur = spec.Timeout.Duration
	}

	dur = minDuration(dur, r.maxTimeout)
	if dur <= 0 {
		return time.Time{}, fmt.Errorf("resolved timeout non-positive for %s/%d/%s", workflowName, workflowVersion, state)
	}
	return now.Add(dur), nil
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
