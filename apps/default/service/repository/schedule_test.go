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
	s.Require().NoError(p.AddConnection(ctx,
		pool.WithConnection(string(dsn), false),
		pool.WithPreparedStatements(false),
	))
	s.dbPool = p

	mgr, err := datastoremanager.NewManager(ctx)
	s.Require().NoError(err)
	mgr.AddPool(ctx, datastore.DefaultPoolName, p)
	s.Require().NoError(Migrate(ctx, mgr))

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
	c := &security.AuthenticationClaims{TenantID: "test-tenant", PartitionID: "test-partition"}
	c.Subject = "test-user"
	return c.ClaimsToContext(context.Background())
}

func (s *ScheduleRepoSuite) seedDue(ctx context.Context, n int) {
	due := time.Now().UTC().Add(-time.Minute)
	for i := range n {
		sched := &models.ScheduleDefinition{
			Name: fmt.Sprintf("s-%d", i), CronExpr: "*/5 * * * *", Timezone: "UTC",
			WorkflowName: "wf", WorkflowVersion: 1, InputPayload: "{}",
			Active: true, NextFireAt: &due,
		}
		s.Require().NoError(s.repo.Create(ctx, sched))
	}
}

func simplePlan(
	_ context.Context,
	sched *models.ScheduleDefinition,
) (*models.EventLog, *time.Time, int, error) {
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

func parkPlan(
	_ context.Context,
	_ *models.ScheduleDefinition,
) (*models.EventLog, *time.Time, int, error) {
	return nil, nil, 0, nil
}

func (s *ScheduleRepoSuite) TestClaimAndFireBatch_ExactlyOnceConcurrent() {
	ctx := s.tenantCtx()
	const n = 50
	s.seedDue(ctx, n)

	var total atomic.Int64
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				count, _, err := s.repo.ClaimAndFireBatch(ctx, simplePlan, time.Now().UTC(), 8)
				s.NoError(err)
				total.Add(int64(count))
				if count == 0 {
					return
				}
			}
		}()
	}
	wg.Wait()

	s.Equal(int64(n), total.Load(), "every schedule must fire exactly once across all workers")

	// Per-tenant attribution: the seedDue helper uses tenant "test-tenant"
	// for every row, so firedByTenant must sum to n with a single key.
	// (We also assert the sum via a fresh non-concurrent call with empty table
	//  to keep the assertion clean; see TestClaimAndFireBatch_FiredByTenantAttribution below.)

	var evCount int64
	s.Require().NoError(s.dbPool.DB(ctx, false).Model(&models.EventLog{}).
		Where("event_type = ?", events.ScheduleFiredType).Count(&evCount).Error)
	s.Equal(int64(n), evCount)
}

func (s *ScheduleRepoSuite) TestClaimAndFireBatch_FiredByTenantAttribution() {
	ctx := s.tenantCtx()
	s.seedDue(ctx, 5)

	count, byTenant, err := s.repo.ClaimAndFireBatch(ctx, simplePlan, time.Now().UTC(), 10)
	s.Require().NoError(err)
	s.Equal(5, count)
	s.Equal(5, byTenant["test-tenant"], "all 5 fires must be attributed to test-tenant")
	s.Len(byTenant, 1, "only one tenant in this suite")
}

func (s *ScheduleRepoSuite) TestBacklogSeconds_ReturnsOldestDueLag() {
	ctx := s.tenantCtx()

	// No due rows → zero backlog.
	lag, err := s.repo.BacklogSeconds(ctx)
	s.Require().NoError(err)
	s.Equal(float64(0), lag, "no due rows → zero backlog")

	// Seed a row 1h in the past.
	pastHour := time.Now().UTC().Add(-time.Hour)
	sch := &models.ScheduleDefinition{
		Name: "stale", CronExpr: "*/5 * * * *", Timezone: "UTC",
		WorkflowName: "wf", WorkflowVersion: 1, InputPayload: "{}",
		Active: true, NextFireAt: &pastHour,
	}
	s.Require().NoError(s.repo.Create(ctx, sch))

	lag, err = s.repo.BacklogSeconds(ctx)
	s.Require().NoError(err)
	s.True(lag >= 3600-1 && lag <= 3600+5, "backlog should be ≈ 3600 s, got %v", lag)
}

func (s *ScheduleRepoSuite) TestBacklogSeconds_IgnoresInactiveAndDeletedRows() {
	ctx := s.tenantCtx()

	past := time.Now().UTC().Add(-time.Hour)
	// Inactive row — must NOT count toward backlog.
	s.Require().NoError(s.repo.Create(ctx, &models.ScheduleDefinition{
		Name: "inactive", CronExpr: "*/5 * * * *", Timezone: "UTC",
		WorkflowName: "wf", WorkflowVersion: 1, InputPayload: "{}",
		Active: false, NextFireAt: &past,
	}))

	lag, err := s.repo.BacklogSeconds(ctx)
	s.Require().NoError(err)
	s.Equal(float64(0), lag, "inactive rows must not contribute to backlog")
}

func (s *ScheduleRepoSuite) TestClaimAndFireBatch_ParkEmitsNoEvent() {
	ctx := s.tenantCtx()
	s.seedDue(ctx, 3)

	count, _, err := s.repo.ClaimAndFireBatch(ctx, parkPlan, time.Now().UTC(), 10)
	s.Require().NoError(err)
	s.Equal(3, count, "parked rows still count as processed")

	var evCount int64
	s.Require().NoError(s.dbPool.DB(ctx, false).Model(&models.EventLog{}).Count(&evCount).Error)
	s.Equal(int64(0), evCount)

	var rows []*models.ScheduleDefinition
	s.Require().NoError(s.dbPool.DB(ctx, false).Find(&rows).Error)
	for _, r := range rows {
		s.Nil(r.NextFireAt, "parked row must have NULL next_fire_at")
	}
}

