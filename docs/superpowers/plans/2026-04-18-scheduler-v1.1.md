# Scheduler v1.1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Raise the cluster-wide fire ceiling from ~20/sec to 6 k+/sec via batched fire transactions; close all v1 correctness bugs; add per-schedule timezone and `ArchiveWorkflow`.

**Architecture:** One tx per sweep inside the repo (`SELECT FOR UPDATE SKIP LOCKED` + multi-row `INSERT event_log` + `UPDATE schedule_definitions FROM VALUES`). Business layer composes sequential single-table operations — no cross-repo transactions anywhere in business code. The fire path is the one documented cross-table tx exception and lives entirely inside `ScheduleRepository`.

**Tech Stack:** Go 1.26, Frame (`github.com/pitabwire/frame@v1.94.1`), GORM + pgx, `github.com/robfig/cron/v3`, ConnectRPC, `pkg/telemetry` (OpenTelemetry).

**Working dir:** `/home/j/code/antinvestor/service-trustage`. Branch: `main`. Direct-to-main commits.

**Spec:** `docs/superpowers/specs/2026-04-18-scheduler-v1.1-design.md`.

**Dependencies between tasks:**
- Tasks 1 – 3 independent.
- Task 4 depends on 1 + 2.
- Task 5 depends on 1 + 2 + 4.
- Task 6 independent.
- Task 7 depends on 4.
- Task 8 depends on 7.
- Task 9 depends on 5 + 7.
- Task 10 depends on 8 + 9.

---

## File Structure

**New:**
- `dsl/schedutil.go` + `dsl/schedutil_test.go`
- `pkg/events/schedule.go` + `pkg/events/schedule_test.go`

**Modified (DSL):**
- `dsl/types.go` — add `Timezone`, remove `Active`
- `dsl/schedule.go` — add `NextInZone`
- `dsl/validator.go` — validate timezone

**Modified (code):**
- `apps/default/config/config.go` — new env vars
- `apps/default/service/models/schedule.go` — `Timezone` column
- `apps/default/service/repository/migrate.go` — tighten `idx_sd_due`, add `idx_sd_workflow_unique`
- `apps/default/service/repository/schedule.go` — new methods + `ClaimAndFireBatch` sig change
- `apps/default/service/repository/schedule_test.go` — updated tests
- `apps/default/service/schedulers/cron.go` — `SchedulePlanFn` caller, metrics, configurable
- `apps/default/service/schedulers/scheduler_test.go`
- `apps/default/service/schedulers/outbox.go` — batch / concurrent publish
- `apps/default/service/business/workflow.go` — two-tx Create / Activate / Archive
- `apps/default/service/business/workflow_integration_test.go`
- `apps/default/service/handlers/workflow_connect.go` — `ArchiveWorkflow` handler
- `apps/default/service/handlers/connect_helpers.go` — timezone in schedule proto
- `apps/default/cmd/main.go` — dedicated scheduler pool, metrics injection
- `pkg/telemetry/metrics.go` — new span + metric

**Modified (proto):**
- `proto/workflow/v1/workflow.proto` — `timezone` on `ScheduleDefinition`, `ArchiveWorkflow` RPC

---

## Task 1: DSL — timezone, remove Active, extract JitterFor

**Files:**
- Modify: `dsl/types.go`, `dsl/schedule.go`, `dsl/validator.go`
- Create: `dsl/schedutil.go`, `dsl/schedutil_test.go`
- Modify (append): `dsl/schedule_test.go`

- [ ] **Step 1: Failing tests**

Create `/home/j/code/antinvestor/service-trustage/dsl/schedutil_test.go`:

```go
package dsl

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestJitterFor_Deterministic(t *testing.T) {
	sched, err := ParseCron("*/5 * * * *")
	require.NoError(t, err)

	base := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	nominal := sched.Next(base)

	require.Equal(t, JitterFor("s-1", sched, nominal), JitterFor("s-1", sched, nominal))
}

func TestJitterFor_RespectsCap(t *testing.T) {
	sched, err := ParseCron("*/5 * * * *")
	require.NoError(t, err)

	base := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	nominal := sched.Next(base)

	for i := 0; i < 100; i++ {
		j := JitterFor(fmt.Sprintf("s-%d", i), sched, nominal)
		require.True(t, j >= 0 && j < CronMaxJitter, "jitter %v out of [0, %v)", j, CronMaxJitter)
	}
}
```

Append to `/home/j/code/antinvestor/service-trustage/dsl/schedule_test.go`:

```go
func TestCronSchedule_NextInZone(t *testing.T) {
	s, err := ParseCron("0 2 * * *") // 02:00 daily
	require.NoError(t, err)

	// 01:30 EDT → 05:30 UTC. Next should be 02:00 EDT = 06:00 UTC.
	baseUTC := time.Date(2026, 4, 18, 5, 30, 0, 0, time.UTC)
	next, err := s.NextInZone(baseUTC, "America/New_York")
	require.NoError(t, err)
	require.Equal(t, 6, next.UTC().Hour())
}

func TestCronSchedule_NextInZone_InvalidZone(t *testing.T) {
	s, err := ParseCron("*/5 * * * *")
	require.NoError(t, err)

	_, err = s.NextInZone(time.Now(), "Not/A/Zone")
	require.Error(t, err)
}

func TestCronSchedule_NextInZone_UTCEquivalentToNext(t *testing.T) {
	s, err := ParseCron("0 2 * * *")
	require.NoError(t, err)

	baseUTC := time.Date(2026, 4, 18, 0, 0, 0, 0, time.UTC)
	want := s.Next(baseUTC)
	got, err := s.NextInZone(baseUTC, "UTC")
	require.NoError(t, err)
	require.True(t, want.Equal(got))
}
```

- [ ] **Step 2: Run — expect failures**

```bash
cd /home/j/code/antinvestor/service-trustage
go test ./dsl/ -run 'TestJitterFor|TestCronSchedule_NextInZone' -race -v 2>&1 | head -20
```
Expected: `undefined: JitterFor`, `undefined: NextInZone`, `undefined: CronMaxJitter`.

- [ ] **Step 3: Create `dsl/schedutil.go`**

```go
package dsl

import (
	"hash/fnv"
	"time"
)

// CronMaxJitter caps per-schedule jitter so spread is <= one poll interval.
const CronMaxJitter = 30 * time.Second

const cronJitterPeriodDivisor = 10

// JitterFor returns a deterministic per-schedule offset in the range
// [0, min(period/10, CronMaxJitter)). Stable across restarts.
func JitterFor(scheduleID string, cronSched CronSchedule, nominal time.Time) time.Duration {
	following := cronSched.Next(nominal)
	period := following.Sub(nominal)
	if period <= 0 {
		return 0
	}

	maxDur := period / cronJitterPeriodDivisor
	if maxDur > CronMaxJitter {
		maxDur = CronMaxJitter
	}
	if maxDur <= 0 {
		return 0
	}

	h := fnv.New64a()
	_, _ = h.Write([]byte(scheduleID))
	return time.Duration(int64(h.Sum64() % uint64(maxDur)))
}
```

- [ ] **Step 4: Extend `dsl/schedule.go` with `NextInZone`**

Append:

```go
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
```

- [ ] **Step 5: Update `dsl/types.go`**

Find `ScheduleSpec`, replace:

```go
// ScheduleSpec declares a cron-triggered workflow schedule inside a
// WorkflowSpec. Schedules follow the workflow's lifecycle — they activate
// when the workflow activates and deactivate on version switch or archive.
type ScheduleSpec struct {
	Name         string         `json:"name"`
	CronExpr     string         `json:"cron_expr"`
	Timezone     string         `json:"timezone,omitempty"` // IANA; default "UTC"
	InputPayload map[string]any `json:"input_payload,omitempty"`
}
```

`Active *bool` is removed.

- [ ] **Step 6: Update `dsl/validator.go`**

Ensure `"time"` is imported. Replace `validateSchedules`:

```go
func validateSchedules(spec *WorkflowSpec, result *ValidationResult) {
	seen := make(map[string]struct{}, len(spec.Schedules))
	for i, sched := range spec.Schedules {
		if sched == nil {
			result.AddError(ErrInvalidSchedule, fmt.Sprintf("schedules[%d]: nil entry", i))
			continue
		}
		if strings.TrimSpace(sched.Name) == "" {
			result.AddError(ErrInvalidSchedule, fmt.Sprintf("schedules[%d]: name is required", i))
		} else if _, dup := seen[sched.Name]; dup {
			result.AddError(ErrInvalidSchedule, fmt.Sprintf("schedules[%d]: duplicate name %q", i, sched.Name))
		} else {
			seen[sched.Name] = struct{}{}
		}
		if _, err := ParseCron(sched.CronExpr); err != nil {
			result.AddError(ErrInvalidSchedule, fmt.Sprintf("schedules[%d] (%s): invalid cron: %s", i, sched.Name, err))
		}
		if sched.Timezone != "" && sched.Timezone != "UTC" {
			if _, err := time.LoadLocation(sched.Timezone); err != nil {
				result.AddError(ErrInvalidSchedule, fmt.Sprintf("schedules[%d] (%s): invalid timezone %q: %s", i, sched.Name, sched.Timezone, err))
			}
		}
	}
}
```

- [ ] **Step 7: Run full DSL tests**

