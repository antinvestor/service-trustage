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

	dbPool pool.Pool
	repo   ScheduleRepository
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
	claims := &security.AuthenticationClaims{
		TenantID:    "test-tenant",
		PartitionID: "test-partition",
	}
	claims.Subject = "test-user"
	return claims.ClaimsToContext(context.Background())
}

func (s *ScheduleRepoSuite) seedDueSchedules(ctx context.Context, n int) {
	due := time.Now().UTC().Add(-time.Minute)
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
	}
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

	var all []*models.ScheduleDefinition
	s.Require().NoError(s.dbPool.DB(ctx, false).Find(&all).Error)
	s.Len(all, n)
	for _, r := range all {
		s.NotNil(r.LastFiredAt, "schedule %s missing last_fired_at", r.Name)
		s.NotNil(r.NextFireAt, "schedule %s missing next_fire_at", r.Name)
		s.True(r.NextFireAt.After(time.Now()), "next_fire_at should be in the future")
	}

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
