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

package config

import (
	"errors"
	"fmt"
	"strings"
)

// ServiceRole selects which surfaces a process instance exposes.
type ServiceRole string

const (
	RoleAPI        ServiceRole = "api"
	RoleWorker     ServiceRole = "worker"
	RoleReconciler ServiceRole = "reconciler"
	RoleAll        ServiceRole = "all"
)

// ProgressDriver selects how due-work is driven in-process.
type ProgressDriver string

const (
	ProgressLegacyTickers ProgressDriver = "legacy_tickers"
	ProgressMultiSweep    ProgressDriver = "multi_sweep"
	ProgressExternalOnly  ProgressDriver = "external_only"

	defaultTimeoutSeconds = 300
)

// ParseServiceRole normalizes and validates SERVICE_ROLE.
func ParseServiceRole(raw string) (ServiceRole, error) {
	r := ServiceRole(strings.ToLower(strings.TrimSpace(raw)))
	if r == "" {
		r = RoleAll
	}
	switch r {
	case RoleAPI, RoleWorker, RoleReconciler, RoleAll:
		return r, nil
	default:
		return "", fmt.Errorf("invalid SERVICE_ROLE %q (want api|worker|reconciler|all)", raw)
	}
}

// ParseProgressDriver normalizes PROGRESS_DRIVER.
func ParseProgressDriver(raw string) (ProgressDriver, error) {
	d := ProgressDriver(strings.ToLower(strings.TrimSpace(raw)))
	if d == "" {
		return ProgressMultiSweep, nil
	}
	switch d {
	case ProgressLegacyTickers, ProgressMultiSweep, ProgressExternalOnly:
		return d, nil
	default:
		return "", fmt.Errorf(
			"invalid PROGRESS_DRIVER %q (want legacy_tickers|multi_sweep|external_only)",
			raw,
		)
	}
}

// PublishesEventIngest reports whether the role may register the event-ingest publisher.
func (r ServiceRole) PublishesEventIngest() bool {
	return r == RoleAPI || r == RoleWorker || r == RoleReconciler || r == RoleAll
}

// PublishesExecDispatch reports whether the role may register the exec-dispatch publisher.
func (r ServiceRole) PublishesExecDispatch() bool {
	return r == RoleWorker || r == RoleReconciler || r == RoleAll
}

// SubscribesHotPath reports whether the role runs event-router / exec-worker subscribers.
func (r ServiceRole) SubscribesHotPath() bool {
	return r == RoleWorker || r == RoleAll
}

// ExposesAPI reports whether ConnectRPC + form/webhook handlers are mounted.
func (r ServiceRole) ExposesAPI(workerExposeAPI bool) bool {
	if r == RoleAPI || r == RoleAll {
		return true
	}
	return r == RoleWorker && workerExposeAPI
}

// ReconcileInWorker reports whether worker also hosts reconcile push handlers / progress loops.
func (c *Config) ReconcileInWorker() bool {
	return c.ServiceRoleReconcileInWorker
}

// SubscribesReconcile reports whether sched-reconcile/cron/cleanup (and multi-sweep) apply.
func (c *Config) SubscribesReconcile() bool {
	r := c.ParsedRole()
	return r == RoleReconciler || r == RoleAll || (r == RoleWorker && c.ReconcileInWorker())
}

// LegacyTickersAllowed is the design allowlist — never api alone.
func (c *Config) LegacyTickersAllowed() bool {
	r := c.ParsedRole()
	return r == RoleAll || r == RoleReconciler || (r == RoleWorker && c.ReconcileInWorker())
}

// ParsedRole returns the validated role (must call ValidateRoleAndProgress at startup).
func (c *Config) ParsedRole() ServiceRole {
	r, err := ParseServiceRole(c.ServiceRole)
	if err != nil {
		return RoleAll
	}
	return r
}

