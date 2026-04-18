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

package tests_test

import (
	"time"

	"github.com/antinvestor/service-trustage/apps/queue/service/business"
	"github.com/antinvestor/service-trustage/apps/queue/service/models"
)

func (s *QueueSuite) TestQueueDefinitionCRUDAndListing() {
	ctx := s.tenantCtx()

	queueA := s.createQueue("queue-a", 0, 3)
	queueB := s.createQueue("queue-b", 0, 2)

	items, err := s.manager.ListQueues(ctx, false)
	s.Require().NoError(err)
	s.Len(items, 2)

	queueA.Description = "updated"
	s.Require().NoError(s.manager.UpdateQueue(ctx, queueA))

	updated, err := s.manager.GetQueue(ctx, queueA.ID)
	s.Require().NoError(err)
	s.Equal("updated", updated.Description)

	s.Require().NoError(s.manager.DeleteQueue(ctx, queueB.ID))
	_, err = s.manager.GetQueue(ctx, queueB.ID)
	s.Require().Error(err)
	s.ErrorIs(err, business.ErrQueueNotFound)
}

func (s *QueueSuite) TestQueueItemPositionAndWaitingList() {
	ctx := s.tenantCtx()
	queue := s.createQueue("positions", 0, 3)

	first := &models.QueueItem{QueueID: queue.ID, Priority: 2}
	second := &models.QueueItem{QueueID: queue.ID, Priority: 1}
	third := &models.QueueItem{QueueID: queue.ID, Priority: 3}
	s.Require().NoError(s.manager.Enqueue(ctx, first))
	s.Require().NoError(s.manager.Enqueue(ctx, second))
	s.Require().NoError(s.manager.Enqueue(ctx, third))

	position, err := s.manager.GetItemPosition(ctx, second.ID)
	s.Require().NoError(err)
	s.Equal(3, position)

	items, err := s.manager.ListWaitingItems(ctx, queue.ID, 10, 0)
	s.Require().NoError(err)
	s.Len(items, 3)
	s.Equal(third.ID, items[0].ID)
	s.Equal(first.ID, items[1].ID)
	s.Equal(second.ID, items[2].ID)
}

func (s *QueueSuite) TestListCountersAndCloseCounterRequeuesCurrentItem() {
	ctx := s.tenantCtx()
	queue := s.createQueue("counters", 0, 1)
	counter := s.createAndOpenCounter(queue.ID, "Window 1", "staff-1")

	item := &models.QueueItem{QueueID: queue.ID, Priority: 1}
	s.Require().NoError(s.manager.Enqueue(ctx, item))
	_, err := s.manager.CallNext(ctx, counter.ID)
	s.Require().NoError(err)

	counters, err := s.manager.ListCounters(ctx, queue.ID)
	s.Require().NoError(err)
	s.Len(counters, 1)

	s.Require().NoError(s.manager.CloseCounter(ctx, counter.ID))

	updatedCounter, err := s.manager.GetCounter(ctx, counter.ID)
	s.Require().NoError(err)
	s.Equal(models.CounterStatusClosed, updatedCounter.Status)
	s.Empty(updatedCounter.CurrentItemID)

	updatedItem, err := s.manager.GetItem(ctx, item.ID)
	s.Require().NoError(err)
	s.Equal(models.ItemStatusWaiting, updatedItem.Status)
}

func (s *QueueSuite) TestQueueRepositories_FindByNameAndItemOperations() {
	ctx := s.tenantCtx()
	queue := s.createQueue("repo-queue", 0, 3)

	found, err := s.defRepo.GetByName(ctx, "repo-queue")
	s.Require().NoError(err)
	s.Equal(queue.ID, found.ID)

	joinedAt := time.Now()
	first := &models.QueueItem{
		QueueID:  queue.ID,
		Priority: 2,
		Status:   models.ItemStatusWaiting,
		Category: "vip",
		Metadata: "{}",
		TicketNo: "VIP-1",
		JoinedAt: joinedAt,
	}
	second := &models.QueueItem{
		QueueID:  queue.ID,
		Priority: 1,
		Status:   models.ItemStatusWaiting,
		Category: "standard",
		Metadata: "{}",
		TicketNo: "STD-1",
		JoinedAt: joinedAt.Add(time.Millisecond),
	}
	s.Require().NoError(s.itemRepo.Create(ctx, first))
	s.Require().NoError(s.itemRepo.Create(ctx, second))

	next, err := s.itemRepo.FindNextWaiting(ctx, queue.ID, []string{"vip"})
	s.Require().NoError(err)
	s.Equal(first.ID, next.ID)

	waitingCount, err := s.itemRepo.CountWaitingForUpdate(ctx, queue.ID)
	s.Require().NoError(err)
	s.Equal(int64(2), waitingCount)

	items, err := s.itemRepo.ListWaiting(ctx, queue.ID, 1, 1)
	s.Require().NoError(err)
	s.Len(items, 1)
	s.Equal(second.ID, items[0].ID)

	counter := s.createAndOpenCounter(queue.ID, "Window Repo", "staff-1")
	openCount, err := s.counterRepo.CountOpen(ctx, queue.ID)
	s.Require().NoError(err)
	s.Equal(int64(1), openCount)

	s.Require().NoError(s.counterRepo.SoftDelete(ctx, counter))
	counters, err := s.counterRepo.ListByQueueID(ctx, queue.ID)
	s.Require().NoError(err)
	s.Empty(counters)
}

func (s *QueueSuite) TestQueueManager_NoShowTransferAndPositionErrors() {
	ctx := s.tenantCtx()
	source := s.createQueue("source-queue", 0, 3)
	target := s.createQueue("target-queue", 0, 3)
	counter := s.createAndOpenCounter(source.ID, "Window 1", "staff-1")

	item := &models.QueueItem{QueueID: source.ID, Priority: 2}
	s.Require().NoError(s.manager.Enqueue(ctx, item))
	_, err := s.manager.CallNext(ctx, counter.ID)
	s.Require().NoError(err)

	s.Require().NoError(s.manager.NoShowItem(ctx, item.ID))
	updated, err := s.manager.GetItem(ctx, item.ID)
	s.Require().NoError(err)
	s.Equal(models.ItemStatusNoShow, updated.Status)

	s.Require().NoError(s.manager.RequeueItem(ctx, item.ID))
	waiting, err := s.manager.GetItem(ctx, item.ID)
	s.Require().NoError(err)
	s.Equal(models.ItemStatusWaiting, waiting.Status)

	s.Require().NoError(s.manager.TransferItem(ctx, item.ID, target.ID))
	transferred, err := s.manager.GetItem(ctx, item.ID)
	s.Require().NoError(err)
	s.Equal(target.ID, transferred.QueueID)
	s.Equal(models.ItemStatusWaiting, transferred.Status)

	position, err := s.manager.GetItemPosition(ctx, item.ID)
	s.Require().NoError(err)
	s.Equal(1, position)

	_, err = s.manager.GetItemPosition(ctx, "missing")
	s.Require().Error(err)
	s.ErrorIs(err, business.ErrQueueItemNotFound)
}
