//nolint:testpackage // package-local queue tests exercise unexported business helpers intentionally.
package business

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/pitabwire/frame/cache"
	"github.com/pitabwire/frame/datastore/pool"
	"github.com/pitabwire/frame/frametests"
	"github.com/pitabwire/frame/frametests/definition"
	"github.com/pitabwire/frame/frametests/deps/testpostgres"
	"github.com/pitabwire/frame/security"
	"github.com/stretchr/testify/suite"

	"github.com/antinvestor/service-trustage/apps/queue/service/models"
	"github.com/antinvestor/service-trustage/apps/queue/service/repository"
)

type QueueBusinessSuite struct {
	frametests.FrameBaseTestSuite

	dbPool      pool.Pool
	rawCache    cache.RawCache
	defRepo     repository.QueueDefinitionRepository
	itemRepo    repository.QueueItemRepository
	counterRepo repository.QueueCounterRepository
	stats       QueueStatsService
	manager     QueueManager
}

func TestQueueBusinessSuite(t *testing.T) {
	suite.Run(t, new(QueueBusinessSuite))
}

func (s *QueueBusinessSuite) SetupSuite() {
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
	s.rawCache = cache.NewInMemoryCache()
	s.defRepo = repository.NewQueueDefinitionRepository(p)
	s.itemRepo = repository.NewQueueItemRepository(p)
	s.counterRepo = repository.NewQueueCounterRepository(p)
	s.stats = NewQueueStatsService(s.itemRepo, s.counterRepo, s.rawCache, 30)
	s.manager = NewQueueManager(s.defRepo, s.itemRepo, s.counterRepo, s.stats)
}

func (s *QueueBusinessSuite) SetupTest() {
	ctx := context.Background()
	s.Require().NoError(s.dbPool.DB(ctx, false).Exec(
		"TRUNCATE queue_definitions, queue_items, queue_counters CASCADE",
	).Error)
	s.Require().NoError(s.rawCache.Flush(ctx))
}

func (s *QueueBusinessSuite) TearDownSuite() {
	ctx := context.Background()
	if s.rawCache != nil {
		_ = s.rawCache.Close()
	}
	if s.dbPool != nil {
		s.dbPool.Close(ctx)
	}
	s.FrameBaseTestSuite.TearDownSuite()
}

func (s *QueueBusinessSuite) tenantCtx() context.Context {
	claims := &security.AuthenticationClaims{
		TenantID:    "test-tenant",
		PartitionID: "test-partition",
	}
	claims.Subject = "test-user"
	return claims.ClaimsToContext(context.Background())
}

func (s *QueueBusinessSuite) createQueue(name string, maxCapacity, priorityLevels int) *models.QueueDefinition {
	s.T().Helper()
	def := &models.QueueDefinition{
		Name:           name,
		Active:         true,
		PriorityLevels: priorityLevels,
		MaxCapacity:    maxCapacity,
		SLAMinutes:     30,
	}
	s.Require().NoError(s.manager.CreateQueue(s.tenantCtx(), def))
	return def
}

func (s *QueueBusinessSuite) createCounter(queueID, name, staffID string) *models.QueueCounter {
	s.T().Helper()
	counter := &models.QueueCounter{QueueID: queueID, Name: name}
	ctx := s.tenantCtx()
	s.Require().NoError(s.manager.CreateCounter(ctx, counter))
	s.Require().NoError(s.manager.OpenCounter(ctx, counter.ID, staffID))
	return counter
}

