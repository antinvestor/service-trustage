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

package dsl

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

// standardCronParser is a strict 5-field parser: minute hour day-of-month month day-of-week.
// No seconds field, no descriptors (@hourly etc.) — the 30s scheduler poll interval is the
// precision floor, so sub-minute schedules are a foot-gun we don't want to offer.
var standardCronParser = cron.NewParser( //nolint:gochecknoglobals // parser is stateless and reusable.
	cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow,
)

// CronSchedule is a parsed, validated 5-field cron expression.
type CronSchedule struct {
	expr     string
	schedule cron.Schedule
}

// ParseCron parses a 5-field cron expression. Returns an error for 6-field inputs,
// descriptors, or any other form the standard parser rejects.
func ParseCron(expr string) (CronSchedule, error) {
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" {
		return CronSchedule{}, errors.New("cron expression is empty")
	}

	sched, err := standardCronParser.Parse(trimmed)
	if err != nil {
		return CronSchedule{}, fmt.Errorf("parse cron %q: %w", trimmed, err)
	}

	return CronSchedule{expr: trimmed, schedule: sched}, nil
}

// Expr returns the canonical cron expression this schedule was parsed from.
func (s CronSchedule) Expr() string { return s.expr }

// Next returns the first fire time strictly after `from` for this schedule.
func (s CronSchedule) Next(from time.Time) time.Time { return s.schedule.Next(from) }

// NextInZone returns the first fire time strictly after `from`, evaluated in
// the specified IANA timezone. "UTC" (or empty) preserves Next's behaviour.
// Returns an error if the zone is not loadable. Result is always in UTC.
func (s CronSchedule) NextInZone(from time.Time, zone string) (time.Time, error) {
	if zone == "" || zone == "UTC" {
		return s.schedule.Next(from.UTC()).UTC(), nil
	}

	loc, err := time.LoadLocation(zone)
	if err != nil {
		return time.Time{}, fmt.Errorf("load zone %q: %w", zone, err)
	}
	return s.schedule.Next(from.In(loc)).UTC(), nil
}
