//nolint:testpackage // package-local repository tests exercise unexported helpers intentionally.
package repository

import (
	"context"
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

	"github.com/antinvestor/service-trustage/apps/queue/service/models"
)

type RepositorySuite struct {
	frametests.FrameBaseTestSuite

	dbPool      pool.Pool
	defRepo     QueueDefinitionRepository
	itemRepo    QueueItemRepository
	counterRepo QueueCounterRepository
}

func TestRepositorySuite(t *testing.T) {
	suite.Run(t, new(RepositorySuite))
}

func (s *RepositorySuite) SetupSuite() {
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

	db := p.DB(ctx, false)
	s.Require().NoError(db.AutoMigrate(
		&models.QueueDefinition{},
		&models.QueueItem{},
		&models.QueueCounter{},
	))

	s.dbPool = p
	s.defRepo = NewQueueDefinitionRepository(p)
	s.itemRepo = NewQueueItemRepository(p)
	s.counterRepo = NewQueueCounterRepository(p)
}

func (s *RepositorySuite) SetupTest() {
	ctx := context.Background()
	s.Require().NoError(s.dbPool.DB(ctx, false).Exec(
		"TRUNCATE queue_definitions, queue_items, queue_counters CASCADE",
	).Error)
}

func (s *RepositorySuite) TearDownSuite() {
	if s.dbPool != nil {
		s.dbPool.Close(context.Background())
	}
	s.FrameBaseTestSuite.TearDownSuite()
}

func (s *RepositorySuite) tenantCtx() context.Context {
	claims := &security.AuthenticationClaims{
		TenantID:    "test-tenant",
		PartitionID: "test-partition",
	}
	claims.Subject = "test-user"
	return claims.ClaimsToContext(context.Background())
}

func (s *RepositorySuite) TestQueueDefinitionRepository_CRUDAndList() {
	ctx := s.tenantCtx()

	defs := []*models.QueueDefinition{
		{Name: "priority", Active: true, PriorityLevels: 3, MaxCapacity: 10, SLAMinutes: 30, Config: "{}"},
		{Name: "archived", Active: true, PriorityLevels: 2, MaxCapacity: 0, SLAMinutes: 15, Config: "{}"},
	}
	for _, def := range defs {
		s.Require().NoError(s.defRepo.Create(ctx, def))
	}
	s.Require().NoError(s.dbPool.DB(ctx, false).Model(&models.QueueDefinition{}).
		Where("id = ?", defs[1].ID).UpdateColumn("active", false).Error)

	byName, err := s.defRepo.GetByName(ctx, "priority")
	s.Require().NoError(err)
	s.Equal(defs[0].ID, byName.ID)

	active, err := s.defRepo.List(ctx, true)
	s.Require().NoError(err)
	s.Len(active, 1)

	defs[0].Description = "updated"
	s.Require().NoError(s.defRepo.Update(ctx, defs[0]))
	got, err := s.defRepo.GetByID(ctx, defs[0].ID)
	s.Require().NoError(err)
	s.Equal("updated", got.Description)

	s.Require().NoError(s.defRepo.SoftDelete(ctx, defs[1]))
	items, err := s.defRepo.List(ctx, false)
	s.Require().NoError(err)
	s.Len(items, 1)
}

func (s *RepositorySuite) TestQueueItemRepository_QueriesAndAggregates() {
	ctx := s.tenantCtx()
	def := &models.QueueDefinition{Name: "main", Active: true, PriorityLevels: 3, Config: "{}"}
	s.Require().NoError(s.defRepo.Create(ctx, def))

	now := time.Now()
	items := []*models.QueueItem{
		{
			QueueID:    def.ID,
			Priority:   3,
			Status:     models.ItemStatusWaiting,
			TicketNo:   "VIP-1",
			Category:   "vip",
			Metadata:   "{}",
			JoinedAt:   now.Add(-15 * time.Minute),
			CustomerID: "customer-1",
		},
		{
			QueueID:    def.ID,
			Priority:   1,
			Status:     models.ItemStatusWaiting,
			TicketNo:   "STD-1",
			Category:   "standard",
			Metadata:   "{}",
			JoinedAt:   now.Add(-10 * time.Minute),
			CustomerID: "customer-2",
		},
		{
			QueueID:      def.ID,
			Priority:     2,
			Status:       models.ItemStatusCompleted,
			TicketNo:     "DONE-1",
			Metadata:     "{}",
			JoinedAt:     now.Add(-30 * time.Minute),
			CalledAt:     timePtr(now.Add(-20 * time.Minute)),
			ServiceStart: timePtr(now.Add(-19 * time.Minute)),
			ServiceEnd:   timePtr(now.Add(-18 * time.Minute)),
		},
		{
			QueueID:    def.ID,
			Priority:   1,
			Status:     models.ItemStatusCancelled,
			TicketNo:   "CXL-1",
			Metadata:   "{}",
			JoinedAt:   now.Add(-5 * time.Minute),
			CustomerID: "customer-4",
		},
	}
	for _, item := range items {
		s.Require().NoError(s.itemRepo.Create(ctx, item))
	}

	next, err := s.itemRepo.FindNextWaiting(ctx, def.ID, []string{"vip"})
	s.Require().NoError(err)
	s.Equal(items[0].ID, next.ID)

	waiting, err := s.itemRepo.ListWaiting(ctx, def.ID, 10, 0)
	s.Require().NoError(err)
	s.Len(waiting, 2)
	s.Equal(items[0].ID, waiting[0].ID)

	pos, err := s.itemRepo.GetPosition(ctx, items[1])
	s.Require().NoError(err)
	s.Equal(2, pos)

	waitingCount, err := s.itemRepo.CountByStatus(ctx, def.ID, models.ItemStatusWaiting)
	s.Require().NoError(err)
	s.Equal(int64(2), waitingCount)

	forUpdate, err := s.itemRepo.CountWaitingForUpdate(ctx, def.ID)
	s.Require().NoError(err)
	s.Equal(int64(2), forUpdate)

	avgWait, err := s.itemRepo.AvgWaitMinutes(ctx, def.ID, now.Add(-24*time.Hour))
	s.Require().NoError(err)
	s.Greater(avgWait, 0.0)

	longestWait, err := s.itemRepo.LongestWaitMinutes(ctx, def.ID)
	s.Require().NoError(err)
	s.Greater(longestWait, 0.0)

	cancelled, err := s.itemRepo.CountByStatusSince(ctx, def.ID, models.ItemStatusCancelled, now.Add(-24*time.Hour))
	s.Require().NoError(err)
	s.Equal(int64(1), cancelled)

	items[1].Status = models.ItemStatusServing
	s.Require().NoError(s.itemRepo.Update(ctx, items[1]))
	updated, err := s.itemRepo.GetByID(ctx, items[1].ID)
	s.Require().NoError(err)
	s.Equal(models.ItemStatusServing, updated.Status)
}

