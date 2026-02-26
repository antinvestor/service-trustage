package tests_test

import (
	"sync"
	"time"

	"github.com/antinvestor/service-trustage/apps/queue/service/business"
	"github.com/antinvestor/service-trustage/apps/queue/service/models"
)

// --- Enqueue tests ---

func (s *QueueSuite) TestEnqueue_Basic() {
	ctx := s.tenantCtx()
	queue := s.createQueue("basic-queue", 0, 3)

	item := &models.QueueItem{
		QueueID:  queue.ID,
		Priority: 2,
		Category: "general",
	}
	err := s.manager.Enqueue(ctx, item)
	s.Require().NoError(err)
	s.NotEmpty(item.ID)
	s.NotEmpty(item.TicketNo)
	s.Equal(models.ItemStatusWaiting, item.Status)
	s.Equal(2, item.Priority)
}

func (s *QueueSuite) TestEnqueue_PriorityClamping() {
	ctx := s.tenantCtx()
	queue := s.createQueue("clamp-queue", 0, 3) // max priority = 3

	item := &models.QueueItem{QueueID: queue.ID, Priority: 10}
	err := s.manager.Enqueue(ctx, item)
	s.Require().NoError(err)
	s.Equal(3, item.Priority, "priority should be clamped to queue max")

	item2 := &models.QueueItem{QueueID: queue.ID, Priority: -5}
	err = s.manager.Enqueue(ctx, item2)
	s.Require().NoError(err)
	s.Equal(1, item2.Priority, "negative priority should be clamped to 1")
}

func (s *QueueSuite) TestEnqueue_CapacityEnforcement() {
	ctx := s.tenantCtx()
	queue := s.createQueue("capped-queue", 2, 3) // max 2 items

	item1 := &models.QueueItem{QueueID: queue.ID, Priority: 1}
	s.Require().NoError(s.manager.Enqueue(ctx, item1))

	item2 := &models.QueueItem{QueueID: queue.ID, Priority: 1}
	s.Require().NoError(s.manager.Enqueue(ctx, item2))

	item3 := &models.QueueItem{QueueID: queue.ID, Priority: 1}
	err := s.manager.Enqueue(ctx, item3)
	s.Require().Error(err)
	s.ErrorIs(err, business.ErrQueueFull, "third enqueue should fail with ErrQueueFull")
}

func (s *QueueSuite) TestEnqueue_InvalidQueue() {
	ctx := s.tenantCtx()
	item := &models.QueueItem{QueueID: "nonexistent-id", Priority: 1}
	err := s.manager.Enqueue(ctx, item)
	s.Require().Error(err)
	s.ErrorIs(err, business.ErrQueueNotFound)
}

// --- CallNext tests ---

func (s *QueueSuite) TestCallNext_PriorityOrdering() {
	ctx := s.tenantCtx()
	queue := s.createQueue("priority-queue", 0, 5)

	// Enqueue items with different priorities.
	low := &models.QueueItem{QueueID: queue.ID, Priority: 1, Category: "test"}
	s.Require().NoError(s.manager.Enqueue(ctx, low))

	high := &models.QueueItem{QueueID: queue.ID, Priority: 5, Category: "test"}
	s.Require().NoError(s.manager.Enqueue(ctx, high))

	med := &models.QueueItem{QueueID: queue.ID, Priority: 3, Category: "test"}
	s.Require().NoError(s.manager.Enqueue(ctx, med))

	counter := s.createAndOpenCounter(queue.ID, "Window 1", "staff-1")

	// CallNext should return highest priority first.
	called, err := s.manager.CallNext(ctx, counter.ID)
	s.Require().NoError(err)
	s.Equal(high.ID, called.ID, "highest priority item should be called first")
	s.Equal(models.ItemStatusServing, called.Status)

	// Verify counter now has current_item_id set.
	ctr, err := s.manager.GetCounter(ctx, counter.ID)
	s.Require().NoError(err)
	s.Equal(called.ID, ctr.CurrentItemID, "counter should have current_item_id after CallNext")

	// Complete and call next.
	s.Require().NoError(s.manager.CompleteService(ctx, counter.ID))

	called2, err := s.manager.CallNext(ctx, counter.ID)
	s.Require().NoError(err)
	s.Equal(med.ID, called2.ID, "medium priority item should be called second")

	s.Require().NoError(s.manager.CompleteService(ctx, counter.ID))

	called3, err := s.manager.CallNext(ctx, counter.ID)
	s.Require().NoError(err)
	s.Equal(low.ID, called3.ID, "lowest priority item should be called last")
}