```bash
cd /home/j/code/antinvestor/service-trustage
go test ./dsl/ -race
```
Expected: green. Delete any pre-existing test touching `ScheduleSpec.Active` (e.g. assertion that `Active: ptr(true)` is persisted — v1 didn't honour it anyway).

- [ ] **Step 8: Commit**

```bash
git add dsl/schedutil.go dsl/schedutil_test.go dsl/schedule.go dsl/schedule_test.go dsl/types.go dsl/validator.go
git commit -m "feat(dsl): JitterFor, NextInZone, per-schedule timezone

- Extract JitterFor + CronMaxJitter into dsl/schedutil.go so scheduler
  + workflow business share one algorithm.
- CronSchedule.NextInZone evaluates cron in any IANA zone. Returns UTC.
- ScheduleSpec.Timezone (IANA, default UTC). ScheduleSpec.Active
  removed — DRAFT-until-activated lifecycle covers the only legitimate
  intent; v1 never honoured it."
```

---

## Task 2: pkg/events — typed `schedule.fired` payload

**Files:**
- Create: `pkg/events/schedule.go`, `pkg/events/schedule_test.go`

- [ ] **Step 1: Check for existing pkg/events**

```bash
ls /home/j/code/antinvestor/service-trustage/pkg/events/ 2>/dev/null && cat /home/j/code/antinvestor/service-trustage/pkg/events/*.go 2>/dev/null | head
```

- [ ] **Step 2: Write failing test**

Create `/home/j/code/antinvestor/service-trustage/pkg/events/schedule_test.go`:

```go
package events

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScheduleFiredPayload_JSONShape(t *testing.T) {
	p := ScheduleFiredPayload{
		ScheduleID: "sched-1", ScheduleName: "nightly",
		FiredAt: "2026-04-18T00:00:00Z",
		Input:   map[string]any{"amount": 100.0},
	}
	raw, err := json.Marshal(p)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	require.Equal(t, "sched-1", m["schedule_id"])
	require.Equal(t, "nightly", m["schedule_name"])
	require.Equal(t, "2026-04-18T00:00:00Z", m["fired_at"])
	require.Equal(t, map[string]any{"amount": 100.0}, m["input"])
}

func TestScheduleFiredPayload_OmitsInputWhenNil(t *testing.T) {
	p := ScheduleFiredPayload{ScheduleID: "s", ScheduleName: "n", FiredAt: "2026-04-18T00:00:00Z"}
	raw, err := json.Marshal(p)
	require.NoError(t, err)
	require.NotContains(t, string(raw), `"input"`)
}

func TestBuildScheduleFiredPayload_SystemFieldsWin(t *testing.T) {
	userInput := map[string]any{
		"schedule_id":   "HIJACK",
		"schedule_name": "HIJACK",
		"fired_at":      "1970-01-01T00:00:00Z",
		"safe":          "stays",
	}
	p := BuildScheduleFiredPayload("real-id", "real-name", "2026-04-18T00:00:00Z", userInput)

	require.Equal(t, "real-id", p.ScheduleID)
	require.Equal(t, "real-name", p.ScheduleName)
	require.Equal(t, "2026-04-18T00:00:00Z", p.FiredAt)
	require.Equal(t, userInput, p.Input)
}

func TestScheduleFiredType(t *testing.T) {
	require.Equal(t, "schedule.fired", ScheduleFiredType)
}
```

- [ ] **Step 3: Run — expect compile error, then PASS**

```bash
go test ./pkg/events/ -race -v 2>&1 | head
```
Expected: `undefined: ScheduleFiredPayload`.

- [ ] **Step 4: Create `pkg/events/schedule.go`**

```go
// Package events defines canonical event payload shapes.
package events

// ScheduleFiredType is the event_type value for schedule-fired events.
const ScheduleFiredType = "schedule.fired"

// ScheduleFiredPayload is the JSON shape of a schedule.fired event.
// System fields are typed (and therefore cannot be shadowed by user input).
// User-supplied data is namespaced under Input.
type ScheduleFiredPayload struct {
	ScheduleID   string         `json:"schedule_id"`
	ScheduleName string         `json:"schedule_name"`
	FiredAt      string         `json:"fired_at"` // RFC3339
	Input        map[string]any `json:"input,omitempty"`
}

// BuildScheduleFiredPayload constructs the payload with system fields set on
// the struct (immune to user shadowing) and user input preserved under Input.
func BuildScheduleFiredPayload(scheduleID, scheduleName, firedAtRFC3339 string, userInput map[string]any) ScheduleFiredPayload {
	return ScheduleFiredPayload{
		ScheduleID:   scheduleID,
		ScheduleName: scheduleName,
		FiredAt:      firedAtRFC3339,
		Input:        userInput,
	}
}
```

- [ ] **Step 5: Run — expect PASS**

```bash
go test ./pkg/events/ -race -v
```

- [ ] **Step 6: Commit**

```bash
git add pkg/events/schedule.go pkg/events/schedule_test.go
git commit -m "feat(events): typed ScheduleFiredPayload with Input namespacing

Fixes v1 audit finding: user-supplied input_payload could shadow
schedule_id/schedule_name/fired_at. With system fields on a typed
struct and user data under Input, key collisions are impossible."
```

---

## Task 3: Model + migrate — timezone column, index updates

**Files:**
- Modify: `apps/default/service/models/schedule.go`
- Modify: `apps/default/service/repository/migrate.go`

- [ ] **Step 1: Add `Timezone` field to the model**

Edit `/home/j/code/antinvestor/service-trustage/apps/default/service/models/schedule.go`:

```go
type ScheduleDefinition struct {
	data.BaseModel `gorm:"embedded"`

	Name            string     `gorm:"column:name;not null"`
	CronExpr        string     `gorm:"column:cron_expr;not null"`
	Timezone        string     `gorm:"column:timezone;not null;default:'UTC'"`
	WorkflowName    string     `gorm:"column:workflow_name;not null"`
	WorkflowVersion int        `gorm:"column:workflow_version;not null"`
	InputPayload    string     `gorm:"column:input_payload;type:jsonb;default:'{}'"`
	Active          bool       `gorm:"column:active;not null;default:false"`
	NextFireAt      *time.Time `gorm:"column:next_fire_at"`
	LastFiredAt     *time.Time `gorm:"column:last_fired_at"`
	JitterSeconds   int        `gorm:"column:jitter_seconds;not null;default:0"`
}

func (ScheduleDefinition) TableName() string { return "schedule_definitions" }
```

Only `Timezone` is new.

- [ ] **Step 2: Update index definitions in `migrate.go`**

Find the `scheduleDefinitionIndexModel` struct (around line 276) and the `migrationIndexes` entry for it. Replace:

```go
// In migrationIndexes():
{
    Model: &scheduleDefinitionIndexModel{},
    Names: []string{"idx_sd_tenant", "idx_sd_due", "idx_sd_workflow_unique"},
},
```

```go
// scheduleDefinitionIndexModel — index schema only.
type scheduleDefinitionIndexModel struct {
	TenantID        string    `gorm:"column:tenant_id;index:idx_sd_tenant,priority:1;index:idx_sd_workflow_unique,unique,where:deleted_at IS NULL,priority:1"`
	PartitionID     string    `gorm:"column:partition_id;index:idx_sd_tenant,priority:2;index:idx_sd_workflow_unique,unique,where:deleted_at IS NULL,priority:2"`
	WorkflowName    string    `gorm:"column:workflow_name;index:idx_sd_workflow_unique,unique,where:deleted_at IS NULL,priority:3"`
	WorkflowVersion int       `gorm:"column:workflow_version;index:idx_sd_workflow_unique,unique,where:deleted_at IS NULL,priority:4"`
	Name            string    `gorm:"column:name;index:idx_sd_workflow_unique,unique,where:deleted_at IS NULL,priority:5"`
	NextFireAt      time.Time `gorm:"column:next_fire_at;index:idx_sd_due,where:active = true AND deleted_at IS NULL AND next_fire_at IS NOT NULL,priority:1"`
}

func (scheduleDefinitionIndexModel) TableName() string { return "schedule_definitions" }
```

- [ ] **Step 3: Drop the v1 `idx_sd_due` so AutoMigrate rebuilds with the tighter predicate**

In `Migrate()`, after the `db.AutoMigrate(...)` call and BEFORE the `for _, indexDef := range migrationIndexes()` loop:

```go
	// v1.1 housekeeping: drop the v1 idx_sd_due so the tightened predicate
	// (WHERE active = true AND deleted_at IS NULL AND next_fire_at IS NOT NULL)
	// is picked up on next CreateIndex. One-time.
	if db.Migrator().HasIndex(&models.ScheduleDefinition{}, "idx_sd_due") {
		if dropErr := db.Migrator().DropIndex(&models.ScheduleDefinition{}, "idx_sd_due"); dropErr != nil {
			return fmt.Errorf("drop v1 idx_sd_due: %w", dropErr)
		}
	}
```

- [ ] **Step 4: Build + test**

```bash
cd /home/j/code/antinvestor/service-trustage
go build ./...
go test ./apps/default/service/repository/... -race -run TestScheduleRepoSuite -v | tail
```
Expected: green. (Fresh tables in tests mean the drop path isn't exercised; that's fine.)

- [ ] **Step 5: Commit**

```bash
git add apps/default/service/models/schedule.go apps/default/service/repository/migrate.go
git commit -m "feat(schedule): timezone column + tightened indexes

- Timezone varchar(64) NOT NULL DEFAULT 'UTC' — IANA zone for cron
  evaluation.
- idx_sd_due predicate gains \"next_fire_at IS NOT NULL\" so parked
  rows are out of the partial index. Migrate() drops the v1 index
  once; AutoMigrate recreates with the tight predicate.
- New idx_sd_workflow_unique over (tenant_id, partition_id,
  workflow_name, workflow_version, name) WHERE deleted_at IS NULL —
  makes schedule materialisation idempotent at the DB layer."
```

---

## Task 4: Repository — batched `ClaimAndFireBatch` + `CreateBatch` / `ActivateByWorkflow` / `DeactivateByWorkflow`

**Files:**
- Modify: `apps/default/service/repository/schedule.go` (substantial changes)
- Modify: `apps/default/service/repository/schedule_test.go` (extended)

- [ ] **Step 1: Write failing tests**

Replace `/home/j/code/antinvestor/service-trustage/apps/default/service/repository/schedule_test.go` with:

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

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/pkg/events"
)

type ScheduleRepoSuite struct {
	frametests.FrameBaseTestSuite

	dbPool pool.Pool
	repo   ScheduleRepository
}