func (s *RepositorySuite) TestQueueCounterRepository_CRUDAndCounts() {
	ctx := s.tenantCtx()
	def := &models.QueueDefinition{Name: "service", Active: true, PriorityLevels: 3, Config: "{}"}
	s.Require().NoError(s.defRepo.Create(ctx, def))

	counters := []*models.QueueCounter{
		{QueueID: def.ID, Name: "Desk A", Status: models.CounterStatusOpen, Categories: `[]`},
		{QueueID: def.ID, Name: "Desk B", Status: models.CounterStatusClosed, Categories: `[]`},
	}
	for _, counter := range counters {
		s.Require().NoError(s.counterRepo.Create(ctx, counter))
	}

	listed, err := s.counterRepo.ListByQueueID(ctx, def.ID)
	s.Require().NoError(err)
	s.Len(listed, 2)

	openCount, err := s.counterRepo.CountOpen(ctx, def.ID)
	s.Require().NoError(err)
	s.Equal(int64(1), openCount)

	counters[1].Status = models.CounterStatusOpen
	s.Require().NoError(s.counterRepo.Update(ctx, counters[1]))
	openCount, err = s.counterRepo.CountOpen(ctx, def.ID)
	s.Require().NoError(err)
	s.Equal(int64(2), openCount)

	s.Require().NoError(s.counterRepo.SoftDelete(ctx, counters[0]))
	listed, err = s.counterRepo.ListByQueueID(ctx, def.ID)
	s.Require().NoError(err)
	s.Len(listed, 1)

	gotCounter, err := s.counterRepo.GetByID(ctx, counters[1].ID)
	s.Require().NoError(err)
	s.Equal(counters[1].ID, gotCounter.ID)
}

func (s *RepositorySuite) TestQueueRepository_MigrateAndHelpers() {
	ctx := s.tenantCtx()

	manager, err := datastoremanager.NewManager(ctx)
	s.Require().NoError(err)
	manager.AddPool(ctx, datastore.DefaultPoolName, s.dbPool)

	s.Require().NoError(Migrate(ctx, manager))

	db := s.dbPool.DB(ctx, false)
	s.True(db.Migrator().HasTable(&models.QueueDefinition{}))
	s.True(db.Migrator().HasTable(&models.QueueItem{}))
	s.True(db.Migrator().HasTable(&models.QueueCounter{}))

	for _, indexDef := range migrationIndexes() {
		for _, indexName := range indexDef.Names {
			s.True(db.Migrator().HasIndex(indexDef.Model, indexName), indexName)
		}
	}

	def := &models.QueueDefinition{Name: "pool-check", Active: true, PriorityLevels: 3, Config: "{}"}
	s.Require().NoError(s.defRepo.Create(ctx, def))
	item := &models.QueueItem{
		QueueID:  def.ID,
		Priority: 1,
		Status:   models.ItemStatusWaiting,
		TicketNo: "POOL-1",
		Metadata: "{}",
		JoinedAt: time.Now(),
	}
	s.Require().NoError(s.itemRepo.Create(ctx, item))

	s.NotNil(s.itemRepo.Pool())

	defs, err := s.defRepo.List(ctx, false)
	s.Require().NoError(err)
	s.Require().NotEmpty(defs)
	s.Require().NoError(s.defRepo.SoftDelete(ctx, defs[0]))

	s.Equal("queue_definitions", queueDefinitionIndexModel{}.TableName())
	s.Equal("queue_items", queueItemIndexModel{}.TableName())
	s.Equal("queue_counters", queueCounterIndexModel{}.TableName())
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func (s *RepositorySuite) TestMigrate_ReturnsErrorWhenPoolMissing() {
	ctx := context.Background()
	manager, err := datastoremanager.NewManager(ctx)
	s.Require().NoError(err)

	// No pool added — Migrate must return an error, not panic.
	err = Migrate(ctx, manager)
	s.Require().Error(err)
	s.Contains(err.Error(), "pool")
}