func (s *QueueSuite) TestCallNext_FIFOWithinSamePriority() {
	ctx := s.tenantCtx()
	queue := s.createQueue("fifo-queue", 0, 1)

	first := &models.QueueItem{QueueID: queue.ID, Priority: 1}
	s.Require().NoError(s.manager.Enqueue(ctx, first))

	time.Sleep(10 * time.Millisecond) // ensure ordering

	second := &models.QueueItem{QueueID: queue.ID, Priority: 1}
	s.Require().NoError(s.manager.Enqueue(ctx, second))

	counter := s.createAndOpenCounter(queue.ID, "Window 1", "staff-1")

	called, err := s.manager.CallNext(ctx, counter.ID)
	s.Require().NoError(err)
	s.Equal(first.ID, called.ID, "first enqueued should be served first (FIFO)")
}

func (s *QueueSuite) TestCallNext_ConcurrentCounters() {
	ctx := s.tenantCtx()
	queue := s.createQueue("concurrent-queue", 0, 1)

	// Enqueue 10 items.
	itemIDs := make([]string, 10)
	for i := range 10 {
		item := &models.QueueItem{QueueID: queue.ID, Priority: 1}
		s.Require().NoError(s.manager.Enqueue(ctx, item))
		itemIDs[i] = item.ID
		time.Sleep(1 * time.Millisecond) // ensure ordering
	}

	// Create 5 counters.
	counters := make([]*models.QueueCounter, 5)
	for i := range 5 {
		counters[i] = s.createAndOpenCounter(queue.ID, "Window "+string(rune('A'+i)), "staff-"+string(rune('0'+i)))
	}

	// All 5 counters call next concurrently.
	var wg sync.WaitGroup
	calledIDs := make([]string, 5)
	callErrors := make([]error, 5)

	for i := range 5 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			called, err := s.manager.CallNext(ctx, counters[idx].ID)
			callErrors[idx] = err
			if called != nil {
				calledIDs[idx] = called.ID
			}
		}(i)
	}
	wg.Wait()

	// All calls should succeed.
	for i, err := range callErrors {
		s.Require().NoErrorf(err, "counter %d should succeed", i)
	}

	// All called items should be unique (no double-serving).
	seen := make(map[string]bool)
	for _, id := range calledIDs {
		s.NotEmpty(id)
		s.Falsef(seen[id], "item %s was served by multiple counters", id)
		seen[id] = true
	}
}

func (s *QueueSuite) TestCallNext_EmptyQueue() {
	ctx := s.tenantCtx()
	queue := s.createQueue("empty-queue", 0, 1)
	counter := s.createAndOpenCounter(queue.ID, "Window 1", "staff-1")

	_, err := s.manager.CallNext(ctx, counter.ID)
	s.Require().Error(err)
	s.ErrorIs(err, business.ErrNoWaitingItems)
}

func (s *QueueSuite) TestCallNext_CounterBusy() {
	ctx := s.tenantCtx()
	queue := s.createQueue("busy-queue", 0, 1)

	item := &models.QueueItem{QueueID: queue.ID, Priority: 1}
	s.Require().NoError(s.manager.Enqueue(ctx, item))

	item2 := &models.QueueItem{QueueID: queue.ID, Priority: 1}
	s.Require().NoError(s.manager.Enqueue(ctx, item2))

	counter := s.createAndOpenCounter(queue.ID, "Window 1", "staff-1")

	_, err := s.manager.CallNext(ctx, counter.ID)
	s.Require().NoError(err)

	// Counter is now busy - should fail.
	_, err = s.manager.CallNext(ctx, counter.ID)
	s.Require().Error(err)
	s.ErrorIs(err, business.ErrCounterBusy)
}