func TestScheduleRepoSuite(t *testing.T) { suite.Run(t, new(ScheduleRepoSuite)) }

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

	manager, err := datastoremanager.NewManager(ctx)
	s.Require().NoError(err)
	manager.AddPool(ctx, datastore.DefaultPoolName, p)
	s.Require().NoError(Migrate(ctx, manager))

	s.repo = NewScheduleRepository(p)
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
	claims := &security.AuthenticationClaims{TenantID: "test-tenant", PartitionID: "test-partition"}
	claims.Subject = "test-user"
	return claims.ClaimsToContext(context.Background())
}

func seedDue(ctx context.Context, s *ScheduleRepoSuite, n int) []*models.ScheduleDefinition {
	due := time.Now().UTC().Add(-time.Minute)
	out := make([]*models.ScheduleDefinition, 0, n)
	for i := 0; i < n; i++ {
		sched := &models.ScheduleDefinition{
			Name: fmt.Sprintf("s-%d", i), CronExpr: "*/5 * * * *", Timezone: "UTC",
			WorkflowName: "wf", WorkflowVersion: 1, InputPayload: "{}",
			Active: true, NextFireAt: &due,
		}
		s.Require().NoError(s.repo.Create(ctx, sched))
		out = append(out, sched)
	}
	return out
}

// simplePlan emits an event + advances next_fire_at by 5 minutes.
func simplePlan(_ context.Context, sched *models.ScheduleDefinition) (*models.EventLog, *time.Time, int, error) {
	now := time.Now().UTC()
	next := now.Add(5 * time.Minute)

	payload := events.BuildScheduleFiredPayload(sched.ID, sched.Name, now.Format(time.RFC3339), nil)
	raw, _ := payload.ToJSON()

	ev := &models.EventLog{
		EventType:      events.ScheduleFiredType,
		Source:         "schedule:" + sched.ID,
		IdempotencyKey: sched.ID + ":" + now.Format(time.RFC3339Nano),
		Payload:        raw,
	}
	ev.CopyPartitionInfo(&sched.BaseModel)
	return ev, &next, 0, nil
}

// parkPlan returns nil event + nil next — the row must be parked.
func parkPlan(_ context.Context, _ *models.ScheduleDefinition) (*models.EventLog, *time.Time, int, error) {
	return nil, nil, 0, nil
}

func (s *ScheduleRepoSuite) TestClaimAndFireBatch_ExactlyOnceConcurrent() {
	ctx := s.tenantCtx()
	const n = 50
	seedDue(ctx, s, n)

	var total atomic.Int64
	var wg sync.WaitGroup
	for w := 0; w < 10; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				count, err := s.repo.ClaimAndFireBatch(ctx, simplePlan, time.Now().UTC(), 8)
				s.Require().NoError(err)
				total.Add(int64(count))
				if count == 0 {
					return
				}
			}
		}()
	}
	wg.Wait()

	s.Equal(int64(n), total.Load())

	var eventCount int64
	s.Require().NoError(s.dbPool.DB(ctx, false).Model(&models.EventLog{}).
		Where("event_type = ?", events.ScheduleFiredType).Count(&eventCount).Error)
	s.Equal(int64(n), eventCount)
}

func (s *ScheduleRepoSuite) TestClaimAndFireBatch_ParkEmitsNoEvent() {
	ctx := s.tenantCtx()
	seedDue(ctx, s, 3)

	count, err := s.repo.ClaimAndFireBatch(ctx, parkPlan, time.Now().UTC(), 10)
	s.Require().NoError(err)
	s.Equal(3, count, "parked rows still count as processed")

	var eventCount int64
	s.Require().NoError(s.dbPool.DB(ctx, false).Model(&models.EventLog{}).Count(&eventCount).Error)
	s.Equal(int64(0), eventCount, "park must not emit events")

	var rows []*models.ScheduleDefinition
	s.Require().NoError(s.dbPool.DB(ctx, false).Find(&rows).Error)
	for _, r := range rows {
		s.Nil(r.NextFireAt, "parked row must have NULL next_fire_at")
	}
}

func (s *ScheduleRepoSuite) TestCreateBatch_AtomicOnConflict() {
	ctx := s.tenantCtx()

	// Two schedules with the same (tenant, partition, workflow, version, name)
	// — second violates idx_sd_workflow_unique.
	scheds := []*models.ScheduleDefinition{
		{Name: "same", CronExpr: "*/5 * * * *", Timezone: "UTC", WorkflowName: "wf", WorkflowVersion: 1, InputPayload: "{}", Active: false},
		{Name: "same", CronExpr: "0 * * * *", Timezone: "UTC", WorkflowName: "wf", WorkflowVersion: 1, InputPayload: "{}", Active: false},
	}
	s.Require().NoError(copyPartition(ctx, scheds...))

	err := s.repo.CreateBatch(ctx, scheds)
	s.Require().Error(err, "duplicate name must fail")

	var count int64
	s.Require().NoError(s.dbPool.DB(ctx, false).Model(&models.ScheduleDefinition{}).Count(&count).Error)
	s.Equal(int64(0), count, "atomic: neither row should have persisted")
}

func (s *ScheduleRepoSuite) TestCreateBatch_InsertsAll() {
	ctx := s.tenantCtx()

	scheds := []*models.ScheduleDefinition{
		{Name: "a", CronExpr: "*/5 * * * *", Timezone: "UTC", WorkflowName: "wf", WorkflowVersion: 1, InputPayload: "{}", Active: false},
		{Name: "b", CronExpr: "0 * * * *", Timezone: "UTC", WorkflowName: "wf", WorkflowVersion: 1, InputPayload: "{}", Active: false},
		{Name: "c", CronExpr: "*/10 * * * *", Timezone: "UTC", WorkflowName: "wf", WorkflowVersion: 1, InputPayload: "{}", Active: false},
	}
	s.Require().NoError(copyPartition(ctx, scheds...))

	s.Require().NoError(s.repo.CreateBatch(ctx, scheds))

	var count int64
	s.Require().NoError(s.dbPool.DB(ctx, false).Model(&models.ScheduleDefinition{}).Count(&count).Error)
	s.Equal(int64(3), count)
}

func (s *ScheduleRepoSuite) TestActivateByWorkflow_SwitchesVersions() {
	ctx := s.tenantCtx()

	// v1 schedules active.
	v1 := &models.ScheduleDefinition{
		Name: "s", CronExpr: "*/5 * * * *", Timezone: "UTC", WorkflowName: "wf-a", WorkflowVersion: 1, InputPayload: "{}", Active: true,
		NextFireAt: timePtr(time.Now().Add(time.Hour)),
	}
	s.Require().NoError(s.repo.Create(ctx, v1))

	// v2 inserted as inactive (like CreateWorkflow leaves it).
	v2 := &models.ScheduleDefinition{
		Name: "s", CronExpr: "*/10 * * * *", Timezone: "UTC", WorkflowName: "wf-a", WorkflowVersion: 2, InputPayload: "{}", Active: false,
	}
	s.Require().NoError(s.repo.Create(ctx, v2))

	// Activate v2.
	fires := []ScheduleActivation{
		{ID: v2.ID, NextFireAt: time.Now().Add(time.Hour), JitterSeconds: 3},
	}
	s.Require().NoError(s.repo.ActivateByWorkflow(ctx, "wf-a", 2, "test-tenant", "test-partition", fires))

	var after []*models.ScheduleDefinition
	s.Require().NoError(s.dbPool.DB(ctx, false).Where("workflow_name = ?", "wf-a").Order("workflow_version ASC").Find(&after).Error)

	s.Len(after, 2)
	s.Equal(1, after[0].WorkflowVersion)
	s.False(after[0].Active, "v1 must be deactivated")
	s.Nil(after[0].NextFireAt)

	s.Equal(2, after[1].WorkflowVersion)
	s.True(after[1].Active, "v2 must be activated")
	s.NotNil(after[1].NextFireAt)
	s.Equal(3, after[1].JitterSeconds)
}

func (s *ScheduleRepoSuite) TestDeactivateByWorkflow_AllVersions() {
	ctx := s.tenantCtx()

	for v := 1; v <= 3; v++ {
		sch := &models.ScheduleDefinition{
			Name: fmt.Sprintf("s-v%d", v), CronExpr: "*/5 * * * *", Timezone: "UTC",
			WorkflowName: "wf-d", WorkflowVersion: v, InputPayload: "{}", Active: true,
			NextFireAt: timePtr(time.Now().Add(time.Hour)),
		}
		s.Require().NoError(s.repo.Create(ctx, sch))
	}

	s.Require().NoError(s.repo.DeactivateByWorkflow(ctx, "wf-d", "test-tenant", "test-partition"))

	var all []*models.ScheduleDefinition
	s.Require().NoError(s.dbPool.DB(ctx, false).Where("workflow_name = ?", "wf-d").Find(&all).Error)
	s.Len(all, 3)
	for _, r := range all {
		s.False(r.Active, "v%d must be deactivated", r.WorkflowVersion)
		s.Nil(r.NextFireAt)
	}
}

func (s *ScheduleRepoSuite) TestDeactivateByWorkflow_TenantScoped() {
	ctx := s.tenantCtx()

	// Create a schedule in tenant-A.
	a := &models.ScheduleDefinition{
		Name: "x", CronExpr: "*/5 * * * *", Timezone: "UTC",
		WorkflowName: "wf-tenant", WorkflowVersion: 1, InputPayload: "{}", Active: true,
		NextFireAt: timePtr(time.Now().Add(time.Hour)),
	}
	s.Require().NoError(s.repo.Create(ctx, a))

	// Deactivate from a different tenant — must be a no-op on tenant-A's row.
	s.Require().NoError(s.repo.DeactivateByWorkflow(ctx, "wf-tenant", "other-tenant", "other-partition"))

	var check models.ScheduleDefinition
	s.Require().NoError(s.dbPool.DB(ctx, false).First(&check, "id = ?", a.ID).Error)
	s.True(check.Active, "cross-tenant deactivate must not affect tenant-A's row")
}

// copyPartition populates tenant_id/partition_id on plain-struct schedules.
func copyPartition(ctx context.Context, scheds ...*models.ScheduleDefinition) error {
	claims, err := security.ClaimsFromContext(ctx)
	if err != nil {
		return err
	}
	for _, s := range scheds {
		s.TenantID = claims.TenantID
		s.PartitionID = claims.PartitionID
	}
	return nil
}

