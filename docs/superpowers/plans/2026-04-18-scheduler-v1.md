# Scheduler v1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `trustage` schedules declarative (defined inside `WorkflowSpec`), cron-based, exactly-once under multi-pod fire, and scalable to millions of rows with zero additional configuration when pods are added.

**Architecture:** `WorkflowSpec.Schedules[]` (new DSL field) → materialised into `schedule_definitions` rows by `CreateWorkflow` inside the workflow-creation transaction. `ActivateWorkflow` flips the new version's schedules on and every previous version's schedules off in one tx. `CronScheduler` on every pod polls every 30 s via a new `ScheduleRepository.ClaimAndFireBatch` that wraps `FOR UPDATE SKIP LOCKED` + `event_log` insert + `next_fire_at` CAS update in a single transaction per schedule — restoring exactly-once. Deterministic per-schedule jitter flattens thundering herds; missed-fire policy skip-forwards.

**Tech Stack:** Go 1.26, Frame (`github.com/pitabwire/frame@v1.94.1`), GORM, `github.com/robfig/cron/v3` (new), ConnectRPC, Protobuf (buf).

**Working dir:** `/home/j/code/antinvestor/service-trustage`. Branch: `main`. Direct-to-main commits, user consented.

**Spec:** `docs/superpowers/specs/2026-04-18-scheduler-v1-design.md`.

---

## File Structure

**New:**
- `dsl/schedule.go` — `CronSchedule` type + `ParseCron` (infrastructure-free).
- `dsl/schedule_test.go` — parser & `Next()` unit tests.

**Modified — `dsl/`:**
- `dsl/types.go` — add `Schedules []*ScheduleSpec` to `WorkflowSpec`, add `ScheduleSpec` type.
- `dsl/validator.go` — add `validateSchedules`, wire into `Validate`.

**Modified — service-trustage code:**
- `apps/default/service/models/schedule.go` — add `JitterSeconds` column.
- `apps/default/service/repository/schedule.go` — remove `FindDue`/`UpdateFireTimes`; add `ClaimAndFireBatch`, `ListByWorkflow`, `SetActiveByWorkflow`, `Create` (kept).
- `apps/default/service/schedulers/cron.go` — rewrite to use `ClaimAndFireBatch` + cron parser + jitter + missed-fire policy.
- `apps/default/service/schedulers/scheduler_test.go` — cron expressions instead of durations.
- `apps/default/service/business/workflow.go` — materialise schedules on `CreateWorkflow`; flip on/off in `ActivateWorkflow`; return schedules on `GetWorkflow`.
- `apps/default/service/handlers/workflow.go` — populate `schedules[]` on `GetWorkflow` response.
- `apps/default/service/repository/schedule_test.go` — new, covers repo methods with testcontainers.

**Modified — proto:**
- `proto/workflow/v1/workflow.proto` — extend `GetWorkflowResponse` with `schedules[]`, add `ScheduleDefinition` message.

**Misc:**
- `go.mod`, `go.sum` — add `github.com/robfig/cron/v3`.
- `apps/default/service/repository/migrate.go` — no schema changes needed beyond GORM picking up the new column; the existing `idx_sd_due` partial index already covers the scan (verified during implementation).

---

## Task 1: Cron parser (dsl/schedule.go)

**Files:**
- Create: `dsl/schedule.go`
- Create: `dsl/schedule_test.go`
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add the cron dependency**

```bash
cd /home/j/code/antinvestor/service-trustage
go get github.com/robfig/cron/v3@latest
```
Expected: `go.mod` gains a `require` line for `github.com/robfig/cron/v3`.

- [ ] **Step 2: Write the failing tests**

Create `/home/j/code/antinvestor/service-trustage/dsl/schedule_test.go`:

```go
package dsl

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseCron_ValidExpression(t *testing.T) {
	s, err := ParseCron("*/5 * * * *")
	require.NoError(t, err)
	require.Equal(t, "*/5 * * * *", s.Expr())
}

func TestParseCron_RejectsSixField(t *testing.T) {
	_, err := ParseCron("0 */5 * * * *")
	require.Error(t, err)
}

func TestParseCron_RejectsDescriptor(t *testing.T) {
	_, err := ParseCron("@hourly")
	require.Error(t, err)
}

func TestParseCron_RejectsEmpty(t *testing.T) {
	_, err := ParseCron("")
	require.Error(t, err)
}

func TestCronSchedule_NextMonotonic(t *testing.T) {
	s, err := ParseCron("*/10 * * * *")
	require.NoError(t, err)

	base := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	n1 := s.Next(base)
	n2 := s.Next(n1)
	n3 := s.Next(n2)

	require.True(t, n2.After(n1))
	require.True(t, n3.After(n2))
	// */10 means minute 0, 10, 20, 30, 40, 50 — so n2-n1 should be 10m.
	require.Equal(t, 10*time.Minute, n2.Sub(n1))
}
```

- [ ] **Step 3: Run — expect compile error**

```bash
cd /home/j/code/antinvestor/service-trustage
go test ./dsl/ -run TestParseCron -race -v
```
Expected: `undefined: ParseCron` / `undefined: CronSchedule`.

- [ ] **Step 4: Implement**

Create `/home/j/code/antinvestor/service-trustage/dsl/schedule.go`:

```go
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
```

- [ ] **Step 5: Run — expect pass**

```bash
cd /home/j/code/antinvestor/service-trustage
go test ./dsl/ -run TestParseCron -race -v
go test ./dsl/ -run TestCronSchedule -race -v
```
Expected: all green.

- [ ] **Step 6: Commit**

```bash
cd /home/j/code/antinvestor/service-trustage
go mod tidy
git add go.mod go.sum dsl/schedule.go dsl/schedule_test.go
git commit -m "feat(dsl): add CronSchedule parser (robfig/cron/v3, 5-field strict)

Introduces dsl.ParseCron and CronSchedule.Next for real cron support
in workflow spec schedules. Strict 5-field parser — no seconds, no
descriptors. The 30s scheduler poll is the precision floor."
```

---

## Task 2: DSL — WorkflowSpec.Schedules + validation

**Files:**
- Modify: `dsl/types.go`
- Modify: `dsl/validator.go`
- Modify: `dsl/validator_test.go` (or add new test file)

- [ ] **Step 1: Write the failing test**

Append to `/home/j/code/antinvestor/service-trustage/dsl/validator_test.go` (create if absent — standard table-driven form):

```go
func TestValidateSchedules(t *testing.T) {
	cases := []struct {
		name      string
		schedules []*ScheduleSpec
		wantValid bool
		wantMsg   string
	}{
		{name: "nil slice is valid", schedules: nil, wantValid: true},
		{name: "empty slice is valid", schedules: []*ScheduleSpec{}, wantValid: true},
		{
			name: "valid single schedule",
			schedules: []*ScheduleSpec{
				{Name: "nightly", CronExpr: "0 2 * * *"},
			},
			wantValid: true,
		},
		{
			name: "empty name",
			schedules: []*ScheduleSpec{
				{Name: "", CronExpr: "*/5 * * * *"},
			},
			wantValid: false,
			wantMsg:   "name",
		},
		{
			name: "duplicate names",
			schedules: []*ScheduleSpec{
				{Name: "same", CronExpr: "*/5 * * * *"},
				{Name: "same", CronExpr: "0 2 * * *"},
			},
			wantValid: false,
			wantMsg:   "duplicate",
		},
		{
			name: "invalid cron",
			schedules: []*ScheduleSpec{
				{Name: "bad", CronExpr: "every 5 minutes"},
			},
			wantValid: false,
			wantMsg:   "cron",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec := &WorkflowSpec{
				Version:   "v1",
				Name:      "w",
				Steps:     []*StepSpec{{ID: "s", Type: StepTypeDelay, Delay: &DelaySpec{Duration: Duration{Duration: time.Second}}}},
				Schedules: tc.schedules,
			}
			result := Validate(spec)
			if tc.wantValid && !result.Valid() {
				t.Fatalf("expected valid, got errors: %v", result.Error())
			}
			if !tc.wantValid {
				if result.Valid() {
					t.Fatalf("expected invalid")
				}
				if tc.wantMsg != "" && !strings.Contains(result.Error().Error(), tc.wantMsg) {
					t.Fatalf("expected error containing %q, got %v", tc.wantMsg, result.Error())
				}
			}
		})
	}
}
```

