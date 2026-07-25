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

package config_test

import (
	"testing"

	"github.com/antinvestor/service-trustage/apps/default/config"
)

func TestParseServiceRole(t *testing.T) {
	t.Parallel()
	r, err := config.ParseServiceRole("API")
	if err != nil || r != config.RoleAPI {
		t.Fatalf("got %q %v", r, err)
	}
	if _, parseErr := config.ParseServiceRole("nope"); parseErr == nil {
		t.Fatal("expected error")
	}
}

func TestValidateRoleAndProgress_APIForcesExternalOnly(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		ServiceRole:                    "api",
		ProgressDriver:                 "multi_sweep",
		CloudRunMaxStepSeconds:         300,
		DefaultExecutionTimeoutSeconds: 300,
	}
	if err := cfg.ValidateRoleAndProgress(); err != nil {
		t.Fatal(err)
	}
	if cfg.ParsedProgressDriver() != config.ProgressExternalOnly {
		t.Fatalf("want external_only, got %s", cfg.ParsedProgressDriver())
	}
	if cfg.ShouldRunProgressLoops() {
		t.Fatal("api must not run progress loops")
	}
}

func TestValidateRoleAndProgress_WorkerMultiSweepNeedsReconcile(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		ServiceRole:                    "worker",
		ServiceRoleReconcileInWorker:   false,
		ProgressDriver:                 "multi_sweep",
		CloudRunMaxStepSeconds:         300,
		DefaultExecutionTimeoutSeconds: 300,
	}
	if err := cfg.ValidateRoleAndProgress(); err == nil {
		t.Fatal("expected error for worker multi_sweep without reconcile-in-worker")
	}
}

func TestLegacyTickersAllowed(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{ServiceRole: "worker", ServiceRoleReconcileInWorker: true}
	_ = cfg.ValidateRoleAndProgress()
	if !cfg.LegacyTickersAllowed() {
		t.Fatal("worker+reconcile should allow legacy")
	}
	cfg2 := &config.Config{ServiceRole: "api"}
	_ = cfg2.ValidateRoleAndProgress()
	if cfg2.LegacyTickersAllowed() {
		t.Fatal("api must not allow legacy")
	}
}

func TestPublisherMatrix(t *testing.T) {
	t.Parallel()
	if config.RoleAPI.PublishesExecDispatch() {
		t.Fatal("api must not publish exec-dispatch")
	}
	if !config.RoleWorker.PublishesExecDispatch() {
		t.Fatal("worker must publish exec-dispatch")
	}
	if !config.RoleAPI.PublishesEventIngest() {
		t.Fatal("api publishes event-ingest")
	}
}