func timePtr(t time.Time) *time.Time { return &t }
```

Note: `payload.ToJSON()` is a convenience. If `ScheduleFiredPayload` doesn't have a `ToJSON()` method, inline `json.Marshal`. Prefer adding the method to `pkg/events/schedule.go` as `func (p ScheduleFiredPayload) ToJSON() (string, error)` for test readability.

- [ ] **Step 2: Update `pkg/events/schedule.go` to add `ToJSON`**

Append to the file from Task 2:

```go
// ToJSON serialises the payload. Convenience for callers building event rows.
func (p ScheduleFiredPayload) ToJSON() (string, error) {
	raw, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("marshal schedule fired payload: %w", err)
	}
	return string(raw), nil
}
```

Add imports (`encoding/json`, `fmt`).

- [ ] **Step 3: Rewrite `apps/default/service/repository/schedule.go`**

Replace the entire file:

```go
package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/datastore/pool"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

// SchedulePlanFn is invoked per row by ClaimAndFireBatch inside the fire tx.
// Must be pure Go — NO DB access, NO I/O. Returns:
//   - event: the event_log row to emit, or nil to park the schedule
//   - nextFire: the new next_fire_at (nil also parks)
//   - jitterSeconds: value to persist for observability
//   - err: nil on success; non-nil skips this row (others in the batch still commit)
type SchedulePlanFn func(ctx context.Context, sched *models.ScheduleDefinition) (
	event *models.EventLog,
	nextFire *time.Time,
	jitterSeconds int,
	err error,
)

// ScheduleActivation is a single row's activation plan passed to ActivateByWorkflow.
type ScheduleActivation struct {
	ID            string
	NextFireAt    time.Time
	JitterSeconds int
}

// ScheduleRepository manages schedule_definitions persistence.
//
// Every write method is atomic on a single table — no cross-table transactions
// except the repository-internal fire path, which is the only safe way to
// achieve exactly-once fire with outbox emission.
type ScheduleRepository interface {
	Create(ctx context.Context, schedule *models.ScheduleDefinition) error
	CreateBatch(ctx context.Context, scheds []*models.ScheduleDefinition) error

	ListByWorkflow(ctx context.Context, workflowName string, workflowVersion int) ([]*models.ScheduleDefinition, error)

	// ActivateByWorkflow atomically, in one tx on schedule_definitions:
	//   - Deactivates all non-deleted schedules for (workflowName, tenantID, partitionID)
	//     whose workflow_version != workflowVersion.
	//   - Activates rows matching `fires` via VALUES-join UPDATE.
	ActivateByWorkflow(
		ctx context.Context,
		workflowName string,
		workflowVersion int,
		tenantID, partitionID string,
		fires []ScheduleActivation,
	) error

	DeactivateByWorkflow(ctx context.Context, workflowName, tenantID, partitionID string) error

	// ClaimAndFireBatch runs one atomic sweep: SKIP LOCKED claim, per-row
	// plan (pure Go), multi-row event_log INSERT, VALUES-join UPDATE on
	// schedule_definitions. This is the ONE cross-table tx in the codebase —
	// necessary for exactly-once fire semantics, fully enclosed in the repo.
	ClaimAndFireBatch(ctx context.Context, plan SchedulePlanFn, now time.Time, limit int) (fired int, err error)

	// Pool retained for compatibility with callers unrelated to this release.
	Pool() pool.Pool
}

type scheduleRepository struct {
	datastore.BaseRepository[*models.ScheduleDefinition]
	p pool.Pool
}

func NewScheduleRepository(dbPool pool.Pool) ScheduleRepository {
	ctx := context.Background()
	return &scheduleRepository{
		BaseRepository: datastore.NewBaseRepository[*models.ScheduleDefinition](
			ctx, dbPool, nil,
			func() *models.ScheduleDefinition { return &models.ScheduleDefinition{} },
		),
		p: dbPool,
	}
}

func (r *scheduleRepository) Create(ctx context.Context, schedule *models.ScheduleDefinition) error {
	return r.BaseRepository.Create(ctx, schedule)
}

func (r *scheduleRepository) CreateBatch(ctx context.Context, scheds []*models.ScheduleDefinition) error {
	if len(scheds) == 0 {
		return nil
	}
	db := r.p.DB(ctx, false)
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(scheds).Error; err != nil {
			return fmt.Errorf("create schedule batch: %w", err)
		}
		return nil
	})
}

func (r *scheduleRepository) Pool() pool.Pool { return r.p }

func (r *scheduleRepository) ListByWorkflow(
	ctx context.Context,
	workflowName string,
	workflowVersion int,
) ([]*models.ScheduleDefinition, error) {
	// Match the tenancy-scoped list idiom used by WorkflowDefinitionRepository.
	// The implementer: grep for existing "ListActiveByName" or equivalent in
	// apps/default/service/repository/workflow_definition.go and use the
	// same BaseRepository accessor. Do NOT use r.p.DB() directly here —
	// that bypasses the tenancy scope and recreates the v1 audit bug.
	// Pseudocode:
	//   q := r.BaseRepository.<scoped-query-helper>(ctx)
	//   return q.Where("workflow_name = ? AND workflow_version = ?", workflowName, workflowVersion).Order("name ASC").Find(...)
	//
	// Replace this stub with the correct idiom before shipping.
	var out []*models.ScheduleDefinition
	err := r.BaseRepository.Search(ctx, func(q *gorm.DB) *gorm.DB {
		return q.Where("workflow_name = ? AND workflow_version = ?", workflowName, workflowVersion).Order("name ASC")
	}).Scan(&out).Error
	if err != nil {
		return nil, fmt.Errorf("list schedules by workflow: %w", err)
	}
	return out, nil
}

func (r *scheduleRepository) ActivateByWorkflow(
	ctx context.Context,
	workflowName string,
	workflowVersion int,
	tenantID, partitionID string,
	fires []ScheduleActivation,
) error {
	db := r.p.DB(ctx, false)
	return db.Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()

		// Deactivate sibling versions.
		if err := tx.Exec(
			`UPDATE schedule_definitions
			    SET active = false, next_fire_at = NULL, modified_at = ?
			  WHERE workflow_name = ?
			    AND workflow_version <> ?
			    AND tenant_id = ?
			    AND partition_id = ?
			    AND deleted_at IS NULL`,
			now, workflowName, workflowVersion, tenantID, partitionID,
		).Error; err != nil {
			return fmt.Errorf("deactivate sibling versions: %w", err)
		}

		if len(fires) == 0 {
			return nil
		}

		// Activate this version's schedules via VALUES-join UPDATE.
		tuples := make([]string, 0, len(fires))
		args := make([]any, 0, 1+3*len(fires))
		args = append(args, now) // $1 = modified_at (reused below)
		args = append(args, tenantID, partitionID) // WHERE bound
		for i, f := range fires {
			if i == 0 {
				tuples = append(tuples, "(?::uuid, ?::timestamptz, ?::int)")
			} else {
				tuples = append(tuples, "(?, ?, ?)")
			}
			args = append(args, f.ID, f.NextFireAt, f.JitterSeconds)
		}

		sql := fmt.Sprintf(`
			UPDATE schedule_definitions s
			   SET active = true,
			       next_fire_at = v.next_fire_at,
			       jitter_seconds = v.jitter_seconds,
			       modified_at = $1
			  FROM (VALUES %s)
			    AS v(id, next_fire_at, jitter_seconds)
			 WHERE s.id = v.id
			   AND s.tenant_id = $2
			   AND s.partition_id = $3
			   AND s.deleted_at IS NULL`,
			strings.Join(tuples, ", "),
		)

		// pgx positional placeholders: $1 (now), $2 (tenantID), $3 (partitionID),
		// then flattened (id, next_fire_at, jitter_seconds) per row.
		// We pass args in order: [now, tenantID, partitionID, id1, next1, jit1, id2, ...]
		// Note: GORM's Exec uses ?-placeholders. We need to switch either the
		// SQL to ? consistently (GORM will renumber) or build the whole
		// statement with ? and rely on GORM's renumbering. Using ? throughout:
		sql = replaceDollarPlaceholders(sql)
		return tx.Exec(sql, args...).Error
	})
}

// replaceDollarPlaceholders converts $N to ? for GORM's Exec. Shape: $1 $2 $3
// appear exactly once each in the known order; tuples use ? already.
func replaceDollarPlaceholders(sql string) string {
	s := sql
	// Order matters for replacement — avoid $10+ collision (we only have three).
	for _, k := range []struct{ from, to string }{
		{"$1", "?"}, {"$2", "?"}, {"$3", "?"},
	} {
		s = strings.Replace(s, k.from, k.to, 1)
	}
	return s
}

func (r *scheduleRepository) DeactivateByWorkflow(
	ctx context.Context,
	workflowName, tenantID, partitionID string,
) error {
	db := r.p.DB(ctx, false)
	now := time.Now().UTC()
	return db.Exec(
		`UPDATE schedule_definitions
		    SET active = false, next_fire_at = NULL, modified_at = ?
		  WHERE workflow_name = ?
		    AND tenant_id = ?
		    AND partition_id = ?
		    AND deleted_at IS NULL`,
		now, workflowName, tenantID, partitionID,
	).Error
}