If the file doesn't have `strings` / `time` imports yet, add them.

- [ ] **Step 2: Run — expect compile error**

```bash
cd /home/j/code/antinvestor/service-trustage
go test ./dsl/ -run TestValidateSchedules -race -v
```
Expected: `undefined: ScheduleSpec` / `unknown field Schedules`.

- [ ] **Step 3: Add `ScheduleSpec` + `WorkflowSpec.Schedules`**

Edit `/home/j/code/antinvestor/service-trustage/dsl/types.go`. Find the `WorkflowSpec` struct (lines 13–22) and add the `Schedules` field at the end:

```go
type WorkflowSpec struct {
	Version     string            `json:"version"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Input       map[string]string `json:"input,omitempty"`
	Config      map[string]any    `json:"config,omitempty"`
	Timeout     Duration          `json:"timeout,omitempty"`
	OnError     *ErrorPolicy      `json:"on_error,omitempty"`
	Steps       []*StepSpec       `json:"steps"`
	Schedules   []*ScheduleSpec   `json:"schedules,omitempty"`
}
```

Below `WorkflowSpec`, add:

```go
// ScheduleSpec declares a cron-triggered workflow schedule inside a WorkflowSpec.
// Schedules are materialised into schedule_definitions rows at CreateWorkflow time
// and follow the workflow's lifecycle — they activate when the workflow activates
// and deactivate when another version of the same workflow is activated.
type ScheduleSpec struct {
	Name         string         `json:"name"`
	CronExpr     string         `json:"cron_expr"`
	InputPayload map[string]any `json:"input_payload,omitempty"`
	// Active is an optional default. Nil means "active once the workflow is activated".
	// Explicitly false ships the schedule disabled even under an active workflow.
	Active *bool `json:"active,omitempty"`
}
```

- [ ] **Step 4: Add `validateSchedules` and wire it into `Validate`**

Edit `/home/j/code/antinvestor/service-trustage/dsl/validator.go`.

Append the new function at the end of the file:

```go
func validateSchedules(spec *WorkflowSpec, result *ValidationResult) {
	seen := make(map[string]struct{}, len(spec.Schedules))
	for i, sched := range spec.Schedules {
		if sched == nil {
			result.AddError(fmt.Sprintf("schedules[%d]: nil entry", i))
			continue
		}

		if strings.TrimSpace(sched.Name) == "" {
			result.AddError(fmt.Sprintf("schedules[%d]: name is required", i))
		} else if _, dup := seen[sched.Name]; dup {
			result.AddError(fmt.Sprintf("schedules[%d]: duplicate name %q", i, sched.Name))
		} else {
			seen[sched.Name] = struct{}{}
		}

		if _, err := ParseCron(sched.CronExpr); err != nil {
			result.AddError(fmt.Sprintf("schedules[%d] (%s): invalid cron expression: %s", i, sched.Name, err))
		}
	}
}
```

(If `strings` / `fmt` aren't imported in validator.go already, add them. Check with `head -12 dsl/validator.go`.)

Modify the `Validate` function at line 8 to call it:

```go
func Validate(spec *WorkflowSpec) *ValidationResult {
	result := &ValidationResult{}

	validateRequiredFields(spec, result)
	validateStepTypes(spec, result)
	validateUniqueIDs(spec, result)
	validateReferences(spec, result)
	validateDependencyGraph(spec, result)
	validateExpressions(spec, result)
	validateTemplates(spec, result)
	validateRetryPolicies(spec, result)
	validateTimeouts(spec, result)
	validateSchedules(spec, result)

	return result
}
```

- [ ] **Step 5: Run — expect pass**

```bash
cd /home/j/code/antinvestor/service-trustage
go test ./dsl/ -race
```
Expected: all green.

- [ ] **Step 6: Commit**

```bash
cd /home/j/code/antinvestor/service-trustage
git add dsl/types.go dsl/validator.go dsl/validator_test.go
git commit -m "feat(dsl): add WorkflowSpec.Schedules and validateSchedules

Schedules are declared inside a workflow spec. Validation rejects
empty names, duplicate names within a spec, and invalid cron
expressions (parsed via dsl.ParseCron)."
```

---

## Task 3: Model — add JitterSeconds column

**Files:**
- Modify: `apps/default/service/models/schedule.go`

- [ ] **Step 1: Add the column**

Edit `/home/j/code/antinvestor/service-trustage/apps/default/service/models/schedule.go`. Replace the struct body with:

```go
// ScheduleDefinition defines a cron schedule that triggers workflow events.
type ScheduleDefinition struct {
	data.BaseModel `gorm:"embedded"`

	Name            string     `gorm:"column:name;not null"`
	CronExpr        string     `gorm:"column:cron_expr;not null"`
	WorkflowName    string     `gorm:"column:workflow_name;not null"`
	WorkflowVersion int        `gorm:"column:workflow_version;not null"`
	InputPayload    string     `gorm:"column:input_payload;type:jsonb;default:'{}'"`
	Active          bool       `gorm:"column:active;not null;default:true"`
	NextFireAt      *time.Time `gorm:"column:next_fire_at"`
	LastFiredAt     *time.Time `gorm:"column:last_fired_at"`
	JitterSeconds   int        `gorm:"column:jitter_seconds;not null;default:0"`
}
```

(Only the last field is new — leave everything else untouched.)

- [ ] **Step 2: Compile**

```bash
cd /home/j/code/antinvestor/service-trustage
go build ./...
```
Expected: clean build.

- [ ] **Step 3: Verify AutoMigrate would pick it up (don't actually run — just read `migrate.go`)**

```bash
grep -n "ScheduleDefinition" /home/j/code/antinvestor/service-trustage/apps/default/service/repository/migrate.go
```
Expected: `&models.ScheduleDefinition{},` in the AutoMigrate block. GORM's AutoMigrate adds a nullable-safe column when a struct grows a new field. No index change needed — `idx_sd_due` (partial on `next_fire_at`) already covers the scan.

- [ ] **Step 4: Commit**

```bash
cd /home/j/code/antinvestor/service-trustage
git add apps/default/service/models/schedule.go
git commit -m "feat(schedule): add jitter_seconds column to ScheduleDefinition

Stores the deterministic per-schedule jitter (seconds) that was added
to next_fire_at. Purely observable — the jitter is already baked
into next_fire_at, this just surfaces it."
```

---

## Task 4: Repository — ClaimAndFireBatch with TDD exactly-once

**Files:**
- Modify: `apps/default/service/repository/schedule.go`
- Create: `apps/default/service/repository/schedule_test.go`

- [ ] **Step 1: Write the failing tests**

Create `/home/j/code/antinvestor/service-trustage/apps/default/service/repository/schedule_test.go`:

```go
//nolint:testpackage // package-local repository tests exercise unexported helpers intentionally.
package repository

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pitabwire/frame/datastore"
	datastoremanager "github.com/pitabwire/frame/datastore/manager"
	"github.com/pitabwire/frame/datastore/pool"
	"github.com/pitabwire/frame/frametests"
	"github.com/pitabwire/frame/frametests/definition"
	"github.com/pitabwire/frame/frametests/deps/testpostgres"
	"github.com/pitabwire/frame/security"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

type ScheduleRepoSuite struct {
	frametests.FrameBaseTestSuite

	dbPool   pool.Pool
	repo     ScheduleRepository
	eventRepo EventLogRepository
}

func TestScheduleRepoSuite(t *testing.T) {
	suite.Run(t, new(ScheduleRepoSuite))
}

