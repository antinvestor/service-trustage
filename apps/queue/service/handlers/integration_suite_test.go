package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/pitabwire/frame/cache"
	"github.com/pitabwire/frame/datastore/pool"
	"github.com/pitabwire/frame/frametests"
	"github.com/pitabwire/frame/frametests/definition"
	"github.com/pitabwire/frame/frametests/deps/testpostgres"
	"github.com/pitabwire/frame/security"
	"github.com/stretchr/testify/suite"

	queueauthz "github.com/antinvestor/service-trustage/apps/queue/service/authz"
	"github.com/antinvestor/service-trustage/apps/queue/service/business"
	"github.com/antinvestor/service-trustage/apps/queue/service/models"
	"github.com/antinvestor/service-trustage/apps/queue/service/repository"
)

type HandlerSuite struct {
	frametests.FrameBaseTestSuite

	dbPool   pool.Pool
	rawCache cache.RawCache

	defRepo     repository.QueueDefinitionRepository
	itemRepo    repository.QueueItemRepository
	counterRepo repository.QueueCounterRepository
	stats       business.QueueStatsService
	manager     business.QueueManager
}

func TestHandlerSuite(t *testing.T) {
	suite.Run(t, new(HandlerSuite))
}

func (s *HandlerSuite) SetupSuite() {
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

	db := p.DB(ctx, false)
	s.Require().NoError(db.AutoMigrate(
		&models.QueueDefinition{},
		&models.QueueItem{},
		&models.QueueCounter{},
	))

	s.dbPool = p
	s.rawCache = cache.NewInMemoryCache()
	s.defRepo = repository.NewQueueDefinitionRepository(p)
	s.itemRepo = repository.NewQueueItemRepository(p)
	s.counterRepo = repository.NewQueueCounterRepository(p)
	s.stats = business.NewQueueStatsService(s.itemRepo, s.counterRepo, s.rawCache, 30)
	s.manager = business.NewQueueManager(s.defRepo, s.itemRepo, s.counterRepo, s.stats)
}

func (s *HandlerSuite) SetupTest() {
	ctx := context.Background()
	s.Require().NoError(s.dbPool.DB(ctx, false).Exec(
		"TRUNCATE queue_definitions, queue_items, queue_counters CASCADE",
	).Error)
	s.Require().NoError(s.rawCache.Flush(ctx))
}

func (s *HandlerSuite) TearDownSuite() {
	if s.rawCache != nil {
		_ = s.rawCache.Close()
	}
	if s.dbPool != nil {
		s.dbPool.Close(context.Background())
	}
	s.FrameBaseTestSuite.TearDownSuite()
}

func (s *HandlerSuite) tenantCtx() context.Context {
	claims := &security.AuthenticationClaims{TenantID: "test-tenant", PartitionID: "test-partition"}
	claims.Subject = "test-user"
	return claims.ClaimsToContext(context.Background())
}

type allowAllAuthz struct{}

func (a allowAllAuthz) CanQueueManage(_ context.Context) error   { return nil }
func (a allowAllAuthz) CanQueueView(_ context.Context) error     { return nil }
func (a allowAllAuthz) CanItemEnqueue(_ context.Context) error   { return nil }
func (a allowAllAuthz) CanQueueItemView(_ context.Context) error { return nil }
func (a allowAllAuthz) CanCounterManage(_ context.Context) error { return nil }
func (a allowAllAuthz) CanStatsView(_ context.Context) error     { return nil }

var _ queueauthz.Middleware = allowAllAuthz{}

func encodeBody(v any) *bytes.Reader {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return bytes.NewReader(data)
}
