package dsl

import (
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
		return CronSchedule{}, fmt.Errorf("cron expression is empty")
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