func (s *QueueBusinessSuite) TestQueueManager_DefinitionLifecycle() {
	ctx := s.tenantCtx()

	queue := &models.QueueDefinition{Name: "priority-support", PriorityLevels: 4, SLAMinutes: 45}
	s.Require().NoError(s.manager.CreateQueue(ctx, queue))
	s.NotEmpty(queue.ID)
	s.Equal("{}", queue.Config)

	loaded, err := s.manager.GetQueue(ctx, queue.ID)
	s.Require().NoError(err)
	s.Equal(queue.Name, loaded.Name)

	queue.Description = "updated description"
	s.Require().NoError(s.manager.UpdateQueue(ctx, queue))

	items, err := s.manager.ListQueues(ctx, false)
	s.Require().NoError(err)
	s.Len(items, 1)
	s.Equal("updated description", items[0].Description)

	s.Require().NoError(s.manager.DeleteQueue(ctx, queue.ID))
	_, err = s.manager.GetQueue(ctx, queue.ID)
	s.Require().Error(err)
	s.ErrorIs(err, ErrQueueNotFound)
}

func (s *QueueBusinessSuite) TestQueueManager_EnqueueOrderingCapacityAndPosition() {
	ctx := s.tenantCtx()
	queue := s.createQueue("frontdesk", 2, 3)

	cases := []struct {
		name       string
		item       *models.QueueItem
		wantPrio   int
		wantTicket bool
	}{
		{
			name:       "priority below minimum clamps to one",
			item:       &models.QueueItem{QueueID: queue.ID, Priority: 0, Metadata: "{}"},
			wantPrio:   1,
			wantTicket: true,
		},
		{
			name:       "priority above configured levels clamps to max",
			item:       &models.QueueItem{QueueID: queue.ID, Priority: 9, Metadata: "{}"},
			wantPrio:   3,
			wantTicket: true,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			item := tc.item
			s.Require().NoError(s.manager.Enqueue(ctx, item))
			s.Equal(tc.wantPrio, item.Priority)
			if tc.wantTicket {
				s.NotEmpty(item.TicketNo)
			}
			s.Equal(models.ItemStatusWaiting, item.Status)
		})
	}

	items, err := s.manager.ListWaitingItems(ctx, queue.ID, 10, 0)
	s.Require().NoError(err)
	s.Len(items, 2)
	s.Equal(3, items[0].Priority)
	s.Equal(1, items[1].Priority)

	pos, err := s.manager.GetItemPosition(ctx, items[1].ID)
	s.Require().NoError(err)
	s.Equal(2, pos)

	err = s.manager.Enqueue(ctx, &models.QueueItem{QueueID: queue.ID, Priority: 1})
	s.Require().ErrorIs(err, ErrQueueFull)
}

func (s *QueueBusinessSuite) TestQueueManager_CounterServiceLifecycle() {
	ctx := s.tenantCtx()
	queue := s.createQueue("service-flow", 0, 3)
	counter := s.createCounter(queue.ID, "Desk 1", "staff-1")

	first := &models.QueueItem{QueueID: queue.ID, Priority: 3}
	second := &models.QueueItem{QueueID: queue.ID, Priority: 1}
	s.Require().NoError(s.manager.Enqueue(ctx, first))
	s.Require().NoError(s.manager.Enqueue(ctx, second))

	next, err := s.manager.CallNext(ctx, counter.ID)
	s.Require().NoError(err)
	s.Equal(first.ID, next.ID)

	s.Require().NoError(s.manager.BeginService(ctx, counter.ID))
	s.Require().NoError(s.manager.CompleteService(ctx, counter.ID))

	served, err := s.manager.GetItem(ctx, first.ID)
	s.Require().NoError(err)
	s.Equal(models.ItemStatusCompleted, served.Status)
}