// ClaimAndFireBatch is the fire hot path: the single cross-table tx in the
// repository layer, fully enclosed so business never sees it.
func (r *scheduleRepository) ClaimAndFireBatch(
	ctx context.Context,
	plan SchedulePlanFn,
	now time.Time,
	limit int,
) (int, error) {
	db := r.p.DB(ctx, false)
	var fired int

	err := db.Transaction(func(tx *gorm.DB) error {
		var batch []*models.ScheduleDefinition
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("active = ? AND deleted_at IS NULL AND next_fire_at IS NOT NULL AND next_fire_at <= ?", true, now).
			Order("next_fire_at ASC").
			Limit(limit).
			Find(&batch).Error; err != nil {
			return fmt.Errorf("claim due batch: %w", err)
		}
		if len(batch) == 0 {
			return nil
		}

		type planRow struct {
			sched *models.ScheduleDefinition
			event *models.EventLog
			next  *time.Time
			jit   int
		}

		rows := make([]planRow, 0, len(batch))
		events := make([]*models.EventLog, 0, len(batch))
		for _, sched := range batch {
			ev, nf, j, err := plan(ctx, sched)
			if err != nil {
				continue // per-row error: skip from writes, other rows still commit
			}
			rows = append(rows, planRow{sched: sched, event: ev, next: nf, jit: j})
			if ev != nil {
				events = append(events, ev)
			}
		}

		if len(rows) == 0 {
			return nil
		}

		if len(events) > 0 {
			if err := tx.CreateInBatches(events, 500).Error; err != nil {
				return fmt.Errorf("batch insert event_log: %w", err)
			}
		}

		// VALUES-join UPDATE for every row (parked or not — parked sets
		// next_fire_at=NULL which parks the row from the partial index).
		tuples := make([]string, 0, len(rows))
		args := make([]any, 0, 7*len(rows))
		for i, r := range rows {
			if i == 0 {
				tuples = append(tuples, "(?::uuid, ?::text, ?::text, ?::timestamptz, ?::timestamptz, ?::int, ?::timestamptz)")
			} else {
				tuples = append(tuples, "(?, ?, ?, ?, ?, ?, ?)")
			}
			args = append(args,
				r.sched.ID, r.sched.TenantID, r.sched.PartitionID,
				now,        // last_fired_at
				r.next,     // next_fire_at (may be nil → NULL → parked)
				r.jit,
				now,        // modified_at
			)
		}

		sql := fmt.Sprintf(`
			UPDATE schedule_definitions s
			   SET last_fired_at  = v.last_fired_at,
			       next_fire_at   = v.next_fire_at,
			       jitter_seconds = v.jitter_seconds,
			       modified_at    = v.modified_at
			  FROM (VALUES %s)
			    AS v(id, tenant_id, partition_id, last_fired_at, next_fire_at, jitter_seconds, modified_at)
			 WHERE s.id = v.id
			   AND s.tenant_id = v.tenant_id
			   AND s.partition_id = v.partition_id`,
			strings.Join(tuples, ", "),
		)

		if err := tx.Exec(sql, args...).Error; err != nil {
			return fmt.Errorf("batch update schedules: %w", err)
		}

		fired = len(rows)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return fired, nil
}
```

**Critical notes for the implementer:**

1. **`ListByWorkflow` tenancy idiom** — the stub uses `r.BaseRepository.Search(ctx, func(q){...})`. This is a guess; the actual method name in `frame.BaseRepository` varies. Before committing, **grep for how other repositories in this project do tenancy-scoped list queries** (start with `apps/default/service/repository/workflow_definition.go`'s `ListActiveByName`, `ListPage`, etc.), and copy that pattern verbatim. The invariant is: tenancy + partition filter is applied automatically by the scope, NOT manually.

2. **pgx `$N` placeholders vs GORM `?`** — GORM converts `?` to the driver's placeholder. The sketch above uses `$1/$2/$3` for clarity but then calls `replaceDollarPlaceholders` to convert to `?`. Simpler: write the SQL with `?` throughout and order the args accordingly — no conversion function needed. Example:

```go
sql := fmt.Sprintf(`
    UPDATE schedule_definitions s
       SET active = true,
           next_fire_at = v.next_fire_at,
           jitter_seconds = v.jitter_seconds,
           modified_at = ?
      FROM (VALUES %s)
        AS v(id, next_fire_at, jitter_seconds)
     WHERE s.id = v.id
       AND s.tenant_id = ?
       AND s.partition_id = ?
       AND s.deleted_at IS NULL`,
    strings.Join(tuples, ", "),
)
// args: now, <flat tuples>, tenantID, partitionID
```

Note: positional `?` means order MUST match SQL appearance order. Write the SQL so `modified_at` comes first, then VALUES, then tenancy filter. Or shuffle args accordingly.

Simpler approach — two separate UPDATEs:

```sql
-- UPDATE 1: deactivate siblings (plain WHERE, no VALUES)
UPDATE schedule_definitions SET active=false, next_fire_at=NULL, modified_at=?
 WHERE workflow_name=? AND workflow_version<>? AND tenant_id=? AND partition_id=? AND deleted_at IS NULL

-- UPDATE 2: activate this version's schedules (VALUES, then tenancy filter)
UPDATE schedule_definitions s
   SET active = true,
       next_fire_at = v.next_fire_at,
       jitter_seconds = v.jitter_seconds,
       modified_at = ?
  FROM (VALUES (?, ?, ?), (?, ?, ?), ...) AS v(id, next_fire_at, jitter_seconds)
 WHERE s.id = v.id AND s.tenant_id = ? AND s.partition_id = ? AND s.deleted_at IS NULL
```

Args for UPDATE 2, in order: `now`, (id, next, jit) × N, `tenantID`, `partitionID`.

- [ ] **Step 4: Run tests**

```bash
cd /home/j/code/antinvestor/service-trustage
go test ./apps/default/service/repository/ -run TestScheduleRepoSuite -race -v -count=2 | tail -30
```
Expected: all eight test methods pass (in both `-count` iterations).

- [ ] **Step 5: Regression**

```bash
go build ./apps/default/service/repository/...
```

Expected: builds. Other packages (`schedulers/`, `business/`, `main.go`) may fail because `ClaimAndFireBatch` signature changed. Task 5, 8, 9 fix them.

- [ ] **Step 6: Commit**

```bash
git add apps/default/service/repository/schedule.go apps/default/service/repository/schedule_test.go pkg/events/schedule.go
git commit -m "feat(schedule-repo): batched fire + CreateBatch/ActivateByWorkflow/DeactivateByWorkflow

Replaces per-row tx fire path with one tx per sweep — SKIP LOCKED
claim, pure-Go per-row plan, multi-row event_log INSERT, VALUES-join
UPDATE on schedule_definitions. Unlocks ~500 fires/sec/pod from the
v1 ~1.67/sec ceiling.

New single-table atomic operations for lifecycle composition in
business:
  CreateBatch          — atomic multi-insert.
  ActivateByWorkflow   — atomic {deactivate siblings, activate this
                         version} via bulk UPDATEs.
  DeactivateByWorkflow — one-statement bulk deactivate.

ClaimAndFireBatch is the only cross-table tx in the codebase, fully
enclosed in the repo per the spec's transaction-boundary rule. All
other repo methods are single-table."
```

---

## Task 5: Scheduler — `SchedulePlanFn` caller + metrics (deferred) + configurable

**Files:**
- Modify: `apps/default/service/schedulers/cron.go`
- Modify: `apps/default/service/schedulers/scheduler_test.go`

- [ ] **Step 1: Rewrite `cron.go`**

Replace `/home/j/code/antinvestor/service-trustage/apps/default/service/schedulers/cron.go`:

```go
package schedulers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/default/config"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/dsl"
	"github.com/antinvestor/service-trustage/pkg/events"
)

const cronMissedFireThreshold = 5 * time.Minute

// CronScheduler runs the fire loop. It implements the plan side (pure Go);
// the repo owns the transaction. Configuration comes from Config.
type CronScheduler struct {
	scheduleRepo repository.ScheduleRepository
	cfg          *config.Config
}

// NewCronScheduler — metrics are NOT yet wired here (Task 9 adds them).
func NewCronScheduler(scheduleRepo repository.ScheduleRepository, cfg *config.Config) *CronScheduler {
	return &CronScheduler{scheduleRepo: scheduleRepo, cfg: cfg}
}

func (s *CronScheduler) Start(ctx context.Context) {
	log := util.Log(ctx)

	interval := time.Duration(s.cfg.CronSchedulerIntervalSecs) * time.Second
	if interval <= 0 {
		interval = time.Second
	}
	log.Debug("cron scheduler started",
		"interval_seconds", int(interval.Seconds()),
		"batch_size", s.cfg.CronSchedulerBatchSize,
	)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.RunOnce(ctx)
		case <-ctx.Done():
			log.Debug("cron scheduler stopped")
			return
		}
	}
}

func (s *CronScheduler) RunOnce(ctx context.Context) int {
	log := util.Log(ctx)
	now := time.Now().UTC()

	batchSize := s.cfg.CronSchedulerBatchSize
	if batchSize <= 0 {
		batchSize = 500
	}

	fired, err := s.scheduleRepo.ClaimAndFireBatch(ctx, s.planOne, now, batchSize)
	if err != nil {
		log.WithError(err).Error("cron scheduler: sweep failed")
		return 0
	}
	if fired > 0 {
		log.Debug("cron scheduler swept", "fired", fired)
	}
	return fired
}

// planOne implements repository.SchedulePlanFn. Pure Go, no DB.
func (s *CronScheduler) planOne(
	ctx context.Context,
	sched *models.ScheduleDefinition,
) (*models.EventLog, *time.Time, int, error) {
	log := util.Log(ctx)
	now := time.Now().UTC()

	cronSched, err := dsl.ParseCron(sched.CronExpr)
	if err != nil {
		log.WithError(err).Error("cron scheduler: invalid cron, parking",
			"schedule_id", sched.ID, "cron_expr", sched.CronExpr)
		return nil, nil, 0, nil // park, no event
	}

	base := now
	if sched.NextFireAt != nil && now.Sub(*sched.NextFireAt) <= cronMissedFireThreshold {
		base = *sched.NextFireAt
	}

	nominal, err := cronSched.NextInZone(base, sched.Timezone)
	if err != nil {
		log.WithError(err).Error("cron scheduler: invalid timezone, parking",
			"schedule_id", sched.ID, "timezone", sched.Timezone)
		return nil, nil, 0, nil
	}

	jitter := dsl.JitterFor(sched.ID, cronSched, nominal)
	next := nominal.Add(jitter)

	ev := buildEvent(sched, now)

	return ev, &next, int(jitter / time.Second), nil
}