func (s *ScheduleRepoSuite) SetupSuite() {
	s.InitResourceFunc = func(_ context.Context) []definition.TestResource {
		return []definition.TestResource{testpostgres.New()}
	}
	s.FrameBaseTestSuite.SetupSuite()

	ctx := context.Background()
	dsn := s.Resources()[0].GetDS(ctx)
	p := pool.NewPool(ctx)
	s.Require().NoError(p.AddConnection(
		ctx,
		pool.WithConnection(string(dsn), false),
		pool.WithPreparedStatements(false),
	))
	s.dbPool = p

	// Run AutoMigrate so schedule_definitions + event_log exist.
	manager, err := datastoremanager.NewManager(ctx)
	s.Require().NoError(err)
	manager.AddPool(ctx, datastore.DefaultPoolName, p)
	s.Require().NoError(Migrate(ctx, manager))

	s.repo = NewScheduleRepository(p)
	s.eventRepo = NewEventLogRepository(p)
}

func (s *ScheduleRepoSuite) SetupTest() {
	ctx := context.Background()
	s.Require().NoError(s.dbPool.DB(ctx, false).Exec(
		"TRUNCATE schedule_definitions, event_log CASCADE",
	).Error)
}

func (s *ScheduleRepoSuite) TearDownSuite() {
	if s.dbPool != nil {
		s.dbPool.Close(context.Background())
	}
	s.FrameBaseTestSuite.TearDownSuite()
}

func (s *ScheduleRepoSuite) tenantCtx() context.Context {
	claims := &security.AuthenticationClaims{
		TenantID:    "test-tenant",
		PartitionID: "test-partition",
	}
	claims.Subject = "test-user"
	return claims.ClaimsToContext(context.Background())
}

func (s *ScheduleRepoSuite) seedDueSchedules(ctx context.Context, n int) []*models.ScheduleDefinition {
	due := time.Now().UTC().Add(-time.Minute)
	out := make([]*models.ScheduleDefinition, 0, n)
	for i := 0; i < n; i++ {
		sched := &models.ScheduleDefinition{
			Name:            fmt.Sprintf("sched-%d", i),
			CronExpr:        "*/5 * * * *",
			WorkflowName:    "wf",
			WorkflowVersion: 1,
			InputPayload:    "{}",
			Active:          true,
			NextFireAt:      &due,
		}
		s.Require().NoError(s.repo.Create(ctx, sched))
		out = append(out, sched)
	}
	return out
}

// TestClaimAndFireBatch_ExactlyOnceUnderConcurrency seeds N due schedules,
// spawns 10 goroutines each looping ClaimAndFireBatch until the queue drains,
// and asserts every schedule fired exactly once.
func (s *ScheduleRepoSuite) TestClaimAndFireBatch_ExactlyOnceUnderConcurrency() {
	ctx := s.tenantCtx()
	const n = 50
	s.seedDueSchedules(ctx, n)

	var fired atomic.Int64
	var wg sync.WaitGroup
	for w := 0; w < 10; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				count, err := s.repo.ClaimAndFireBatch(ctx, time.Now().UTC(), 8,
					func(_ context.Context, tx *gorm.DB, sched *models.ScheduleDefinition) (*time.Time, int, error) {
						next := time.Now().Add(5 * time.Minute)
						// simulate event_log write inside the tx
						ev := &models.EventLog{
							EventType:      "schedule.fired",
							Source:         "schedule:" + sched.ID,
							IdempotencyKey: sched.ID + ":" + sched.NextFireAt.Format(time.RFC3339Nano),
							Payload:        "{}",
						}
						ev.CopyPartitionInfo(&sched.BaseModel)
						if err := tx.Create(ev).Error; err != nil {
							return nil, 0, err
						}
						return &next, 0, nil
					})
				s.Require().NoError(err)
				fired.Add(int64(count))
				if count == 0 {
					return
				}
			}
		}()
	}
	wg.Wait()

	s.Equal(int64(n), fired.Load(), "each schedule must fire exactly once across all workers")

	// Verify every schedule row has last_fired_at set and a future next_fire_at.
	var all []*models.ScheduleDefinition
	s.Require().NoError(s.dbPool.DB(ctx, false).Find(&all).Error)
	s.Len(all, n)
	for _, r := range all {
		s.NotNil(r.LastFiredAt, "schedule %s missing last_fired_at", r.Name)
		s.NotNil(r.NextFireAt, "schedule %s missing next_fire_at", r.Name)
		s.True(r.NextFireAt.After(time.Now()), "next_fire_at should be in the future")
	}

	// And event_log has exactly n schedule.fired rows.
	var evCount int64
	s.Require().NoError(s.dbPool.DB(ctx, false).Model(&models.EventLog{}).
		Where("event_type = ?", "schedule.fired").Count(&evCount).Error)
	s.Equal(int64(n), evCount)
}

// TestClaimAndFireBatch_IgnoresInactive proves the predicate filter is honoured.
func (s *ScheduleRepoSuite) TestClaimAndFireBatch_IgnoresInactive() {
	ctx := s.tenantCtx()
	due := time.Now().UTC().Add(-time.Minute)
	active := &models.ScheduleDefinition{Name: "a", CronExpr: "*/5 * * * *", WorkflowName: "wf", WorkflowVersion: 1,
		InputPayload: "{}", Active: true, NextFireAt: &due}
	inactive := &models.ScheduleDefinition{Name: "i", CronExpr: "*/5 * * * *", WorkflowName: "wf", WorkflowVersion: 1,
		InputPayload: "{}", Active: false, NextFireAt: &due}
	s.Require().NoError(s.repo.Create(ctx, active))
	s.Require().NoError(s.repo.Create(ctx, inactive))

	count, err := s.repo.ClaimAndFireBatch(ctx, time.Now().UTC(), 10,
		func(_ context.Context, _ *gorm.DB, _ *models.ScheduleDefinition) (*time.Time, int, error) {
			next := time.Now().Add(time.Hour)
			return &next, 0, nil
		})
	s.Require().NoError(err)
	s.Equal(1, count)
}
```

- [ ] **Step 2: Run — expect compile error**

```bash
cd /home/j/code/antinvestor/service-trustage
go test ./apps/default/service/repository/ -run TestScheduleRepoSuite -race -v
```
Expected: `ClaimAndFireBatch undefined` (and possibly `NewScheduleRepository` signature drift if other methods changed).

- [ ] **Step 3: Update the repository interface and implement `ClaimAndFireBatch`**

Edit `/home/j/code/antinvestor/service-trustage/apps/default/service/repository/schedule.go`. Replace the whole file content with:

```go
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/datastore/pool"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

// ScheduleRepository manages schedule_definitions persistence.
//
// The v1 surface is intentionally narrow: schedules are declared in workflow specs
// and materialised at CreateWorkflow; there is no schedule-level mutation RPC.
// Callers:
//   - business layer (Create, ListByWorkflow, SetActiveByWorkflow) — workflow lifecycle.
//   - CronScheduler (ClaimAndFireBatch) — the fire hot path.
type ScheduleRepository interface {
	Create(ctx context.Context, schedule *models.ScheduleDefinition) error

	ListByWorkflow(ctx context.Context, workflowName string, workflowVersion int) ([]*models.ScheduleDefinition, error)

	// SetActiveByWorkflow flips active on all non-deleted schedules for the given
	// (workflowName, workflowVersion) tuple. When activating (active=true), it also
	// seeds next_fire_at using the provided baseline; when deactivating, it clears
	// next_fire_at (avoids stale due rows lingering in the partial index).
	//
	// Must be called inside tx so the flip is atomic with the workflow status update.
	SetActiveByWorkflow(
		ctx context.Context,
		tx *gorm.DB,
		workflowName string,
		workflowVersion int,
		active bool,
		seedNextFireAt *time.Time,
		seedJitterSeconds int,
	) error

	// ClaimAndFireBatch scans for due schedules under one tx, invokes fireFn for each,
	// and commits atomically. fireFn receives the schedule and a DB handle bound to
	// the same tx so event_log inserts and next_fire_at updates participate in the
	// same transaction as the FOR UPDATE SKIP LOCKED row lock.
	//
	// fireFn returns the new next_fire_at and jitter_seconds. The repository persists
	// those onto the row before committing.
	//
	// Returns the number of schedules for which fireFn returned nil error.
	ClaimAndFireBatch(
		ctx context.Context,
		now time.Time,
		limit int,
		fireFn func(ctx context.Context, tx *gorm.DB, sched *models.ScheduleDefinition) (nextFire *time.Time, jitterSeconds int, err error),
	) (int, error)
}

