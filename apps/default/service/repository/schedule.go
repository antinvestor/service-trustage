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

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/datastore/pool"
	"github.com/pitabwire/frame/security"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

const (
	// activateArgsTrailingCount is the number of trailing args (tenantID, partitionID)
	// appended after the per-activation VALUES tuples in ActivateByWorkflow.
	activateArgsTrailingCount = 2
	// activateArgsPerFire is the number of positional args per ScheduleActivation tuple.
	activateArgsPerFire = 3
	// claimAndFireArgsPerRow is the number of positional args per row in the ClaimAndFireBatch VALUES update.
	claimAndFireArgsPerRow = 7
	// eventLogInsertBatchSize is the GORM batch size for multi-row event_log inserts.
	eventLogInsertBatchSize = 500
)

// schedulePlanRow holds the per-row output of the plan function inside
// ClaimAndFireBatch. It is unexported; only the helper functions in this file
// consume it.
type schedulePlanRow struct {
	sched *models.ScheduleDefinition
	event *models.EventLog
	next  *time.Time
	jit   int
}

// buildBatchUpdateSQL constructs the VALUES-join UPDATE SQL and its positional
// args for ClaimAndFireBatch. IDs are stored as character varying — the first
// tuple casts to ::text/::timestamptz/::int so PostgreSQL can infer types for
// the remaining plain-? rows.
func buildBatchUpdateSQL(rows []schedulePlanRow, now time.Time) (string, []any) {
	tuples := make([]string, 0, len(rows))
	args := make([]any, 0, claimAndFireArgsPerRow*len(rows))
	for i, pr := range rows {
		if i == 0 {
			tuples = append(tuples,
				"(?::text, ?::text, ?::text, ?::timestamptz, ?::timestamptz, ?::int, ?::timestamptz)",
			)
		} else {
			tuples = append(tuples, "(?, ?, ?, ?, ?, ?, ?)")
		}
		args = append(args,
			pr.sched.ID, pr.sched.TenantID, pr.sched.PartitionID,
			now,     // last_fired_at
			pr.next, // nil → NULL (parks the row)
			pr.jit,
			now, // modified_at
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
	return sql, args
}

// SchedulePlanFn is invoked per row by ClaimAndFireBatch inside the fire tx.
// Must be pure Go — NO DB access, NO I/O. Returns:
//   - event: event_log row to emit, or nil to park the schedule
//   - nextFire: new next_fire_at, or nil to park
//   - jitterSeconds: value persisted on the schedule row
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
// Every write method is atomic on a single table, except ClaimAndFireBatch
// which writes schedule_definitions + event_log in one tx for exactly-once
// fire semantics. That is the only cross-table tx in the codebase and it is
// fully enclosed in this package — business logic never sees it.
type ScheduleRepository interface {
	Create(ctx context.Context, schedule *models.ScheduleDefinition) error
	CreateBatch(ctx context.Context, scheds []*models.ScheduleDefinition) error
	ListByWorkflow(
		ctx context.Context,
		workflowName string,
		workflowVersion int,
	) ([]*models.ScheduleDefinition, error)

	ActivateByWorkflow(
		ctx context.Context,
		workflowName string,
		workflowVersion int,
		tenantID, partitionID string,
		fires []ScheduleActivation,
	) error

	DeactivateByWorkflow(ctx context.Context, workflowName, tenantID, partitionID string) error

	ClaimAndFireBatch(
		ctx context.Context,
		plan SchedulePlanFn,
		now time.Time,
		limit int,
	) (fired int, firedByTenant map[string]int, err error)

	// BacklogSeconds returns the age (in seconds) of the oldest schedule whose
	// next_fire_at is in the past and that is still active + not deleted. Returns
	// 0 if no rows are due.
	//
	// This is the single most important scaling signal: operators watch this to
	// know when fires are arriving faster than pods drain them.
	BacklogSeconds(ctx context.Context) (float64, error)

	Pool() pool.Pool
}

type scheduleRepository struct {
	datastore.BaseRepository[*models.ScheduleDefinition]
	p pool.Pool
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
		p: dbPool,
	}
}

func (r *scheduleRepository) Pool() pool.Pool { return r.p }

func (r *scheduleRepository) Create(
	ctx context.Context,
	schedule *models.ScheduleDefinition,
) error {
	return r.BaseRepository.Create(ctx, schedule)
}

// CreateBatch inserts all schedules in a single atomic transaction.
// If any row violates a constraint (e.g. idx_sd_workflow_unique) the entire
// batch is rolled back — no partial inserts.
func (r *scheduleRepository) CreateBatch(
	ctx context.Context,
	scheds []*models.ScheduleDefinition,
) error {
	if len(scheds) == 0 {
		return nil
	}
	db := r.p.DB(ctx, false)
	if err := db.Transaction(func(tx *gorm.DB) error {
		return tx.Create(scheds).Error
	}); err != nil {
		return fmt.Errorf("create schedule batch: %w", err)
	}
	return nil
}

// ListByWorkflow returns all non-deleted schedules for the given workflow name and version,
// ordered by name. Tenant + partition filtering is applied automatically by pool.DB via
// the TenancyPartition scope (same idiom as WorkflowDefinitionRepository.ListPage).
func (r *scheduleRepository) ListByWorkflow(
	ctx context.Context,
	workflowName string,
	workflowVersion int,
) ([]*models.ScheduleDefinition, error) {
	db := r.p.DB(ctx, true)

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

// ActivateByWorkflow atomically deactivates sibling workflow versions and
// activates the specified version's schedules in one transaction.
// tenantID and partitionID are explicit arguments so callers can operate on
// behalf of any tenant without relying on context claims (e.g. background jobs).
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

		// Step 1: deactivate all sibling versions (version != workflowVersion).
		if err := tx.Exec(`
			UPDATE schedule_definitions
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

		// Step 2: activate this version's schedules via VALUES-join UPDATE.
		// Positional order: modified_at, then the VALUES tuples, then tenant_id, partition_id.
		// IDs are stored as character varying — cast to ::text, not ::uuid.
		tuples := make([]string, 0, len(fires))
		orderedArgs := make([]any, 0, 1+activateArgsPerFire*len(fires)+activateArgsTrailingCount)
		orderedArgs = append(orderedArgs, now) // modified_at placeholder
		for i, f := range fires {
			if i == 0 {
				tuples = append(tuples, "(?::text, ?::timestamptz, ?::int)")
			} else {
				tuples = append(tuples, "(?, ?, ?)")
			}
			orderedArgs = append(orderedArgs, f.ID, f.NextFireAt, f.JitterSeconds)
		}
		orderedArgs = append(orderedArgs, tenantID, partitionID)

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

		return tx.Exec(sql, orderedArgs...).Error
	})
}

// DeactivateByWorkflow deactivates all non-deleted schedules for the given workflow
// (all versions), scoped strictly to the provided tenant_id and partition_id.
// No transaction needed — this is a single-statement UPDATE.
func (r *scheduleRepository) DeactivateByWorkflow(
	ctx context.Context,
	workflowName, tenantID, partitionID string,
) error {
	db := r.p.DB(ctx, false)
	now := time.Now().UTC()
	return db.Exec(`
		UPDATE schedule_definitions
		   SET active = false, next_fire_at = NULL, modified_at = ?
		 WHERE workflow_name = ?
		   AND tenant_id = ?
		   AND partition_id = ?
		   AND deleted_at IS NULL`,
		now, workflowName, tenantID, partitionID,
	).Error
}

// ClaimAndFireBatch is the fire hot path. One tx per sweep: SKIP LOCKED claim
// up to limit rows, pure-Go per-row plan, multi-row INSERT into event_log for
// rows that produced an event, VALUES-join UPDATE on schedule_definitions for
// all claimed rows (park rows get next_fire_at = NULL).
//
// This is the only cross-table tx in the codebase; it is fully enclosed here.
//
//nolint:gocognit // complexity is inherent in the three-phase atomic tx (claim, plan, fire); further extraction would hurt readability
func (r *scheduleRepository) ClaimAndFireBatch(
	ctx context.Context,
	plan SchedulePlanFn,
	now time.Time,
	limit int,
) (int, map[string]int, error) {
	db := r.p.DB(ctx, false)

	var fired int
	var firedByTenant map[string]int

	txErr := db.Transaction(func(tx *gorm.DB) error {
		// Claim up to limit due schedules under SKIP LOCKED.
		var batch []*models.ScheduleDefinition
		claimErr := tx.
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("active = ? AND deleted_at IS NULL AND next_fire_at IS NOT NULL AND next_fire_at <= ?", true, now).
			Order("next_fire_at ASC").
			Limit(limit).
			Find(&batch).Error
		if claimErr != nil {
			return fmt.Errorf("claim due batch: %w", claimErr)
		}
		if len(batch) == 0 {
			return nil
		}

		rows := make([]schedulePlanRow, 0, len(batch))
		eventsToInsert := make([]*models.EventLog, 0, len(batch))

		for _, sched := range batch {
			ev, nf, j, pErr := plan(ctx, sched)
			if pErr != nil {
				// Per-row error: skip this row. The batch still commits the rest.
				continue
			}
			rows = append(rows, schedulePlanRow{sched: sched, event: ev, next: nf, jit: j})
			if ev != nil {
				eventsToInsert = append(eventsToInsert, ev)
			}
		}

		if len(rows) == 0 {
			return nil
		}

		// Multi-row INSERT INTO event_log (only for rows with a non-nil event).
		if len(eventsToInsert) > 0 {
			insertErr := tx.CreateInBatches(eventsToInsert, eventLogInsertBatchSize).Error
			if insertErr != nil {
				return fmt.Errorf("batch insert event_log: %w", insertErr)
			}
		}

		// VALUES-join UPDATE for every claimed row.
		updateSQL, args := buildBatchUpdateSQL(rows, now)
		updateErr := tx.Exec(updateSQL, args...).Error
		if updateErr != nil {
			return fmt.Errorf("batch update schedules: %w", updateErr)
		}

		firedByTenant = make(map[string]int, len(rows))
		for _, pr := range rows {
			firedByTenant[pr.sched.TenantID]++
		}
		fired = len(rows)
		return nil
	})
	if txErr != nil {
		return 0, nil, txErr
	}
	return fired, firedByTenant, nil
}

// BacklogSeconds returns the age (in seconds) of the oldest schedule whose
// next_fire_at is in the past and that is still active + not deleted. Returns
// 0 if no rows are due.
//
// The query bypasses tenancy scoping (via SkipTenancyChecksOnClaims) so it
// reflects the pod-wide operational backlog across all tenants — not just one.
func (r *scheduleRepository) BacklogSeconds(ctx context.Context) (float64, error) {
	// Strip tenancy scope: backlog is a pod-wide operational signal, not per-tenant data.
	bgCtx := security.SkipTenancyChecksOnClaims(ctx)
	db := r.p.DB(bgCtx, true) // read replica is fine

	var lag sql.NullFloat64
	err := db.Raw(`
		SELECT EXTRACT(EPOCH FROM (now() - MIN(next_fire_at)))
		  FROM schedule_definitions
		 WHERE active = true
		   AND deleted_at IS NULL
		   AND next_fire_at IS NOT NULL
		   AND next_fire_at <= now()
	`).Scan(&lag).Error
	if err != nil {
		return 0, fmt.Errorf("scheduler backlog query: %w", err)
	}
	if !lag.Valid {
		return 0, nil
	}
	if lag.Float64 < 0 {
		return 0, nil // clock skew guard
	}
	return lag.Float64, nil
}