func buildEvent(sched *models.ScheduleDefinition, now time.Time) *models.EventLog {
	var input map[string]any
	if sched.InputPayload != "" {
		var tmp map[string]any
		if err := json.Unmarshal([]byte(sched.InputPayload), &tmp); err == nil {
			input = tmp
		}
	}

	payload := events.BuildScheduleFiredPayload(
		sched.ID, sched.Name, now.Format(time.RFC3339), input,
	)
	raw, _ := payload.ToJSON()

	ev := &models.EventLog{
		EventType:      events.ScheduleFiredType,
		Source:         "schedule:" + sched.ID,
		IdempotencyKey: sched.ID + ":" + now.Format(time.RFC3339Nano),
		Payload:        raw,
	}
	ev.CopyPartitionInfo(&sched.BaseModel)
	return ev
}
```

- [ ] **Step 2: Update `scheduler_test.go`**

Remove references to the v1 signature (`NewCronScheduler(scheduleRepo, eventRepo, cfg)` → `NewCronScheduler(scheduleRepo, cfg)`). Remove references to deleted helpers. Replace duration-syntax `CronExpr: "1h"` with `"*/5 * * * *"`.

Add unit tests:

```go
func TestPlanOne_InvalidCronParks(t *testing.T) {
	sched := &models.ScheduleDefinition{Name: "bad", CronExpr: "not-a-cron", Timezone: "UTC"}
	sched.ID = "s-1"

	s := &CronScheduler{cfg: &config.Config{CronSchedulerBatchSize: 50, CronSchedulerIntervalSecs: 1}}
	ev, next, jit, err := s.planOne(context.Background(), sched)

	require.NoError(t, err)
	require.Nil(t, ev, "invalid cron must not emit an event")
	require.Nil(t, next, "invalid cron must park")
	require.Equal(t, 0, jit)
}

func TestPlanOne_MissedFireSkipsForward(t *testing.T) {
	past := time.Now().UTC().Add(-time.Hour) // > 5 min stale
	sched := &models.ScheduleDefinition{
		Name: "stale", CronExpr: "*/5 * * * *", Timezone: "UTC", NextFireAt: &past,
	}
	sched.ID = "s-2"

	s := &CronScheduler{cfg: &config.Config{CronSchedulerBatchSize: 50, CronSchedulerIntervalSecs: 1}}
	ev, next, _, err := s.planOne(context.Background(), sched)

	require.NoError(t, err)
	require.NotNil(t, ev)
	require.NotNil(t, next)
	require.True(t, next.After(time.Now().UTC()), "missed-fire must skip forward")
}

func TestPlanOne_InvalidTimezoneParks(t *testing.T) {
	sched := &models.ScheduleDefinition{Name: "tz", CronExpr: "*/5 * * * *", Timezone: "Not/A/Zone"}
	sched.ID = "s-3"

	s := &CronScheduler{cfg: &config.Config{CronSchedulerBatchSize: 50, CronSchedulerIntervalSecs: 1}}
	ev, next, _, _ := s.planOne(context.Background(), sched)

	require.Nil(t, ev)
	require.Nil(t, next)
}
```

- [ ] **Step 3: Build + test**

```bash
cd /home/j/code/antinvestor/service-trustage
go build ./apps/default/service/schedulers/
go test ./apps/default/service/schedulers/ -race -v | tail -20
```
Expected: green. `main.go` may still fail to build — Task 9 fixes the wiring.

- [ ] **Step 4: Commit**

```bash
git add apps/default/service/schedulers/cron.go apps/default/service/schedulers/scheduler_test.go
git commit -m "feat(scheduler): implement SchedulePlanFn + configurable batch/interval

Scheduler becomes a thin SchedulePlanFn — pure Go, no DB. Repo drives
the transaction. Config-driven batch size and interval move ceiling
tuning out of code. Invalid cron or invalid timezone at fire time
parks the schedule (no ghost event). Timezone-aware Next via
dsl.NextInZone."
```

---

## Task 6: Outbox batch / concurrent publish

**Files:**
- Modify: `apps/default/service/schedulers/outbox.go`
- Modify: `apps/default/config/config.go`

- [ ] **Step 1: Read Frame's QueueManager API**

```bash
grep -rn "func.*Publish\|type.*QueueManager" /home/j/go/pkg/mod/github.com/pitabwire/frame@v1.94.1/ 2>/dev/null | grep -iE "publish|queue" | head
```

If Frame exposes a batched `Publish(subject, payloads ...[]byte)`, use it. Otherwise use `workerpool` from `github.com/pitabwire/util` (or `golang.org/x/sync/errgroup` with a bounded semaphore).

- [ ] **Step 2: Implement batching/concurrency**

Example (adapt to what Frame actually exposes). Read `outbox.go` to see the current single-loop publish, then:

- Simple case (Frame batches): swap the single-publish loop for one variadic call.
- Fallback case (one-at-a-time): worker pool sized by `cfg.OutboxPublishConcurrency`.

Add `OutboxPublishConcurrency int \`env:"OUTBOX_PUBLISH_CONCURRENCY" envDefault:"16"\`` to `config.Config`.

- [ ] **Step 3: Run tests**

```bash
go test ./apps/default/service/schedulers/ -run 'TestOutbox' -race -v | tail
go build ./apps/default/service/schedulers/
```

- [ ] **Step 4: Commit**

```bash
git add apps/default/service/schedulers/outbox.go apps/default/config/config.go
git commit -m "feat(outbox): batch / parallel NATS publish per sweep

At high fire rates, per-event NATS round-trips become the secondary
bottleneck. Publishes are now either batched via Frame's variadic
Publish (if supported) or parallelised through a bounded worker pool
(OUTBOX_PUBLISH_CONCURRENCY, default 16)."
```

---

## Task 7: Business layer — atomic-by-ordering CreateWorkflow / ActivateWorkflow / ArchiveWorkflow

**Files:**
- Modify: `apps/default/service/business/workflow.go`
- Modify: `apps/default/service/business/workflow_integration_test.go`

- [ ] **Step 1: Add failing tests**

Append to `workflow_integration_test.go`:

```go
func (s *WorkflowSuite) TestCreateWorkflow_SchedulesAtomic() {
	ctx := s.tenantCtx()

	// Two schedules with the same name — second hits idx_sd_workflow_unique.
	blob := []byte(`{
		"version":"v1","name":"w-dup",
		"steps":[{"id":"s","type":"delay","delay":{"duration":"1s"}}],
		"schedules":[
			{"name":"dup","cron_expr":"*/5 * * * *"},
			{"name":"dup","cron_expr":"0 * * * *"}
		]
	}`)
	_, err := s.biz.CreateWorkflow(ctx, blob)
	s.Require().Error(err, "duplicate schedule names must fail")
	// The workflow row may still exist (Tx1 committed) — that's the documented
	// orphan state. Retrying the same DSL blocks on idx_wd_name_version.
	_, retryErr := s.biz.CreateWorkflow(ctx, blob)
	s.Require().Error(retryErr, "retry must be blocked by workflow unique index")
}

func (s *WorkflowSuite) TestArchiveWorkflow_DeactivatesSchedulesFirst() {
	ctx := s.tenantCtx()

	blob := []byte(`{
		"version":"v1","name":"w-arch",
		"steps":[{"id":"s","type":"delay","delay":{"duration":"1s"}}],
		"schedules":[{"name":"h","cron_expr":"0 * * * *"}]
	}`)
	v1, err := s.biz.CreateWorkflow(ctx, blob)
	s.Require().NoError(err)
	s.Require().NoError(s.biz.ActivateWorkflow(ctx, v1.ID))

	s.Require().NoError(s.biz.ArchiveWorkflow(ctx, v1.ID))

	scheds, err := s.scheduleRepo.ListByWorkflow(ctx, v1.Name, v1.WorkflowVersion)
	s.Require().NoError(err)
	s.Len(scheds, 1)
	s.False(scheds[0].Active)
	s.Nil(scheds[0].NextFireAt)

	got, err := s.biz.GetWorkflow(ctx, v1.ID)
	s.Require().NoError(err)
	s.Equal(models.WorkflowStatusArchived, got.Status)
}

func (s *WorkflowSuite) TestListByWorkflow_TenantIsolated() {
	ctxA := s.tenantCtx()
	blob := []byte(`{
		"version":"v1","name":"w-iso",
		"steps":[{"id":"s","type":"delay","delay":{"duration":"1s"}}],
		"schedules":[{"name":"x","cron_expr":"*/5 * * * *"}]
	}`)
	_, err := s.biz.CreateWorkflow(ctxA, blob)
	s.Require().NoError(err)

	// Tenant B sees no schedules for the same workflow name.
	claimsB := &security.AuthenticationClaims{TenantID: "tenant-B", PartitionID: "partition-B"}
	claimsB.Subject = "user-B"
	ctxB := claimsB.ClaimsToContext(context.Background())

	out, err := s.scheduleRepo.ListByWorkflow(ctxB, "w-iso", 1)
	s.Require().NoError(err)
	s.Empty(out, "cross-tenant read must return empty")
}
```

- [ ] **Step 2: Rewrite `CreateWorkflow`**

In `apps/default/service/business/workflow.go`, replace `CreateWorkflow`:

```go
func (b *workflowBusiness) CreateWorkflow(
	ctx context.Context,
	dslBlob json.RawMessage,
) (*models.WorkflowDefinition, error) {
	log := util.Log(ctx)

	spec, err := dsl.Parse(dslBlob)
	if err != nil {
		return nil, fmt.Errorf("parse DSL: %w", err)
	}
	if res := dsl.Validate(spec); !res.Valid() {
		return nil, fmt.Errorf("%w: %w", ErrDSLValidationFailed, res.Error())
	}
	if err := validateExecutableWorkflow(spec); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDSLValidationFailed, err)
	}

	def := &models.WorkflowDefinition{
		Name:            spec.Name,
		WorkflowVersion: 1,
		Status:          models.WorkflowStatusDraft,
		DSLBlob:         string(dslBlob),
	}
	if spec.Timeout.Duration > 0 {
		def.TimeoutSeconds = int64(spec.Timeout.Duration.Seconds())
	}

	if err := b.registerStepSchemas(ctx, spec); err != nil {
		return nil, fmt.Errorf("register schemas: %w", err)
	}

	// Tx1: workflow row (single-table).
	if err := b.defRepo.Create(ctx, def); err != nil {
		return nil, fmt.Errorf("persist workflow: %w", err)
	}

	// Tx2: schedule rows (single-table atomic batch).
	scheds, err := planScheduleRows(def, spec)
	if err != nil {
		return nil, err
	}
	if len(scheds) > 0 {
		if err := b.scheduleRepo.CreateBatch(ctx, scheds); err != nil {
			log.WithError(err).Error("schedule materialisation failed; workflow is orphan DRAFT",
				"workflow_id", def.ID, "name", def.Name)
			return nil, fmt.Errorf("materialise schedules (orphan DRAFT at workflow %s — retry blocked by idx_wd_name_version): %w", def.ID, err)
		}
	}

	log.Info("workflow created", "workflow_id", def.ID, "name", def.Name)
	return def, nil
}

func planScheduleRows(def *models.WorkflowDefinition, spec *dsl.WorkflowSpec) ([]*models.ScheduleDefinition, error) {
	out := make([]*models.ScheduleDefinition, 0, len(spec.Schedules))
	for _, sspec := range spec.Schedules {
		payloadJSON := "{}"
		if len(sspec.InputPayload) > 0 {
			raw, err := json.Marshal(sspec.InputPayload)
			if err != nil {
				return nil, fmt.Errorf("marshal input_payload for %s: %w", sspec.Name, err)
			}
			payloadJSON = string(raw)
		}

		tz := sspec.Timezone
		if tz == "" {
			tz = "UTC"
		}

		sched := &models.ScheduleDefinition{
			Name:            sspec.Name,
			CronExpr:        sspec.CronExpr,
			Timezone:        tz,
			WorkflowName:    def.Name,
			WorkflowVersion: def.WorkflowVersion,
			InputPayload:    payloadJSON,
			Active:          false,
			NextFireAt:      nil,
			JitterSeconds:   0,
		}
		sched.CopyPartitionInfo(&def.BaseModel)
		out = append(out, sched)
	}
	return out, nil
}
```

Delete the old `materialiseSchedules` method.

- [ ] **Step 3: Rewrite `ActivateWorkflow`**

```go
func (b *workflowBusiness) ActivateWorkflow(ctx context.Context, id string) error {
	log := util.Log(ctx)

	def, err := b.defRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrWorkflowNotFound, err)
	}
	if err := def.TransitionTo(models.WorkflowStatusActive); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidWorkflowStatus, err)
	}

	// Tx1: workflow status.
	if err := b.defRepo.Update(ctx, def); err != nil {
		return fmt.Errorf("update workflow: %w", err)
	}

	// Build fire plans for this version's schedules (pure Go, no DB write).
	myScheds, err := b.scheduleRepo.ListByWorkflow(ctx, def.Name, def.WorkflowVersion)
	if err != nil {
		return fmt.Errorf("list schedules: %w", err)
	}

	now := time.Now().UTC()
	fires := make([]repository.ScheduleActivation, 0, len(myScheds))
	for _, sch := range myScheds {
		cronSched, parseErr := dsl.ParseCron(sch.CronExpr)
		if parseErr != nil {
			return fmt.Errorf("parse cron for %s: %w", sch.Name, parseErr)
		}
		nominal, err := cronSched.NextInZone(now, sch.Timezone)
		if err != nil {
			return fmt.Errorf("timezone for %s: %w", sch.Name, err)
		}
		jitter := dsl.JitterFor(sch.ID, cronSched, nominal)
		fires = append(fires, repository.ScheduleActivation{
			ID:            sch.ID,
			NextFireAt:    nominal.Add(jitter),
			JitterSeconds: int(jitter / time.Second),
		})
	}

	// Tx2: deactivate siblings + activate this version.
	if err := b.scheduleRepo.ActivateByWorkflow(
		ctx, def.Name, def.WorkflowVersion, def.TenantID, def.PartitionID, fires,
	); err != nil {
		log.WithError(err).Error("activate schedules failed; workflow ACTIVE but schedules stale; retry to reconcile",
			"workflow_id", def.ID)
		return fmt.Errorf("activate schedules: %w", err)
	}
	return nil
}
```

Delete the old V1 `ActivateWorkflow` body and its inline SQL. Delete any local `jitterForSchedule`, `listSchedulesTx`, `applyActivationUpdates`, `buildActivationPlans` that v1 added — they're superseded by the repo's `ActivateByWorkflow` + `dsl.JitterFor`.

- [ ] **Step 4: Add `ArchiveWorkflow`**

Append:

```go
func (b *workflowBusiness) ArchiveWorkflow(ctx context.Context, id string) error {
	log := util.Log(ctx)

	def, err := b.defRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrWorkflowNotFound, err)
	}
	if err := def.TransitionTo(models.WorkflowStatusArchived); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidWorkflowStatus, err)
	}

	// Tx1: schedules off FIRST (safe failure ordering).
	if err := b.scheduleRepo.DeactivateByWorkflow(ctx, def.Name, def.TenantID, def.PartitionID); err != nil {
		return fmt.Errorf("deactivate schedules: %w", err)
	}

	// Tx2: workflow status.
	if err := b.defRepo.Update(ctx, def); err != nil {
		log.WithError(err).Error("workflow status update failed after schedules deactivated; retry to reconcile",
			"workflow_id", def.ID)
		return fmt.Errorf("update workflow status: %w", err)
	}
	return nil
}
```

Add `ArchiveWorkflow(ctx, id) error` to the `WorkflowBusiness` interface.

- [ ] **Step 5: Fix imports**

Ensure `workflow.go` imports:
- `"time"`, `"encoding/json"`, `"fmt"`
- `"github.com/pitabwire/util"`
- `"github.com/antinvestor/service-trustage/dsl"`
- `"github.com/antinvestor/service-trustage/apps/default/service/models"`
- `"github.com/antinvestor/service-trustage/apps/default/service/repository"`

Remove `"hash/fnv"` (gone now), `"gorm.io/gorm"` if no longer directly referenced, `"strings"` if no longer used.

- [ ] **Step 6: Build + test**

```bash
cd /home/j/code/antinvestor/service-trustage
go build ./...
go test ./apps/default/service/business/... -race -run 'TestCreateWorkflow|TestActivateWorkflow|TestArchiveWorkflow|TestListByWorkflow|TestGetWorkflow' -v | tail -40
```

- [ ] **Step 7: Commit**

```bash
git add apps/default/service/business/workflow.go apps/default/service/business/workflow_integration_test.go
git commit -m "feat(workflow): two-tx Create/Activate; new Archive; tenancy-safe List

All lifecycle operations are two sequential single-table tx — no
cross-repo transaction in business code.
  CreateWorkflow:   Tx1 workflow (DRAFT), Tx2 scheduleRepo.CreateBatch.
  ActivateWorkflow: Tx1 workflow ACTIVE, Tx2
                    scheduleRepo.ActivateByWorkflow (which atomically
                    activates this version + deactivates siblings on
                    one table).
  ArchiveWorkflow:  Tx1 scheduleRepo.DeactivateByWorkflow (schedules
                    off first for safe failure mode), Tx2 workflow
                    ARCHIVED.

ListByWorkflow now honours tenancy scope — fixes v1 cross-tenant
read leak."
```

---

## Task 8: Proto + handler — `ArchiveWorkflow` RPC, `timezone` field

**Files:**
- Modify: `proto/workflow/v1/workflow.proto`
- Modify: `apps/default/service/handlers/workflow_connect.go`
- Modify: `apps/default/service/handlers/connect_helpers.go`
- Regenerated: `gen/go/workflow/v1/...`

- [ ] **Step 1: Proto changes**

Add to `ScheduleDefinition` message:

```proto
string timezone = 12;
```

Inside the `WorkflowService` block:

```proto
rpc ArchiveWorkflow(ArchiveWorkflowRequest) returns (ArchiveWorkflowResponse) {
  option (common.v1.method_permissions) = { permissions: ["workflow_manage"] };
}
```

Messages:

```proto
message ArchiveWorkflowRequest  { string id = 1; }
message ArchiveWorkflowResponse { WorkflowDefinition workflow = 1; }
```

- [ ] **Step 2: Regenerate**

```bash
cd /home/j/code/antinvestor/service-trustage
make proto-gen
```

- [ ] **Step 3: Handler**

Add to `apps/default/service/handlers/workflow_connect.go`:

```go
func (h *WorkflowConnectHandler) ArchiveWorkflow(
	ctx context.Context,
	req *connect.Request[workflowv1.ArchiveWorkflowRequest],
) (*connect.Response[workflowv1.ArchiveWorkflowResponse], error) {
	id := req.Msg.GetId()
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id required"))
	}

	if err := h.workflowBiz.ArchiveWorkflow(ctx, id); err != nil {
		switch {
		case errors.Is(err, business.ErrWorkflowNotFound):
			return nil, connect.NewError(connect.CodeNotFound, err)
		case errors.Is(err, business.ErrInvalidWorkflowStatus):
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	def, err := h.workflowBiz.GetWorkflow(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&workflowv1.ArchiveWorkflowResponse{
		Workflow: workflowDefinitionToProto(def),
	}), nil
}
```

- [ ] **Step 4: Add `Timezone` to schedule proto conversion**

In `connect_helpers.go`, find `scheduleDefinitionsToProto` / `scheduleDefinitionToProto`. Add:

```go
api.Timezone = s.Timezone
```

- [ ] **Step 5: Build + test**

```bash
go build ./...
go test ./apps/default/... -race | tail
```

- [ ] **Step 6: Commit**

```bash
git add proto/workflow/v1/workflow.proto gen/go/workflow/v1/ apps/default/service/handlers/workflow_connect.go apps/default/service/handlers/connect_helpers.go
git commit -m "feat(workflow-proto): ArchiveWorkflow RPC + Schedule.timezone wire field

ArchiveWorkflow reuses existing workflow_manage permission — no
Keto OPL churn. Schedule.timezone surfaces the per-schedule IANA
zone through GetWorkflow responses."
```

---