type scheduleRepository struct {
	datastore.BaseRepository[*models.ScheduleDefinition]
}

// NewScheduleRepository creates a new ScheduleRepository.
func NewScheduleRepository(dbPool pool.Pool) ScheduleRepository {
	ctx := context.Background()

	return &scheduleRepository{
		BaseRepository: datastore.NewBaseRepository[*models.ScheduleDefinition](
			ctx,
			dbPool,
			nil,
			func() *models.ScheduleDefinition { return &models.ScheduleDefinition{} },
		),
	}
}

func (r *scheduleRepository) Create(ctx context.Context, schedule *models.ScheduleDefinition) error {
	return r.BaseRepository.Create(ctx, schedule)
}

func (r *scheduleRepository) ListByWorkflow(
	ctx context.Context,
	workflowName string,
	workflowVersion int,
) ([]*models.ScheduleDefinition, error) {
	db := r.BaseRepository.Pool().DB(ctx, false)

	var out []*models.ScheduleDefinition
	result := db.Where(
		"workflow_name = ? AND workflow_version = ? AND deleted_at IS NULL",
		workflowName, workflowVersion,
	).Order("name ASC").Find(&out)

	if result.Error != nil {
		return nil, fmt.Errorf("list schedules by workflow: %w", result.Error)
	}

	return out, nil
}

func (r *scheduleRepository) SetActiveByWorkflow(
	ctx context.Context,
	tx *gorm.DB,
	workflowName string,
	workflowVersion int,
	active bool,
	seedNextFireAt *time.Time,
	seedJitterSeconds int,
) error {
	if tx == nil {
		return fmt.Errorf("SetActiveByWorkflow requires a non-nil tx")
	}

	updates := map[string]any{
		"active":      active,
		"modified_at": time.Now().UTC(),
	}
	if active {
		updates["next_fire_at"] = seedNextFireAt
		updates["jitter_seconds"] = seedJitterSeconds
	} else {
		// Clear next_fire_at to keep the partial index small and avoid stale due rows
		// surviving a deactivate/reactivate cycle.
		updates["next_fire_at"] = nil
	}

	result := tx.Model(&models.ScheduleDefinition{}).
		Where("workflow_name = ? AND workflow_version = ? AND deleted_at IS NULL", workflowName, workflowVersion).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("set active by workflow: %w", result.Error)
	}

	return nil
}

func (r *scheduleRepository) ClaimAndFireBatch(
	ctx context.Context,
	now time.Time,
	limit int,
	fireFn func(ctx context.Context, tx *gorm.DB, sched *models.ScheduleDefinition) (*time.Time, int, error),
) (int, error) {
	db := r.BaseRepository.Pool().DB(ctx, false)

	fired := 0

	txErr := db.Transaction(func(tx *gorm.DB) error {
		var due []*models.ScheduleDefinition
		selectErr := tx.
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("active = ? AND deleted_at IS NULL AND next_fire_at IS NOT NULL AND next_fire_at <= ?", true, now).
			Order("next_fire_at ASC").
			Limit(limit).
			Find(&due).Error
		if selectErr != nil {
			return fmt.Errorf("claim due schedules: %w", selectErr)
		}

		for _, sched := range due {
			nextFire, jitterSeconds, fireErr := fireFn(ctx, tx, sched)
			if fireErr != nil {
				// Log in caller; continue batch — already-fired rows stay committed in the tx,
				// but we must not include this one in the update.
				continue
			}

			updateErr := tx.Model(&models.ScheduleDefinition{}).
				Where("id = ?", sched.ID).
				Updates(map[string]any{
					"last_fired_at":  now,
					"next_fire_at":   nextFire,
					"jitter_seconds": jitterSeconds,
					"modified_at":    now,
				}).Error
			if updateErr != nil {
				return fmt.Errorf("update fire times for %s: %w", sched.ID, updateErr)
			}

			fired++
		}

		return nil
	})

	if txErr != nil {
		return 0, txErr
	}

	return fired, nil
}
```

- [ ] **Step 4: Run — expect PASS**

```bash
cd /home/j/code/antinvestor/service-trustage
go test ./apps/default/service/repository/ -run TestScheduleRepoSuite -race -v
```
Expected: both `TestClaimAndFireBatch_*` tests green. If `undefined: EventLogRepository` or similar, it's already present; no change needed.

- [ ] **Step 5: Run the wider repository suite — regression check**

```bash
cd /home/j/code/antinvestor/service-trustage
go test ./apps/default/service/repository/... -race
```
Expected: all green *except* files that still reference the old `FindDue`/`UpdateFireTimes`. If any such file fails, note it — Task 5 fixes the scheduler caller.

- [ ] **Step 6: Commit**

```bash
cd /home/j/code/antinvestor/service-trustage
git add apps/default/service/repository/schedule.go apps/default/service/repository/schedule_test.go
git commit -m "feat(schedule-repo): add ClaimAndFireBatch, ListByWorkflow, SetActiveByWorkflow

ClaimAndFireBatch wraps SKIP LOCKED scan + event_log insert (via caller
fireFn) + next_fire_at update in one tx per row lock, restoring
exactly-once under multi-pod fire. Validated by a 10-worker
concurrency test.

ListByWorkflow feeds GetWorkflow's new schedules[]. SetActiveByWorkflow
is used by CreateWorkflow (materialise, active=false) and
ActivateWorkflow (flip on/off across versions). FindDue and
UpdateFireTimes are removed — remaining callers are fixed in the
next task."
```

---

## Task 5: Scheduler rewrite (cron + jitter + missed-fire) + remove old callers

**Files:**
- Modify: `apps/default/service/schedulers/cron.go`
- Modify: `apps/default/service/schedulers/scheduler_test.go`

- [ ] **Step 1: Rewrite `cron.go`**

Replace `/home/j/code/antinvestor/service-trustage/apps/default/service/schedulers/cron.go` with:

```go
package schedulers

import (
	"context"
	"encoding/json"
	"hash/fnv"
	"maps"
	"time"

	"github.com/pitabwire/util"
	"gorm.io/gorm"

	"github.com/antinvestor/service-trustage/apps/default/config"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/dsl"
)

const (
	cronSchedulerBatchSize    = 50
	cronCheckInterval         = 30 * time.Second
	cronMissedFireThreshold   = 5 * time.Minute
	cronMaxJitter             = 30 * time.Second
)

// CronScheduler fires events for schedule definitions whose next_fire_at has passed.
// Uses ScheduleRepository.ClaimAndFireBatch to ensure exactly-once fire under
// multi-pod deployment.
type CronScheduler struct {
	scheduleRepo repository.ScheduleRepository
	cfg          *config.Config
}

// NewCronScheduler creates a new CronScheduler.
func NewCronScheduler(
	scheduleRepo repository.ScheduleRepository,
	_ repository.EventLogRepository, // kept in signature for backwards compatibility with existing main.go wiring; unused now that fire goes through the tx.
	cfg *config.Config,
) *CronScheduler {
	return &CronScheduler{
		scheduleRepo: scheduleRepo,
		cfg:          cfg,
	}
}

// Start begins the cron scheduler loop. It blocks until context is cancelled.
func (s *CronScheduler) Start(ctx context.Context) {
	log := util.Log(ctx)
	log.Debug("cron scheduler started", "interval_seconds", int(cronCheckInterval.Seconds()))

	ticker := time.NewTicker(cronCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fired := s.RunOnce(ctx)
			if fired > 0 {
				log.Debug("cron scheduler completed", "fired", fired)
			}
		case <-ctx.Done():
			log.Debug("cron scheduler stopped")
			return
		}
	}
}

// RunOnce performs one transactional sweep for due schedules.
func (s *CronScheduler) RunOnce(ctx context.Context) int {
	log := util.Log(ctx)
	now := time.Now().UTC()

	fired, err := s.scheduleRepo.ClaimAndFireBatch(ctx, now, cronSchedulerBatchSize,
		func(ctx context.Context, tx *gorm.DB, sched *models.ScheduleDefinition) (*time.Time, int, error) {
			return fireOne(ctx, tx, sched, now)
		})
	if err != nil {
		log.WithError(err).Error("cron scheduler: batch fire failed")
		return 0
	}

	return fired
}