// --- State transition tests ---

func (s *QueueSuite) TestCompleteService_FullFlow() {
	ctx := s.tenantCtx()
	queue := s.createQueue("complete-queue", 0, 1)

	item := &models.QueueItem{QueueID: queue.ID, Priority: 1}
	s.Require().NoError(s.manager.Enqueue(ctx, item))

	counter := s.createAndOpenCounter(queue.ID, "Window 1", "staff-1")

	// Call next.
	called, err := s.manager.CallNext(ctx, counter.ID)
	s.Require().NoError(err)
	s.Equal(models.ItemStatusServing, called.Status)
	s.NotNil(called.CalledAt)

	// Begin service.
	s.Require().NoError(s.manager.BeginService(ctx, counter.ID))

	// Check service start was set.
	refreshed, err := s.manager.GetItem(ctx, called.ID)
	s.Require().NoError(err)
	s.NotNil(refreshed.ServiceStart)

	// Complete service.
	s.Require().NoError(s.manager.CompleteService(ctx, counter.ID))

	// Verify item is completed.
	final, err := s.manager.GetItem(ctx, called.ID)
	s.Require().NoError(err)
	s.Equal(models.ItemStatusCompleted, final.Status)
	s.NotNil(final.ServiceEnd)

	// Verify counter is freed.
	ctr, err := s.manager.GetCounter(ctx, counter.ID)
	s.Require().NoError(err)
	s.Empty(ctr.CurrentItemID)
	s.Equal(1, ctr.TotalServed)
}

func (s *QueueSuite) TestCancelItem_WhileWaiting() {
	ctx := s.tenantCtx()
	queue := s.createQueue("cancel-queue", 0, 1)

	item := &models.QueueItem{QueueID: queue.ID, Priority: 1}
	s.Require().NoError(s.manager.Enqueue(ctx, item))

	s.Require().NoError(s.manager.CancelItem(ctx, item.ID))

	cancelled, err := s.manager.GetItem(ctx, item.ID)
	s.Require().NoError(err)
	s.Equal(models.ItemStatusCancelled, cancelled.Status)
}

func (s *QueueSuite) TestCancelItem_WhileServing_FreesCounter() {
	ctx := s.tenantCtx()
	queue := s.createQueue("cancel-serving-queue", 0, 1)

	item := &models.QueueItem{QueueID: queue.ID, Priority: 1}
	s.Require().NoError(s.manager.Enqueue(ctx, item))

	counter := s.createAndOpenCounter(queue.ID, "Window 1", "staff-1")
	_, err := s.manager.CallNext(ctx, counter.ID)
	s.Require().NoError(err)

	// Cancel while serving.
	s.Require().NoError(s.manager.CancelItem(ctx, item.ID))

	// Counter should be freed.
	ctr, err := s.manager.GetCounter(ctx, counter.ID)
	s.Require().NoError(err)
	s.Empty(ctr.CurrentItemID, "counter should be freed after cancellation")
}

func (s *QueueSuite) TestNoShow_AndRequeue() {
	ctx := s.tenantCtx()
	queue := s.createQueue("noshow-queue", 0, 1)

	item := &models.QueueItem{QueueID: queue.ID, Priority: 1}
	s.Require().NoError(s.manager.Enqueue(ctx, item))

	counter := s.createAndOpenCounter(queue.ID, "Window 1", "staff-1")
	_, err := s.manager.CallNext(ctx, counter.ID)
	s.Require().NoError(err)

	// Mark no-show.
	s.Require().NoError(s.manager.NoShowItem(ctx, item.ID))

	noShow, err := s.manager.GetItem(ctx, item.ID)
	s.Require().NoError(err)
	s.Equal(models.ItemStatusNoShow, noShow.Status)
	s.Empty(noShow.CounterID, "counter should be cleared on no-show")

	// Counter should be freed.
	ctr, err := s.manager.GetCounter(ctx, counter.ID)
	s.Require().NoError(err)
	s.Empty(ctr.CurrentItemID)

	// Re-queue.
	s.Require().NoError(s.manager.RequeueItem(ctx, item.ID))

	requeued, err := s.manager.GetItem(ctx, item.ID)
	s.Require().NoError(err)
	s.Equal(models.ItemStatusWaiting, requeued.Status)
}