## Task 9: Observability + Config + main.go wiring

**Files:**
- Modify: `pkg/telemetry/metrics.go`
- Modify: `apps/default/config/config.go`
- Modify: `apps/default/cmd/main.go`
- Modify: `apps/default/service/schedulers/cron.go` (re-add metrics calls)

- [ ] **Step 1: Add telemetry constants + Metrics fields**

Follow the `outbox.go` pattern at `apps/default/service/schedulers/outbox.go:107`. In `pkg/telemetry/metrics.go`:

```go
const SpanSchedulerCron = "scheduler.cron.sweep"
```

Add three instruments on `Metrics`:
- `SchedulerCronFired` — `metric.Int64Counter` (`scheduler_cron_fired_total`), labeled `result="ok"|"fail"`
- `SchedulerCronSweepDuration` — `metric.Float64Histogram` (`scheduler_cron_sweep_duration_seconds`)
- `SchedulerCronInvalid` — `metric.Int64Counter` (`scheduler_cron_invalid_cron_total`)

Expose helpers:

```go
func (m *Metrics) RecordSchedulerCronSweep(ctx context.Context, fired int, dur time.Duration, ok bool) { … }
func (m *Metrics) IncrementSchedulerCronInvalid(ctx context.Context, sched *models.ScheduleDefinition) { … }
```

- [ ] **Step 2: Config additions**

Add to `apps/default/config/config.go`:

```go
CronSchedulerBatchSize     int `env:"CRON_SCHEDULER_BATCH_SIZE"       envDefault:"500"`
CronSchedulerIntervalSecs  int `env:"CRON_SCHEDULER_INTERVAL_SECONDS" envDefault:"1"`

SchedulerPoolMaxConns      int `env:"SCHEDULER_POOL_MAX_CONNS" envDefault:"10"`
SchedulerPoolMinConns      int `env:"SCHEDULER_POOL_MIN_CONNS" envDefault:"2"`
```

(`OutboxPublishConcurrency` was added in Task 6.)

- [ ] **Step 3: Scheduler — re-add metrics calls**

Edit `apps/default/service/schedulers/cron.go`:

Add `metrics *telemetry.Metrics` field to `CronScheduler`. Update `NewCronScheduler` signature:

```go
func NewCronScheduler(scheduleRepo repository.ScheduleRepository, cfg *config.Config, metrics *telemetry.Metrics) *CronScheduler {
    return &CronScheduler{scheduleRepo: scheduleRepo, cfg: cfg, metrics: metrics}
}
```

In `RunOnce`:

```go
ctx, span := telemetry.StartSpan(ctx, telemetry.TracerEngine, telemetry.SpanSchedulerCron)
defer telemetry.EndSpan(span)

start := time.Now()
// ... existing call to ClaimAndFireBatch ...

if s.metrics != nil {
    s.metrics.RecordSchedulerCronSweep(ctx, fired, time.Since(start), err == nil)
}
```

In `planOne`, add metrics increment on the park-for-invalid paths:

```go
if parseErr != nil {
    if s.metrics != nil {
        s.metrics.IncrementSchedulerCronInvalid(ctx, sched)
    }
    return nil, nil, 0, nil
}
// ... similar for invalid timezone ...
```

- [ ] **Step 4: Wire dedicated scheduler pool in `main.go`**

Edit `apps/default/cmd/main.go`. After the main service setup, before scheduler creation:

```go
// Dedicated scheduler pool — isolates fire-path connections from HTTP/RPC handlers.
schedulerPool := pool.NewPool(ctx)
dbURLs := cfg.GetDatabasePrimaryHostURL() // Frame ConfigurationDatabase accessor
if len(dbURLs) == 0 {
    log.Fatal("no database primary URL available for scheduler pool")
}
if err := schedulerPool.AddConnection(ctx,
    pool.WithConnection(dbURLs[0], false),
    pool.WithPreparedStatements(false),
    pool.WithPreferSimpleProtocol(true),
    pool.WithMaxConnections(cfg.SchedulerPoolMaxConns),
    pool.WithMinConnections(cfg.SchedulerPoolMinConns),
); err != nil {
    log.WithError(err).Fatal("scheduler pool init")
}
svc.DatastoreManager().AddPool(ctx, "scheduler", schedulerPool)

schedulerScheduleRepo := repository.NewScheduleRepository(schedulerPool)
```

Verify the exact pool accessors against Frame (grep `WithMaxConnections` / `WithMinConnections` in `frame@v1.94.1/datastore/pool/options.go`).

Update the `CronScheduler` wiring:

```go
cronSched := schedulers.NewCronScheduler(schedulerScheduleRepo, &cfg, metrics)
```

(Replacing the v1 signature that took `eventLogRepo`.)

- [ ] **Step 5: Build + run tests**

```bash
cd /home/j/code/antinvestor/service-trustage
go build ./...
go test ./apps/default/... -race | tail
```

- [ ] **Step 6: Commit**

```bash
git add pkg/telemetry/metrics.go apps/default/config/config.go apps/default/cmd/main.go apps/default/service/schedulers/cron.go
git commit -m "feat(scheduler): observability + dedicated pool + config-driven tuning

- Metrics: scheduler_cron_fired_total{result}, sweep_duration_seconds
  histogram, invalid_cron_total counter. SpanSchedulerCron span.
- Config: CRON_SCHEDULER_BATCH_SIZE (500), CRON_SCHEDULER_INTERVAL_SECONDS
  (1), SCHEDULER_POOL_{MAX,MIN}_CONNS.
- main.go: dedicated scheduler pool via datastore.Manager.AddPool
  isolates fire connections from HTTP/RPC under herd load."
```

---

## Task 10: Release v0.3.35 + cluster verify

**Files:** none.

- [ ] **Step 1: Full tests + lint**

```bash
cd /home/j/code/antinvestor/service-trustage
make tests
make lint
```
Expected: green. If anything fails, STOP.

- [ ] **Step 2: Push + tag**

```bash
git log --oneline origin/main..HEAD
git push origin main
git tag -a v0.3.35 -m "Release v0.3.35: scheduler v1.1"
git push origin v0.3.35
```

- [ ] **Step 3: Watch release workflow**

```bash
RUN_ID=$(gh run list --workflow=Release --limit 5 --json databaseId,headBranch -q '.[] | select(.headBranch == "v0.3.35") | .databaseId' | head -1)
gh run watch "$RUN_ID" --exit-status
```

- [ ] **Step 4: Nudge Flux**

```bash
kubectl -n trustage annotate imagerepository trustage reconcile.fluxcd.io/requestedAt="$(date -Iseconds)" --overwrite
for i in $(seq 1 6); do
  tag=$(kubectl -n trustage get imagepolicy trustage -o jsonpath='{.status.latestRef.tag}' 2>/dev/null)
  echo "[$i] $tag"
  [[ "$tag" == "v0.3.35" ]] && break
  sleep 30
done
```

- [ ] **Step 5: Bounce pgBouncer** (migration added timezone column + dropped/recreated an index)

```bash
kubectl -n datastore rollout restart deployment/hub-pooler-rw deployment/hub-pooler-ro 2>/dev/null || \
  kubectl -n datastore delete pods -l cnpg.io/cluster=hub,application=pooler --grace-period=30
```

- [ ] **Step 6: Observe roll**

```bash
kubectl -n trustage get helmrelease trustage
kubectl -n trustage get jobs -l app.kubernetes.io/component=migration --sort-by=.metadata.creationTimestamp | tail
kubectl -n trustage get pods -l app.kubernetes.io/name=trustage
kubectl -n trustage logs deploy/trustage --tail=40 | grep -E 'Build info|cron scheduler|migrat'
```
Expected: migration Job complete, pods 1/1 Running, logs show `version=v0.3.35` and `cron scheduler started interval_seconds=1 batch_size=500`.

- [ ] **Step 7: End-to-end smoke** (if token available)

```bash
TOKEN=...
curl -sS -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  https://api.stawi.dev/trustage/workflow.v1.WorkflowService/CreateWorkflow \
  -X POST -d '{
    "dsl":{
      "version":"v1","name":"v11smoke",
      "steps":[{"id":"s","type":"delay","delay":{"duration":"1s"}}],
      "schedules":[{"name":"m","cron_expr":"*/1 * * * *","timezone":"UTC"}]
    }
  }' | jq '.workflow.id,.schedules'
```

Activate, wait 90 s, verify event_log accrual:

```bash
WID=<id>
curl -sS -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  https://api.stawi.dev/trustage/workflow.v1.WorkflowService/ActivateWorkflow \
  -X POST -d "{\"id\":\"$WID\"}" | jq

sleep 90
kubectl -n trustage exec deploy/trustage -- sh -c 'PGPASSWORD=$DATABASE_PASSWORD psql -h pooler-ro.datastore.svc -U $DATABASE_USERNAME trustage -tAc "SELECT count(*) FROM event_log WHERE event_type='\''schedule.fired'\'';"'
```

Archive the smoke workflow:

```bash
curl -sS -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  https://api.stawi.dev/trustage/workflow.v1.WorkflowService/ArchiveWorkflow \
  -X POST -d "{\"id\":\"$WID\"}" | jq
```

- [ ] **Step 8: Sustained health**

```bash
for i in $(seq 1 8); do
  printf "[%02d] " "$i"; date -u '+%H:%M:%SZ'
  kubectl -n trustage get pods -l app.kubernetes.io/name=trustage --no-headers
  sleep 15
done
```
Expected: continuous `1/1 Running`, 0 restarts.

- [ ] **Step 9: Sign-off**

Report:
- Release workflow URL + conclusion
- ImagePolicy tag + digest
- HelmRelease status, pod restart count
- Metrics series present in Prometheus
- Smoke test event_log count
- Any operational notes

---

## Rollback

- **Dial-down**: `CRON_SCHEDULER_BATCH_SIZE=1` env on the Deployment → restart. Back to per-row semantics on the new code. No redeploy needed.
- **Full rollback**: pin `v0.3.34` in the ImagePolicy filter. Schema changes are additive; safe to keep.
- **Revert main**: `git revert origin/main..HEAD^` and retag.