// fireOne emits the event_log row and returns the next fire time + jitter.
// Runs inside the tx that holds the FOR UPDATE SKIP LOCKED lock.
func fireOne(
	ctx context.Context,
	tx *gorm.DB,
	sched *models.ScheduleDefinition,
	now time.Time,
) (*time.Time, int, error) {
	log := util.Log(ctx)

	// Build event payload.
	payload := map[string]any{
		"schedule_id":   sched.ID,
		"schedule_name": sched.Name,
		"fired_at":      now.Format(time.RFC3339),
	}
	if sched.InputPayload != "" {
		var inputData map[string]any
		if err := json.Unmarshal([]byte(sched.InputPayload), &inputData); err == nil {
			maps.Copy(payload, inputData)
		}
	}
	payloadBytes, _ := json.Marshal(payload)

	eventLog := &models.EventLog{
		EventType:      "schedule.fired",
		Source:         "schedule:" + sched.ID,
		IdempotencyKey: sched.ID + ":" + now.Format(time.RFC3339Nano),
		Payload:        string(payloadBytes),
	}
	eventLog.CopyPartitionInfo(&sched.BaseModel)

	if err := tx.Create(eventLog).Error; err != nil {
		return nil, 0, err
	}

	// Compute next fire.
	cronSched, err := dsl.ParseCron(sched.CronExpr)
	if err != nil {
		// Invalid cron — log and park next_fire_at = nil so the row drops out of the partial index.
		log.WithError(err).Error("cron scheduler: invalid cron expression, parking schedule",
			"schedule_id", sched.ID, "cron_expr", sched.CronExpr)
		return nil, 0, nil
	}

	base := now
	if sched.NextFireAt != nil && now.Sub(*sched.NextFireAt) <= cronMissedFireThreshold {
		base = *sched.NextFireAt
	}

	nominal := cronSched.Next(base)
	jitter := jitterFor(sched.ID, cronSched, base, nominal)
	next := nominal.Add(jitter)

	return &next, int(jitter / time.Second), nil
}

// jitterFor returns a deterministic per-schedule offset to flatten herds.
// Capped at min(period/10, cronMaxJitter).
func jitterFor(scheduleID string, cronSched dsl.CronSchedule, base, nominal time.Time) time.Duration {
	following := cronSched.Next(nominal)
	period := following.Sub(nominal)
	if period <= 0 {
		return 0
	}

	cap := period / 10
	if cap > cronMaxJitter {
		cap = cronMaxJitter
	}
	if cap <= 0 {
		return 0
	}

	h := fnv.New64a()
	_, _ = h.Write([]byte(scheduleID))
	return time.Duration(int64(h.Sum64()) % int64(cap))
}
```

- [ ] **Step 2: Update `scheduler_test.go`**

Read the current test file:
```bash
cat /home/j/code/antinvestor/service-trustage/apps/default/service/schedulers/scheduler_test.go | head -80
```

If any test sets `CronExpr: "1h"` or similar duration-style values, update them to cron — e.g. `"*/1 * * * *"`. If any test references `computeNextFire`, remove it.

The minimal scheduler-level unit test file (replacing any duration-based content) should look like:

```go
package schedulers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/antinvestor/service-trustage/dsl"
)

func TestJitterFor_Deterministic(t *testing.T) {
	sched, err := dsl.ParseCron("*/5 * * * *")
	require.NoError(t, err)
	base := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	nominal := sched.Next(base)

	a := jitterFor("sched-1", sched, base, nominal)
	b := jitterFor("sched-1", sched, base, nominal)
	require.Equal(t, a, b, "jitter must be deterministic per schedule id")

	c := jitterFor("sched-2", sched, base, nominal)
	// Extremely unlikely to collide, but allow it — the point is determinism, not spread.
	_ = c
}

func TestJitterFor_RespectsCap(t *testing.T) {
	sched, err := dsl.ParseCron("*/5 * * * *") // 5-min period → cap = min(30s, 30s) = 30s.
	require.NoError(t, err)
	base := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	nominal := sched.Next(base)

	for i := 0; i < 50; i++ {
		j := jitterFor(fmt.Sprintf("s-%d", i), sched, base, nominal)
		require.True(t, j >= 0 && j < 30*time.Second, "jitter %v out of bounds", j)
	}
}
```

(Add `"fmt"` import in the test file if needed.)

- [ ] **Step 3: Run — expect PASS**

```bash
cd /home/j/code/antinvestor/service-trustage
go build ./apps/default/...
go test ./apps/default/service/schedulers/... -race
```
Expected: green. If cron.go signatures conflict with `main.go` wiring, go to step 4.

- [ ] **Step 4: Verify `main.go` wiring still compiles**

```bash
cd /home/j/code/antinvestor/service-trustage
go build ./apps/default/cmd/...
```
Expected: clean. `NewCronScheduler` still takes `(scheduleRepo, eventRepo, cfg)` for signature stability — the `eventRepo` arg is ignored but retained.

- [ ] **Step 5: Commit**

```bash
cd /home/j/code/antinvestor/service-trustage
git add apps/default/service/schedulers/cron.go apps/default/service/schedulers/scheduler_test.go
git commit -m "feat(scheduler): rewrite for transactional fire, real cron, jitter, missed-fire

fireSchedule now runs as a callback inside ScheduleRepository.ClaimAndFireBatch
so the event_log insert and next_fire_at update share the SKIP LOCKED
transaction — restoring exactly-once across pods.

Real cron via dsl.ParseCron replaces the old duration parser (dropped
per spec). Deterministic per-schedule jitter (cap = min(period/10, 30s))
flattens herds at common cron times. Missed-fire policy: if next_fire_at
is > 5 min in the past, fire once at now and skip-forward."
```

---

## Task 6: Business layer — materialise schedules on CreateWorkflow

**Files:**
- Modify: `apps/default/service/business/workflow.go`
- Modify: `apps/default/service/business/workflow_test.go` (extend)

- [ ] **Step 1: Read the current `CreateWorkflow` and `workflowBusiness` fields**

```bash
sed -n '40,110p' /home/j/code/antinvestor/service-trustage/apps/default/service/business/workflow.go
```

Note the struct's existing fields — we add `scheduleRepo repository.ScheduleRepository`.

- [ ] **Step 2: Write the failing test**

Append to `/home/j/code/antinvestor/service-trustage/apps/default/service/business/workflow_test.go`:

```go
func (s *WorkflowSuite) TestCreateWorkflow_MaterialisesSchedules() {
	ctx := s.tenantCtx()

	dslBlob := []byte(`{
		"version": "v1",
		"name": "w-sched",
		"steps": [{"id": "s", "type": "delay", "delay": {"duration": "1s"}}],
		"schedules": [
			{"name": "nightly", "cron_expr": "0 2 * * *"},
			{"name": "hourly",  "cron_expr": "0 * * * *"}
		]
	}`)

	def, err := s.biz.CreateWorkflow(ctx, dslBlob)
	s.Require().NoError(err)

	// Workflow in DRAFT.
	s.Equal(models.WorkflowStatusDraft, def.Status)

	// Schedules materialised, active=false, next_fire_at=nil.
	out, err := s.scheduleRepo.ListByWorkflow(ctx, def.Name, def.WorkflowVersion)
	s.Require().NoError(err)
	s.Len(out, 2)
	for _, sch := range out {
		s.False(sch.Active, "new schedule should be inactive")
		s.Nil(sch.NextFireAt, "new schedule should have no next_fire_at")
		s.Equal(def.Name, sch.WorkflowName)
		s.Equal(def.WorkflowVersion, sch.WorkflowVersion)
	}
}