// ParsedProgressDriver returns the validated progress driver.
func (c *Config) ParsedProgressDriver() ProgressDriver {
	d, err := ParseProgressDriver(c.ProgressDriver)
	if err != nil {
		return ProgressMultiSweep
	}
	// ENABLE_LEGACY_TICKERS forces legacy when allowed and multi_sweep not explicit.
	if c.EnableLegacyTickers && d != ProgressMultiSweep {
		return ProgressLegacyTickers
	}
	return d
}

// ValidateRoleAndProgress enforces mutual exclusion and role allowlists.
// Call once after loading config; fatal on error.
func (c *Config) ValidateRoleAndProgress() error {
	role, err := ParseServiceRole(c.ServiceRole)
	if err != nil {
		return err
	}
	c.ServiceRole = string(role)

	driver, err := c.resolveProgressDriver()
	if err != nil {
		return err
	}
	c.ProgressDriver = string(driver)

	if roleErr := c.validateDriverForRole(role, driver); roleErr != nil {
		return roleErr
	}

	if c.EnableLegacyTickers && c.ParsedProgressDriver() == ProgressMultiSweep {
		return errors.New("cannot enable both legacy tickers and multi_sweep progress driver")
	}

	if c.CloudRunMaxStepSeconds <= 0 {
		c.CloudRunMaxStepSeconds = defaultTimeoutSeconds
	}
	if c.DefaultExecutionTimeoutSeconds <= 0 {
		c.DefaultExecutionTimeoutSeconds = defaultTimeoutSeconds
	}
	return nil
}

func (c *Config) resolveProgressDriver() (ProgressDriver, error) {
	driver, err := ParseProgressDriver(c.ProgressDriver)
	if err != nil {
		return "", err
	}
	if !c.EnableLegacyTickers {
		return driver, nil
	}
	// Explicit multi_sweep wins over legacy flag.
	if driver == ProgressMultiSweep &&
		strings.EqualFold(strings.TrimSpace(c.ProgressDriver), string(ProgressMultiSweep)) {
		c.EnableLegacyTickers = false
		return ProgressMultiSweep, nil
	}
	return ProgressLegacyTickers, nil
}

func (c *Config) validateDriverForRole(role ServiceRole, driver ProgressDriver) error {
	if driver != ProgressLegacyTickers && driver != ProgressMultiSweep {
		return nil
	}
	if role == RoleAPI {
		// API never runs progress loops — force external_only.
		c.ProgressDriver = string(ProgressExternalOnly)
		c.EnableLegacyTickers = false
		return nil
	}
	if driver == ProgressLegacyTickers && !c.LegacyTickersAllowed() {
		return fmt.Errorf("PROGRESS_DRIVER=legacy_tickers not allowed for SERVICE_ROLE=%s", role)
	}
	if driver == ProgressMultiSweep && role == RoleWorker && !c.ReconcileInWorker() {
		return errors.New(
			"PROGRESS_DRIVER=multi_sweep requires SERVICE_ROLE_RECONCILE_IN_WORKER=true for role worker",
		)
	}
	return nil
}

// ShouldRunProgressLoops is true when this process hosts multi-sweep or legacy tickers.
func (c *Config) ShouldRunProgressLoops() bool {
	if c.ParsedRole() == RoleAPI {
		return false
	}
	if !c.SubscribesReconcile() && c.ParsedRole() != RoleAll {
		return false
	}
	d := c.ParsedProgressDriver()
	return d == ProgressLegacyTickers || d == ProgressMultiSweep
}

// ShouldRunLegacyTickers is true when nine Start() loops should be started.
func (c *Config) ShouldRunLegacyTickers() bool {
	return c.ShouldRunProgressLoops() && c.ParsedProgressDriver() == ProgressLegacyTickers
}

// ShouldRunMultiSweep is true when a single multi-job loop should run.
func (c *Config) ShouldRunMultiSweep() bool {
	return c.ShouldRunProgressLoops() && c.ParsedProgressDriver() == ProgressMultiSweep
}