func (s *QueueBusinessSuite) TestQueueManager_NoShowRequeueTransferAndClose() {
	ctx := s.tenantCtx()
	source := s.createQueue("source", 0, 3)
	target := s.createQueue("target", 0, 3)
	counter := s.createCounter(source.ID, "Desk 2", "staff-2")

	item := &models.QueueItem{QueueID: source.ID, Priority: 2}
	s.Require().NoError(s.manager.Enqueue(ctx, item))
	_, err := s.manager.CallNext(ctx, counter.ID)
	s.Require().NoError(err)

	s.Require().NoError(s.manager.NoShowItem(ctx, item.ID))
	noShow, err := s.manager.GetItem(ctx, item.ID)
	s.Require().NoError(err)
	s.Equal(models.ItemStatusNoShow, noShow.Status)

	s.Require().NoError(s.manager.RequeueItem(ctx, item.ID))
	requeued, err := s.manager.GetItem(ctx, item.ID)
	s.Require().NoError(err)
	s.Equal(models.ItemStatusWaiting, requeued.Status)

	s.Require().NoError(s.manager.TransferItem(ctx, item.ID, target.ID))
	transferred, err := s.manager.GetItem(ctx, item.ID)
	s.Require().NoError(err)
	s.Equal(target.ID, transferred.QueueID)

	_, err = s.manager.CallNext(ctx, counter.ID)
	s.Require().ErrorIs(err, ErrNoWaitingItems)

	s.Require().NoError(s.manager.CloseCounter(ctx, counter.ID))
	closed, err := s.manager.GetCounter(ctx, counter.ID)
	s.Require().NoError(err)
	s.Equal(models.CounterStatusClosed, closed.Status)
}

func (s *QueueBusinessSuite) TestQueueManager_CancelPauseListAndValidationErrors() {
	ctx := s.tenantCtx()
	queue := s.createQueue("ops", 0, 3)
	counter := s.createCounter(queue.ID, "Desk Ops", "staff-ops")

	item := &models.QueueItem{QueueID: queue.ID, Priority: 2}
	s.Require().NoError(s.manager.Enqueue(ctx, item))
	s.Require().NoError(s.manager.CancelItem(ctx, item.ID))

	cancelled, err := s.manager.GetItem(ctx, item.ID)
	s.Require().NoError(err)
	s.Equal(models.ItemStatusCancelled, cancelled.Status)

	counters, err := s.manager.ListCounters(ctx, queue.ID)
	s.Require().NoError(err)
	s.Len(counters, 1)

	s.Require().NoError(s.manager.PauseCounter(ctx, counter.ID))
	paused, err := s.manager.GetCounter(ctx, counter.ID)
	s.Require().NoError(err)
	s.Equal(models.CounterStatusPaused, paused.Status)

	_, err = s.manager.GetItemPosition(ctx, item.ID)
	s.Require().ErrorIs(err, ErrItemNotWaiting)

	s.Require().ErrorIs(s.manager.CancelItem(ctx, "missing-item"), ErrQueueItemNotFound)
	s.Require().ErrorIs(s.manager.PauseCounter(ctx, "missing-counter"), ErrCounterNotFound)
	_, err = s.manager.GetCounter(ctx, "missing-counter")
	s.Require().ErrorIs(err, ErrCounterNotFound)
	s.Require().ErrorIs(s.manager.RequeueItem(ctx, item.ID), ErrItemNotNoShow)
	s.Require().ErrorIs(s.manager.TransferItem(ctx, item.ID, queue.ID), ErrInvalidTransition)
}

func (s *QueueBusinessSuite) TestQueueManager_OpenCloseBeginComplete_ErrorCases() {
	ctx := s.tenantCtx()
	queue := s.createQueue("service-errors", 0, 3)
	counter := &models.QueueCounter{QueueID: queue.ID, Name: "Desk Fail"}
	s.Require().NoError(s.manager.CreateCounter(ctx, counter))

	_, err := s.manager.CallNext(ctx, counter.ID)
	s.Require().ErrorIs(err, ErrCounterNotOpen)

	s.Require().NoError(s.manager.OpenCounter(ctx, counter.ID, "staff-1"))
	s.Require().ErrorIs(s.manager.OpenCounter(ctx, counter.ID, "staff-2"), ErrInvalidTransition)

	item := &models.QueueItem{QueueID: queue.ID, Priority: 1}
	s.Require().NoError(s.manager.Enqueue(ctx, item))
	_, err = s.manager.CallNext(ctx, counter.ID)
	s.Require().NoError(err)

	_, err = s.manager.CallNext(ctx, counter.ID)
	s.Require().ErrorIs(err, ErrCounterBusy)

	s.Require().NoError(s.manager.BeginService(ctx, counter.ID))
	s.Require().NoError(s.manager.CompleteService(ctx, counter.ID))
	s.Require().ErrorIs(s.manager.BeginService(ctx, counter.ID), ErrCounterNotServing)
	s.Require().ErrorIs(s.manager.CompleteService(ctx, counter.ID), ErrCounterNotServing)
}