func (s *WorkflowSuite) TestCreateWorkflow_InvalidScheduleCronRejected() {
	ctx := s.tenantCtx()

	dslBlob := []byte(`{
		"version": "v1",
		"name": "w-bad-sched",
		"steps": [{"id": "s", "type": "delay", "delay": {"duration": "1s"}}],
		"schedules": [{"name": "bad", "cron_expr": "not-a-cron"}]
	}`)

	_, err := s.biz.CreateWorkflow(ctx, dslBlob)
	s.Require().Error(err)
	s.Contains(err.Error(), "cron")
}
```

(If the test suite file doesn't already have a `scheduleRepo` field, Step 3 adds one alongside wiring.)

- [ ] **Step 3: Wire `scheduleRepo` into `workflowBusiness`**

Edit `/home/j/code/antinvestor/service-trustage/apps/default/service/business/workflow.go`:

```go
type workflowBusiness struct {
	defRepo      repository.WorkflowDefinitionRepository
	scheduleRepo repository.ScheduleRepository
	schemaReg    SchemaRegistry
}

// NewWorkflowBusiness creates a new WorkflowBusiness.
func NewWorkflowBusiness(
	defRepo repository.WorkflowDefinitionRepository,
	scheduleRepo repository.ScheduleRepository,
	schemaReg SchemaRegistry,
) WorkflowBusiness {
	return &workflowBusiness{
		defRepo:      defRepo,
		scheduleRepo: scheduleRepo,
		schemaReg:    schemaReg,
	}
}
```

- [ ] **Step 4: Materialise schedules inside `CreateWorkflow`**

In the same file, replace the persistence portion of `CreateWorkflow` (lines 95-104 in the pre-edit file):

```go
	if err = b.defRepo.Create(ctx, def); err != nil {
		return nil, fmt.Errorf("persist workflow: %w", err)
	}

	if schedErr := b.materialiseSchedules(ctx, def, spec); schedErr != nil {
		return nil, fmt.Errorf("materialise schedules: %w", schedErr)
	}

	log.Info("workflow created",
		"workflow_id", def.ID,
		"name", spec.Name,
	)

	return def, nil
}

func (b *workflowBusiness) materialiseSchedules(
	ctx context.Context,
	def *models.WorkflowDefinition,
	spec *dsl.WorkflowSpec,
) error {
	for _, sspec := range spec.Schedules {
		payloadJSON := "{}"
		if len(sspec.InputPayload) > 0 {
			raw, err := json.Marshal(sspec.InputPayload)
			if err != nil {
				return fmt.Errorf("marshal input_payload for schedule %s: %w", sspec.Name, err)
			}
			payloadJSON = string(raw)
		}

		sched := &models.ScheduleDefinition{
			Name:            sspec.Name,
			CronExpr:        sspec.CronExpr,
			WorkflowName:    def.Name,
			WorkflowVersion: def.WorkflowVersion,
			InputPayload:    payloadJSON,
			Active:          false, // DRAFT — activated by ActivateWorkflow.
			NextFireAt:      nil,
			JitterSeconds:   0,
		}
		sched.CopyPartitionInfo(&def.BaseModel)

		if err := b.scheduleRepo.Create(ctx, sched); err != nil {
			return fmt.Errorf("create schedule %s: %w", sspec.Name, err)
		}
	}

	return nil
}
```

(If `def.CopyPartitionInfo` isn't the method name, use whatever the existing business layer uses to copy tenant/partition — most commonly the `CopyPartitionInfo` method from `data.BaseModel`. Verify with `grep -n CopyPartitionInfo /home/j/code/antinvestor/service-trustage/apps/default/service/business/*.go`.)

- [ ] **Step 5: Fix `main.go` wiring — `NewWorkflowBusiness` signature changed**

Edit `/home/j/code/antinvestor/service-trustage/apps/default/cmd/main.go`. Find `workflowBiz := business.NewWorkflowBusiness(defRepo, schemaReg)` and change to:

```go
workflowBiz := business.NewWorkflowBusiness(defRepo, scheduleRepo, schemaReg)
```

`scheduleRepo` is already instantiated in main.go (grep to confirm location).

- [ ] **Step 6: Run — expect pass**

```bash
cd /home/j/code/antinvestor/service-trustage
go build ./...
go test ./apps/default/service/business/... -race -run TestWorkflowSuite
```
Expected: the two new tests pass; no other regressions.

- [ ] **Step 7: Commit**

```bash
cd /home/j/code/antinvestor/service-trustage
git add apps/default/service/business/workflow.go apps/default/service/business/workflow_test.go apps/default/cmd/main.go
git commit -m "feat(workflow): materialise spec schedules on CreateWorkflow

Schedules declared in WorkflowSpec.Schedules become schedule_definitions
rows inside the CreateWorkflow flow. New workflows are DRAFT, so the
rows land with active=false and next_fire_at=nil. ActivateWorkflow
will flip them on (next task)."
```

---

## Task 7: Business layer — ActivateWorkflow lifecycle

**Files:**
- Modify: `apps/default/service/business/workflow.go`
- Modify: `apps/default/service/repository/schedule.go` (tx plumbing if needed)
- Modify: `apps/default/service/business/workflow_test.go`

- [ ] **Step 1: Write the failing test**

Append to `workflow_test.go`:

```go
func (s *WorkflowSuite) TestActivateWorkflow_ActivatesSchedulesAndDeactivatesPrevious() {
	ctx := s.tenantCtx()

	// v1 with two schedules.
	v1DSL := []byte(`{
		"version": "v1",
		"name": "w-activate",
		"steps": [{"id": "s", "type": "delay", "delay": {"duration": "1s"}}],
		"schedules": [
			{"name": "a", "cron_expr": "*/5 * * * *"},
			{"name": "b", "cron_expr": "0 * * * *"}
		]
	}`)
	v1, err := s.biz.CreateWorkflow(ctx, v1DSL)
	s.Require().NoError(err)
	s.Require().NoError(s.biz.ActivateWorkflow(ctx, v1.ID))

	// Both v1 schedules must now be active with next_fire_at set.
	v1Scheds, err := s.scheduleRepo.ListByWorkflow(ctx, v1.Name, v1.WorkflowVersion)
	s.Require().NoError(err)
	s.Len(v1Scheds, 2)
	for _, sch := range v1Scheds {
		s.True(sch.Active, "schedule %s must be active after workflow activation", sch.Name)
		s.NotNil(sch.NextFireAt)
		s.True(sch.NextFireAt.After(time.Now()), "next_fire_at must be in the future")
	}

	// v2 of the same workflow.
	// (Workflow repo auto-increments WorkflowVersion on create for the same name — verify
	//  with a lookup after CreateWorkflow if needed.)
	v2DSL := []byte(`{
		"version": "v1",
		"name": "w-activate",
		"steps": [{"id": "s", "type": "delay", "delay": {"duration": "1s"}}],
		"schedules": [{"name": "only", "cron_expr": "*/10 * * * *"}]
	}`)
	v2, err := s.biz.CreateWorkflow(ctx, v2DSL)
	s.Require().NoError(err)
	s.NotEqual(v1.WorkflowVersion, v2.WorkflowVersion, "v2 must have a different workflow_version")

	s.Require().NoError(s.biz.ActivateWorkflow(ctx, v2.ID))

	// v2 schedule now active.
	v2Scheds, err := s.scheduleRepo.ListByWorkflow(ctx, v2.Name, v2.WorkflowVersion)
	s.Require().NoError(err)
	s.Len(v2Scheds, 1)
	s.True(v2Scheds[0].Active)
	s.NotNil(v2Scheds[0].NextFireAt)

	// v1 schedules must be deactivated.
	v1After, err := s.scheduleRepo.ListByWorkflow(ctx, v1.Name, v1.WorkflowVersion)
	s.Require().NoError(err)
	s.Len(v1After, 2)
	for _, sch := range v1After {
		s.False(sch.Active, "v1 schedule %s must be deactivated after v2 activation", sch.Name)
	}
}
```

- [ ] **Step 2: Extend `ScheduleRepository` interface with `Pool()` and a workflowVersion=-1 wildcard on `SetActiveByWorkflow`**

Edit `/home/j/code/antinvestor/service-trustage/apps/default/service/repository/schedule.go`:

```go
// In the interface block, add:
Pool() pool.Pool
```