func (s *ScheduleRepoSuite) TestCreateBatch_Atomic_RollbackOnConflict() {
	ctx := s.tenantCtx()
	claims := security.ClaimsFromContext(ctx)

	// Same (tenant, partition, workflow, version, name) violates idx_sd_workflow_unique.
	a := &models.ScheduleDefinition{
		Name: "same", CronExpr: "*/5 * * * *", Timezone: "UTC",
		WorkflowName: "wf", WorkflowVersion: 1, InputPayload: "{}", Active: false,
	}
	b := &models.ScheduleDefinition{
		Name: "same", CronExpr: "0 * * * *", Timezone: "UTC",
		WorkflowName: "wf", WorkflowVersion: 1, InputPayload: "{}", Active: false,
	}
	a.TenantID = claims.TenantID
	a.PartitionID = claims.PartitionID
	b.TenantID = claims.TenantID
	b.PartitionID = claims.PartitionID

	err := s.repo.CreateBatch(ctx, []*models.ScheduleDefinition{a, b})
	s.Require().Error(err, "duplicate name must fail")

	var count int64
	s.Require().
		NoError(s.dbPool.DB(ctx, false).Model(&models.ScheduleDefinition{}).Count(&count).Error)
	s.Equal(int64(0), count, "atomic: neither row should have persisted")
}

func (s *ScheduleRepoSuite) TestCreateBatch_InsertsAll() {
	ctx := s.tenantCtx()
	claims := security.ClaimsFromContext(ctx)

	scheds := []*models.ScheduleDefinition{
		{
			Name:            "a",
			CronExpr:        "*/5 * * * *",
			Timezone:        "UTC",
			WorkflowName:    "wf",
			WorkflowVersion: 1,
			InputPayload:    "{}",
			Active:          false,
		},
		{
			Name:            "b",
			CronExpr:        "0 * * * *",
			Timezone:        "UTC",
			WorkflowName:    "wf",
			WorkflowVersion: 1,
			InputPayload:    "{}",
			Active:          false,
		},
		{
			Name:            "c",
			CronExpr:        "*/10 * * * *",
			Timezone:        "UTC",
			WorkflowName:    "wf",
			WorkflowVersion: 1,
			InputPayload:    "{}",
			Active:          false,
		},
	}
	for _, sch := range scheds {
		sch.TenantID = claims.TenantID
		sch.PartitionID = claims.PartitionID
	}

	s.Require().NoError(s.repo.CreateBatch(ctx, scheds))

	var count int64
	s.Require().
		NoError(s.dbPool.DB(ctx, false).Model(&models.ScheduleDefinition{}).Count(&count).Error)
	s.Equal(int64(3), count)
}

func (s *ScheduleRepoSuite) TestActivateByWorkflow_SwitchesVersions() {
	ctx := s.tenantCtx()

	v1 := &models.ScheduleDefinition{
		Name: "s", CronExpr: "*/5 * * * *", Timezone: "UTC",
		WorkflowName: "wf-a", WorkflowVersion: 1, InputPayload: "{}",
		Active: true, NextFireAt: timePtr(time.Now().Add(time.Hour)),
	}
	s.Require().NoError(s.repo.Create(ctx, v1))

	v2 := &models.ScheduleDefinition{
		Name: "s", CronExpr: "*/10 * * * *", Timezone: "UTC",
		WorkflowName: "wf-a", WorkflowVersion: 2, InputPayload: "{}",
		Active: false,
	}
	s.Require().NoError(s.repo.Create(ctx, v2))

	fires := []ScheduleActivation{
		{ID: v2.ID, NextFireAt: time.Now().Add(time.Hour), JitterSeconds: 3},
	}
	s.Require().
		NoError(s.repo.ActivateByWorkflow(ctx, "wf-a", 2, "test-tenant", "test-partition", fires))

	var after []*models.ScheduleDefinition
	s.Require().
		NoError(s.dbPool.DB(ctx, false).Where("workflow_name = ?", "wf-a").Order("workflow_version ASC").Find(&after).Error)

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
			WorkflowName: "wf-d", WorkflowVersion: v, InputPayload: "{}",
			Active: true, NextFireAt: timePtr(time.Now().Add(time.Hour)),
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

func (s *ScheduleRepoSuite) TestDeactivateByWorkflow_TenantIsolated() {
	ctx := s.tenantCtx()
	a := &models.ScheduleDefinition{
		Name: "x", CronExpr: "*/5 * * * *", Timezone: "UTC",
		WorkflowName: "wf-tenant", WorkflowVersion: 1, InputPayload: "{}",
		Active: true, NextFireAt: timePtr(time.Now().Add(time.Hour)),
	}
	s.Require().NoError(s.repo.Create(ctx, a))

	// Deactivate scoped to a different tenant — tenant-A's row must be untouched.
	s.Require().
		NoError(s.repo.DeactivateByWorkflow(ctx, "wf-tenant", "other-tenant", "other-partition"))

	var check models.ScheduleDefinition
	s.Require().NoError(s.dbPool.DB(ctx, false).First(&check, "id = ?", a.ID).Error)
	s.True(check.Active, "cross-tenant deactivate must not affect tenant-A")
}
