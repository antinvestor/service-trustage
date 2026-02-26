package tests_test

import (
	"context"
	"testing"

	"github.com/pitabwire/frame/cache"
	"github.com/pitabwire/frame/datastore/pool"
	"github.com/pitabwire/frame/frametests"
	"github.com/pitabwire/frame/frametests/definition"
	"github.com/pitabwire/frame/frametests/deps/testpostgres"
	"github.com/pitabwire/frame/security"
	"github.com/stretchr/testify/suite"

	"github.com/antinvestor/service-trustage/apps/queue/service/business"
	"github.com/antinvestor/service-trustage/apps/queue/service/models"
	"github.com/antinvestor/service-trustage/apps/queue/service/repository"
)

const (
	testTenantID    = "test-tenant-001"
	testPartitionID = "test-partition-001"
)

type QueueSuite struct {
	frametests.FrameBaseTestSuite

	dbPool      pool.Pool
	rawCache    cache.RawCache
	defRepo     repository.QueueDefinitionRepository
	itemRepo    repository.QueueItemRepository
	counterRepo repository.QueueCounterRepository
	stats       business.QueueStatsService
	manager     business.QueueManager
}

func TestQueueSuite(t *testing.T) {
	suite.Run(t, new(QueueSuite))
}

func (s *QueueSuite) SetupSuite() {
	s.InitResourceFunc = func(_ context.Context) []definition.TestResource {
		return []definition.TestResource{
			testpostgres.New(),
		}
	}
	s.FrameBaseTestSuite.SetupSuite()

	ctx := context.Background()

	dsn := s.Resources()[0].GetDS(ctx)

	p := pool.NewPool(ctx)
	err := p.AddConnection(ctx,
		pool.WithConnection(string(dsn), false),
		pool.WithPreparedStatements(false),
	)
	s.Require().NoError(err, "connect to test database")

	db := p.DB(ctx, false)
	err = db.AutoMigrate(
		&models.QueueDefinition{},
		&models.QueueItem{},
		&models.QueueCounter{},
	)
	s.Require().NoError(err, "auto-migrate")

	s.dbPool = p
	s.rawCache = cache.NewInMemoryCache()

	s.defRepo = repository.NewQueueDefinitionRepository(p)
	s.itemRepo = repository.NewQueueItemRepository(p)
	s.counterRepo = repository.NewQueueCounterRepository(p)
	s.stats = business.NewQueueStatsService(s.itemRepo, s.counterRepo, s.rawCache, 30)
	s.manager = business.NewQueueManager(s.defRepo, s.itemRepo, s.counterRepo, s.stats)
}

func (s *QueueSuite) SetupTest() {
	// Use background context (not tenant-scoped) so TRUNCATE isn't affected by GORM tenant scoping.
	ctx := context.Background()
	db := s.dbPool.DB(ctx, false)
	db.Exec("TRUNCATE queue_definitions, queue_items, queue_counters CASCADE")
	_ = s.rawCache.Flush(ctx)
}

func (s *QueueSuite) TearDownSuite() {
	ctx := context.Background()
	if s.rawCache != nil {
		_ = s.rawCache.Close()
	}
	if s.dbPool != nil {
		s.dbPool.Close(ctx)
	}
	s.FrameBaseTestSuite.TearDownSuite()
}

func (s *QueueSuite) tenantCtx() context.Context {
	claims := &security.AuthenticationClaims{
		TenantID:    testTenantID,
		PartitionID: testPartitionID,
	}
	return claims.ClaimsToContext(context.Background())
}

// createQueue is a test helper that creates a queue definition.
func (s *QueueSuite) createQueue(name string, maxCapacity, priorityLevels int) *models.QueueDefinition {
	s.T().Helper()
	ctx := s.tenantCtx()
	def := &models.QueueDefinition{
		Name:           name,
		Active:         true,
		PriorityLevels: priorityLevels,
		MaxCapacity:    maxCapacity,
		SLAMinutes:     30,
	}
	err := s.manager.CreateQueue(ctx, def)
	s.Require().NoError(err)
	s.Require().NotEmpty(def.ID)
	return def
}

// createCounter is a test helper that creates and opens a counter.
func (s *QueueSuite) createAndOpenCounter(queueID, name, staffID string) *models.QueueCounter {
	s.T().Helper()
	ctx := s.tenantCtx()
	counter := &models.QueueCounter{
		QueueID: queueID,
		Name:    name,
	}
	err := s.manager.CreateCounter(ctx, counter)
	s.Require().NoError(err)

	err = s.manager.OpenCounter(ctx, counter.ID, staffID)
	s.Require().NoError(err)
	return counter
}