func (s *QueueBusinessSuite) TestQueueStatsService_ComputesCachesAndInvalidates() {
	ctx := s.tenantCtx()
	queue := s.createQueue("metrics", 0, 2)
	counter := s.createCounter(queue.ID, "Desk 9", "staff-9")

	now := time.Now().Add(-10 * time.Minute)
	waiting := &models.QueueItem{
		QueueID:  queue.ID,
		Priority: 1,
		Status:   models.ItemStatusWaiting,
		TicketNo: "WAIT-1",
		JoinedAt: now,
		Metadata: "{}",
	}
	completed := &models.QueueItem{
		QueueID:      queue.ID,
		Priority:     1,
		Status:       models.ItemStatusCompleted,
		TicketNo:     "DONE-1",
		JoinedAt:     now.Add(-5 * time.Minute),
		CalledAt:     timePtr(now),
		ServiceStart: timePtr(now.Add(1 * time.Minute)),
		ServiceEnd:   timePtr(now.Add(2 * time.Minute)),
		Metadata:     "{}",
	}
	cancelled := &models.QueueItem{
		QueueID:  queue.ID,
		Priority: 1,
		Status:   models.ItemStatusCancelled,
		TicketNo: "CXL-1",
		JoinedAt: now,
		Metadata: "{}",
	}
	noShow := &models.QueueItem{
		QueueID:  queue.ID,
		Priority: 1,
		Status:   models.ItemStatusNoShow,
		TicketNo: "NS-1",
		JoinedAt: now,
		Metadata: "{}",
	}

	for _, item := range []*models.QueueItem{waiting, completed, cancelled, noShow} {
		s.Require().NoError(s.itemRepo.Create(ctx, item))
	}

	stats, err := s.stats.GetStats(ctx, queue.ID)
	s.Require().NoError(err)
	s.Equal(int64(1), stats.TotalWaiting)
	s.Equal(int64(1), stats.TodayServed)
	s.Equal(int64(1), stats.TodayCancelled)
	s.Equal(int64(1), stats.TodayNoShow)
	s.Equal(int64(1), stats.OpenCounters)
	s.GreaterOrEqual(stats.LongestWait, 0.0)

	s.Require().NoError(s.dbPool.DB(ctx, false).
		Model(&models.QueueCounter{}).
		Where("id = ?", counter.ID).
		UpdateColumn("status", models.CounterStatusClosed).Error)

	cached, err := s.stats.GetStats(ctx, queue.ID)
	s.Require().NoError(err)
	s.Equal(int64(1), cached.OpenCounters)

	statsSvc := s.stats.(*queueStatsService)
	statsSvc.lastInvalidateMu.Lock()
	statsSvc.lastInvalidate[queue.ID] = time.Now().Add(-2 * minInvalidateInterval)
	statsSvc.lastInvalidateMu.Unlock()
	s.stats.InvalidateCache(ctx, queue.ID)
	refreshed, err := s.stats.GetStats(ctx, queue.ID)
	s.Require().NoError(err)
	s.Equal(int64(0), refreshed.OpenCounters)
	s.True(strings.HasPrefix(statsCacheKey(queue.ID), "queue:stats:"))
}

func timePtr(t time.Time) *time.Time {
	return &t
}