func (s *QueueSuite) TestTransferItem() {
	ctx := s.tenantCtx()
	sourceQueue := s.createQueue("source-queue", 0, 3)
	targetQueue := s.createQueue("target-queue", 0, 3)

	item := &models.QueueItem{QueueID: sourceQueue.ID, Priority: 2}
	s.Require().NoError(s.manager.Enqueue(ctx, item))

	// Transfer to target queue.
	s.Require().NoError(s.manager.TransferItem(ctx, item.ID, targetQueue.ID))

	transferred, err := s.manager.GetItem(ctx, item.ID)
	s.Require().NoError(err)
	s.Equal(targetQueue.ID, transferred.QueueID, "item should be in target queue")
	s.Equal(models.ItemStatusWaiting, transferred.Status)
}

func (s *QueueSuite) TestTransferItem_WhileServing_FreesCounter() {
	ctx := s.tenantCtx()
	sourceQueue := s.createQueue("transfer-src", 0, 1)
	targetQueue := s.createQueue("transfer-dst", 0, 1)

	item := &models.QueueItem{QueueID: sourceQueue.ID, Priority: 1}
	s.Require().NoError(s.manager.Enqueue(ctx, item))

	counter := s.createAndOpenCounter(sourceQueue.ID, "Window 1", "staff-1")
	_, err := s.manager.CallNext(ctx, counter.ID)
	s.Require().NoError(err)

	// Transfer while serving.
	s.Require().NoError(s.manager.TransferItem(ctx, item.ID, targetQueue.ID))

	// Counter should be freed.
	ctr, err := s.manager.GetCounter(ctx, counter.ID)
	s.Require().NoError(err)
	s.Empty(ctr.CurrentItemID, "counter should be freed after transfer")
}

// --- Counter tests ---

func (s *QueueSuite) TestCounter_InvalidTransitions() {
	ctx := s.tenantCtx()
	queue := s.createQueue("counter-trans-queue", 0, 1)

	counter := &models.QueueCounter{QueueID: queue.ID, Name: "Window 1"}
	s.Require().NoError(s.manager.CreateCounter(ctx, counter))

	// Counter starts closed. Cannot pause from closed.
	err := s.manager.PauseCounter(ctx, counter.ID)
	s.Require().Error(err)
	s.Require().ErrorIs(err, business.ErrInvalidTransition)

	// Open the counter.
	s.Require().NoError(s.manager.OpenCounter(ctx, counter.ID, "staff-1"))

	// Can pause from open.
	s.Require().NoError(s.manager.PauseCounter(ctx, counter.ID))

	// Can resume from paused.
	s.Require().NoError(s.manager.OpenCounter(ctx, counter.ID, "staff-1"))
}

// --- Stats tests ---

func (s *QueueSuite) TestStats_BasicCounts() {
	ctx := s.tenantCtx()
	queue := s.createQueue("stats-queue", 0, 1)

	// Enqueue 3 items.
	for range 3 {
		item := &models.QueueItem{QueueID: queue.ID, Priority: 1}
		s.Require().NoError(s.manager.Enqueue(ctx, item))
	}

	stats, err := s.stats.GetStats(ctx, queue.ID)
	s.Require().NoError(err)
	s.Equal(int64(3), stats.TotalWaiting)
	s.Equal(int64(0), stats.TotalServing)
}