The `SetActiveByWorkflow` query already covers the wildcard case via a simple tweak. Replace its body with:

```go
func (r *scheduleRepository) SetActiveByWorkflow(
	ctx context.Context,
	tx *gorm.DB,
	workflowName string,
	workflowVersion int,
	active bool,
	seedNextFireAt *time.Time,
	seedJitterSeconds int,
) error {
	if tx == nil {
		return fmt.Errorf("SetActiveByWorkflow requires a non-nil tx")
	}

	updates := map[string]any{
		"active":      active,
		"modified_at": time.Now().UTC(),
	}
	if active {
		updates["next_fire_at"] = seedNextFireAt
		updates["jitter_seconds"] = seedJitterSeconds
	} else {
		updates["next_fire_at"] = nil
	}

	query := tx.Model(&models.ScheduleDefinition{}).
		Where("workflow_name = ? AND deleted_at IS NULL", workflowName)
	if workflowVersion >= 0 {
		query = query.Where("workflow_version = ?", workflowVersion)
	}
	result := query.Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("set active by workflow: %w", result.Error)
	}
	return nil
}
```

Add the `Pool()` method on `scheduleRepository`:

```go
func (r *scheduleRepository) Pool() pool.Pool {
	return r.BaseRepository.Pool()
}
```

`pool` is already imported.

- [ ] **Step 3: Replace `ActivateWorkflow`**

Edit `/home/j/code/antinvestor/service-trustage/apps/default/service/business/workflow.go`. Replace `ActivateWorkflow` (around lines 252-267) with:

```go
func (b *workflowBusiness) ActivateWorkflow(ctx context.Context, id string) error {
	def, err := b.defRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrWorkflowNotFound, err)
	}

	if err = def.TransitionTo(models.WorkflowStatusActive); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidWorkflowStatus, err)
	}

	// Persist the workflow status change and flip schedule rows in one tx.
	db := b.scheduleRepo.Pool().DB(ctx, false)

	txErr := db.Transaction(func(tx *gorm.DB) error {
		if updateErr := tx.Save(def).Error; updateErr != nil {
			return fmt.Errorf("update workflow: %w", updateErr)
		}

		// Deactivate every previously-active version's schedules for the same workflow name.
		if deactErr := b.scheduleRepo.SetActiveByWorkflow(
			ctx, tx, def.Name, /*workflowVersion*/ -1, false, nil, 0,
		); deactErr != nil {
			return deactErr
		}

		// Compute initial next_fire_at per schedule: cronNext(now) + jitter.
		now := time.Now().UTC()
		myScheds, listErr := listByWorkflowTx(tx, def.Name, def.WorkflowVersion)
		if listErr != nil {
			return listErr
		}
		for _, sch := range myScheds {
			cronSched, parseErr := dsl.ParseCron(sch.CronExpr)
			if parseErr != nil {
				return fmt.Errorf("parse cron for schedule %s: %w", sch.Name, parseErr)
			}
			nominal := cronSched.Next(now)
			jitter := jitterForID(sch.ID, cronSched, nominal)
			next := nominal.Add(jitter)

			if updErr := tx.Model(&models.ScheduleDefinition{}).
				Where("id = ?", sch.ID).
				Updates(map[string]any{
					"active":         true,
					"next_fire_at":   &next,
					"jitter_seconds": int(jitter / time.Second),
					"modified_at":    now,
				}).Error; updErr != nil {
				return fmt.Errorf("activate schedule %s: %w", sch.ID, updErr)
			}
		}
		return nil
	})
	if txErr != nil {
		return txErr
	}

	return nil
}

// listByWorkflowTx is a tx-bound version of ScheduleRepository.ListByWorkflow.
// Used by ActivateWorkflow where the flip must be atomic with the workflow status change.
func listByWorkflowTx(tx *gorm.DB, workflowName string, workflowVersion int) ([]*models.ScheduleDefinition, error) {
	var out []*models.ScheduleDefinition
	res := tx.Where("workflow_name = ? AND workflow_version = ? AND deleted_at IS NULL",
		workflowName, workflowVersion).Find(&out)
	if res.Error != nil {
		return nil, fmt.Errorf("list schedules by workflow (tx): %w", res.Error)
	}
	return out, nil
}

// jitterForID is a duplicate of schedulers.jitterFor, scoped here to avoid importing schedulers from business.
// Deterministic per-schedule offset capped at min(period/10, 30s).
func jitterForID(scheduleID string, cronSched dsl.CronSchedule, nominal time.Time) time.Duration {
	const cronMaxJitter = 30 * time.Second
	following := cronSched.Next(nominal)
	period := following.Sub(nominal)
	if period <= 0 {
		return 0
	}
	cap := period / 10
	if cap > cronMaxJitter {
		cap = cronMaxJitter
	}
	if cap <= 0 {
		return 0
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(scheduleID))
	return time.Duration(int64(h.Sum64()) % int64(cap))
}
```

Imports: add to the `import` block at the top of `workflow.go`:
- `"hash/fnv"`
- `"time"`
- `"gorm.io/gorm"`
- `"github.com/pitabwire/frame/datastore/pool"`

(`dsl` is already imported.)

- [ ] **Step 4: Run — expect PASS**

```bash
cd /home/j/code/antinvestor/service-trustage
go build ./...
go test ./apps/default/service/business/... -run TestWorkflowSuite -race
```
Expected: the activation test passes; previous tests unchanged.

- [ ] **Step 5: Commit**

```bash
cd /home/j/code/antinvestor/service-trustage
git add apps/default/service/business/workflow.go apps/default/service/repository/schedule.go apps/default/service/business/workflow_test.go
git commit -m "feat(workflow): ActivateWorkflow flips schedules on; prior versions off

Activation now runs in one tx: workflow status update + deactivate
all other versions' schedules for the same workflow_name + activate
this version's schedules with seeded next_fire_at (cronNext(now) +
jitter). Matches the spec's lifecycle contract where activating a
new version stops the prior version's schedules atomically."
```

---

## Task 8: GetWorkflow returns schedules + proto extension + handler wiring

**Files:**
- Modify: `proto/workflow/v1/workflow.proto`
- Modify: `apps/default/service/business/workflow.go`
- Modify: `apps/default/service/handlers/workflow.go`
- Auto-regenerated: `gen/go/workflow/v1/*.go`

- [ ] **Step 1: Extend the proto**

Edit `/home/j/code/antinvestor/service-trustage/proto/workflow/v1/workflow.proto`. Add a new message and extend `GetWorkflowResponse`:

```proto
message ScheduleDefinition {
  string id = 1;
  string name = 2;
  string cron_expr = 3;
  string workflow_name = 4;
  int32 workflow_version = 5;
  bool active = 6;
  google.protobuf.Timestamp next_fire_at = 7;
  google.protobuf.Timestamp last_fired_at = 8;
  int32 jitter_seconds = 9;
  google.protobuf.Timestamp created_at = 10;
  google.protobuf.Timestamp updated_at = 11;
}

message GetWorkflowResponse {
  WorkflowDefinition workflow = 1;
  repeated ScheduleDefinition schedules = 2;
}
```

(Remove the old `GetWorkflowResponse` definition and replace with the one above.)

- [ ] **Step 2: Regenerate proto**

```bash
cd /home/j/code/antinvestor/service-trustage
make proto-gen
```
Expected: `gen/go/workflow/v1/workflow.pb.go` updated with new `ScheduleDefinition` type and `GetWorkflowResponse.Schedules` field.

- [ ] **Step 3: Business layer — `GetWorkflowWithSchedules`**

Edit `workflow.go`. Add to the interface:

```go
type WorkflowBusiness interface {
    // ... existing ...
    GetWorkflowWithSchedules(ctx context.Context, id string) (*models.WorkflowDefinition, []*models.ScheduleDefinition, error)
}
```

Implement:

```go
func (b *workflowBusiness) GetWorkflowWithSchedules(
	ctx context.Context,
	id string,
) (*models.WorkflowDefinition, []*models.ScheduleDefinition, error) {
	def, err := b.defRepo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", ErrWorkflowNotFound, err)
	}

	scheds, err := b.scheduleRepo.ListByWorkflow(ctx, def.Name, def.WorkflowVersion)
	if err != nil {
		return nil, nil, fmt.Errorf("list schedules: %w", err)
	}

	return def, scheds, nil
}
```

- [ ] **Step 4: Update the handler's `GetWorkflow`**

Edit `/home/j/code/antinvestor/service-trustage/apps/default/service/handlers/workflow.go`. In `GetWorkflow`, replace the `workflowBiz.GetWorkflow(ctx, id)` call with `GetWorkflowWithSchedules`, and include the schedules in the response:

```go
def, schedules, err := h.workflowBiz.GetWorkflowWithSchedules(ctx, id)
if err != nil { /* existing error handling */ }

resp := &workflowv1.GetWorkflowResponse{
    Workflow:  toAPIWorkflow(def),
    Schedules: toAPISchedules(schedules),
}
// write resp as before
```

Add a helper near the existing `toAPIWorkflow`:

```go
func toAPISchedules(in []*models.ScheduleDefinition) []*workflowv1.ScheduleDefinition {
	out := make([]*workflowv1.ScheduleDefinition, 0, len(in))
	for _, s := range in {
		api := &workflowv1.ScheduleDefinition{
			Id:              s.ID,
			Name:            s.Name,
			CronExpr:        s.CronExpr,
			WorkflowName:    s.WorkflowName,
			WorkflowVersion: int32(s.WorkflowVersion),
			Active:          s.Active,
			JitterSeconds:   int32(s.JitterSeconds),
			CreatedAt:       timestamppb.New(s.CreatedAt),
			UpdatedAt:       timestamppb.New(s.ModifiedAt),
		}
		if s.NextFireAt != nil {
			api.NextFireAt = timestamppb.New(*s.NextFireAt)
		}
		if s.LastFiredAt != nil {
			api.LastFiredAt = timestamppb.New(*s.LastFiredAt)
		}
		out = append(out, api)
	}
	return out
}
```

(If `timestamppb` isn't imported, add `"google.golang.org/protobuf/types/known/timestamppb"` to the import block.)

- [ ] **Step 5: Test**

Append to `workflow_test.go`:

```go
func (s *WorkflowSuite) TestGetWorkflowWithSchedules_ReturnsMaterialised() {
	ctx := s.tenantCtx()
	dslBlob := []byte(`{
		"version": "v1",
		"name": "w-get",
		"steps": [{"id": "s", "type": "delay", "delay": {"duration": "1s"}}],
		"schedules": [{"name": "x", "cron_expr": "*/5 * * * *"}]
	}`)
	def, err := s.biz.CreateWorkflow(ctx, dslBlob)
	s.Require().NoError(err)

	got, scheds, err := s.biz.GetWorkflowWithSchedules(ctx, def.ID)
	s.Require().NoError(err)
	s.Equal(def.ID, got.ID)
	s.Len(scheds, 1)
	s.Equal("x", scheds[0].Name)
}
```

- [ ] **Step 6: Run**

```bash
cd /home/j/code/antinvestor/service-trustage
go build ./...
go test ./apps/default/... -race
```
Expected: green.

- [ ] **Step 7: Commit**

```bash
cd /home/j/code/antinvestor/service-trustage
git add proto/workflow/v1/workflow.proto gen/go/workflow/v1/ apps/default/service/business/workflow.go apps/default/service/handlers/workflow.go apps/default/service/business/workflow_test.go
git commit -m "feat(workflow): GetWorkflow response includes materialised schedules

Adds a ScheduleDefinition proto message and a repeated schedules field
to GetWorkflowResponse. Business layer grows GetWorkflowWithSchedules;
handler populates the response using ListByWorkflow. ListWorkflows is
intentionally NOT extended — keeping list scans cheap."
```

---

## Task 9: Full test run, release v0.3.34, cluster verification

**Files:** none.

- [ ] **Step 1: Full suite**

```bash
cd /home/j/code/antinvestor/service-trustage
make tests
```
Expected: all packages green. If any unrelated package fails, STOP — do not tag a broken release.

- [ ] **Step 2: Lint**

```bash
cd /home/j/code/antinvestor/service-trustage
make lint
```
Expected: no new findings.

- [ ] **Step 3: Confirm queued commits**

```bash
cd /home/j/code/antinvestor/service-trustage
git log --oneline origin/main..HEAD
```
Expected: the commits from Tasks 1-8, plus any documentation commits.

- [ ] **Step 4: Push, tag, push tag**

```bash
cd /home/j/code/antinvestor/service-trustage
git push origin main
git tag -a v0.3.34 -m "Release v0.3.34: spec-driven cron scheduler"
git push origin v0.3.34
```

- [ ] **Step 5: Watch release workflow**

```bash
cd /home/j/code/antinvestor/service-trustage
RUN_ID=$(gh run list --workflow=Release --limit 5 --json databaseId,headBranch -q '.[] | select(.headBranch == "v0.3.34") | .databaseId' | head -1)
gh run watch "$RUN_ID" --exit-status
```
Expected: all three matrix images build (default, formstore, queue).

- [ ] **Step 6: Nudge Flux image-automation and wait**

```bash
kubectl -n trustage annotate imagerepository trustage reconcile.fluxcd.io/requestedAt="$(date -Iseconds)" --overwrite
for i in $(seq 1 6); do
  echo "--- attempt $i ---"
  kubectl -n trustage get imagepolicy trustage -o jsonpath='{.status.latestRef}'; echo
  kubectl -n trustage get imagepolicy trustage -o jsonpath='{.status.latestRef.tag}' 2>/dev/null | grep -q 'v0.3.34' && break
  sleep 30
done
```
Expected: `tag: v0.3.34` within 3 minutes.

- [ ] **Step 7: Observe the HelmRelease roll**

```bash
kubectl -n trustage get helmrelease trustage
kubectl -n trustage get jobs -l app.kubernetes.io/component=migration --sort-by=.metadata.creationTimestamp | tail
kubectl -n trustage get pods -l app.kubernetes.io/name=trustage
kubectl -n trustage logs deploy/trustage --tail=20
```
Expected: migration Job completes, new pods `1/1 Running`, logs show `starting trustage orchestrator` / `Build info version: v0.3.34`.

- [ ] **Step 8: End-to-end smoke test**

```bash
# Create a workflow carrying a schedule
TOKEN=... # tenant-scoped token
curl -sS -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  https://api.stawi.dev/trustage/workflow.v1.WorkflowService/CreateWorkflow \
  -X POST -d '{
    "dsl": {
      "version":"v1",
      "name":"schedsmoke",
      "steps":[{"id":"s","type":"delay","delay":{"duration":"1s"}}],
      "schedules":[{"name":"every-minute","cron_expr":"*/1 * * * *"}]
    }
  }' | jq '.workflow.id,.schedules'
```
Expected: 200, `workflow.id` present, `schedules[0].active = false`, `schedules[0].cron_expr = "*/1 * * * *"`.

```bash
# Activate
WID=<id from above>
curl -sS -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  https://api.stawi.dev/trustage/workflow.v1.WorkflowService/ActivateWorkflow \
  -X POST -d "{\"id\":\"$WID\"}" | jq
```
Expected: 200.

```bash
# Wait 90s, verify event_log accrual
sleep 90
kubectl -n trustage exec deploy/trustage -- sh -c 'PGPASSWORD=$DATABASE_PASSWORD psql -h pooler-ro.datastore.svc -U $DATABASE_USERNAME trustage -tAc "SELECT count(*) FROM event_log WHERE event_type=$$schedule.fired$$ AND source LIKE $$schedule:%$$;"'
```
Expected: ≥ 1.

- [ ] **Step 9: Sustained health**

```bash
for i in $(seq 1 8); do
  printf "[%02d] " "$i"; date -u '+%H:%M:%SZ'
  kubectl -n trustage get pods -l app.kubernetes.io/name=trustage --no-headers
  sleep 15
done
```
Expected: continuous `1/1 Running`, `RESTARTS = 0`.

- [ ] **Step 10: Declare done**

If steps 1-9 are all green, the scheduler v1 is live. Capture anything that looked flaky in a follow-up note.
