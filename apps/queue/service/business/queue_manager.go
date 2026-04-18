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

package business

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/pitabwire/frame/security"
	"github.com/pitabwire/util"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"gorm.io/gorm"

	"github.com/antinvestor/service-trustage/apps/queue/service/models"
	"github.com/antinvestor/service-trustage/apps/queue/service/repository"
)

// QueueManager manages queue definitions, items, and counters.
type QueueManager interface {
	// Queue definitions.
	CreateQueue(ctx context.Context, def *models.QueueDefinition) error
	GetQueue(ctx context.Context, id string) (*models.QueueDefinition, error)
	ListQueues(ctx context.Context, activeOnly bool) ([]*models.QueueDefinition, error)
	UpdateQueue(ctx context.Context, def *models.QueueDefinition) error
	DeleteQueue(ctx context.Context, id string) error

	// Queue items.
	Enqueue(ctx context.Context, item *models.QueueItem) error
	GetItem(ctx context.Context, id string) (*models.QueueItem, error)
	GetItemPosition(ctx context.Context, id string) (int, error)
	ListWaitingItems(ctx context.Context, queueID string, limit, offset int) ([]*models.QueueItem, error)
	CancelItem(ctx context.Context, id string) error
	NoShowItem(ctx context.Context, id string) error
	RequeueItem(ctx context.Context, id string) error
	TransferItem(ctx context.Context, id, newQueueID string) error

	// Counters.
	CreateCounter(ctx context.Context, counter *models.QueueCounter) error
	GetCounter(ctx context.Context, id string) (*models.QueueCounter, error)
	ListCounters(ctx context.Context, queueID string) ([]*models.QueueCounter, error)
	OpenCounter(ctx context.Context, id, staffID string) error
	CloseCounter(ctx context.Context, id string) error
	PauseCounter(ctx context.Context, id string) error
	CallNext(ctx context.Context, counterID string) (*models.QueueItem, error)
	BeginService(ctx context.Context, counterID string) error
	CompleteService(ctx context.Context, counterID string) error
}

type queueManager struct {
	defRepo     repository.QueueDefinitionRepository
	itemRepo    repository.QueueItemRepository
	counterRepo repository.QueueCounterRepository
	stats       QueueStatsService
}

// NewQueueManager creates a new QueueManager.
func NewQueueManager(
	defRepo repository.QueueDefinitionRepository,
	itemRepo repository.QueueItemRepository,
	counterRepo repository.QueueCounterRepository,
	stats QueueStatsService,
) QueueManager {
	return &queueManager{
		defRepo:     defRepo,
		itemRepo:    itemRepo,
		counterRepo: counterRepo,
		stats:       stats,
	}
}

// --- Queue definitions ---

func (m *queueManager) CreateQueue(ctx context.Context, def *models.QueueDefinition) error {
	log := util.Log(ctx)

	// Ensure JSONB fields contain valid JSON for PostgreSQL.
	if def.Config == "" {
		def.Config = "{}"
	}

	if err := m.defRepo.Create(ctx, def); err != nil {
		return fmt.Errorf("persist queue definition: %w", err)
	}

	log.Info("queue created", "queue_id", def.ID, "name", def.Name)

	return nil
}

func (m *queueManager) GetQueue(ctx context.Context, id string) (*models.QueueDefinition, error) {
	def, err := m.defRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrQueueNotFound, err)
	}

	return def, nil
}

func (m *queueManager) ListQueues(ctx context.Context, activeOnly bool) ([]*models.QueueDefinition, error) {
	return m.defRepo.List(ctx, activeOnly)
}

func (m *queueManager) UpdateQueue(ctx context.Context, def *models.QueueDefinition) error {
	if err := m.defRepo.Update(ctx, def); err != nil {
		return fmt.Errorf("update queue definition: %w", err)
	}

	return nil
}

func (m *queueManager) DeleteQueue(ctx context.Context, id string) error {
	def, err := m.defRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrQueueNotFound, err)
	}

	if deleteErr := m.defRepo.SoftDelete(ctx, def); deleteErr != nil {
		return fmt.Errorf("delete queue: %w", deleteErr)
	}

	return nil
}

// --- Queue items ---

//nolint:gocognit // enqueue flow handles multiple transitions and validations
func (m *queueManager) Enqueue(ctx context.Context, item *models.QueueItem) error {
	log := util.Log(ctx)
	start := time.Now()
	attrs := tenantMetricAttributes(ctx)

	// Validate queue exists and get its config.
	queueDef, err := m.defRepo.GetByID(ctx, item.QueueID)
	if err != nil {
		enqueueErrorCounter.Add(ctx, 1, attrs)
		return fmt.Errorf("%w: %w", ErrQueueNotFound, err)
	}

	// Clamp priority to the queue's configured levels.
	if item.Priority < 1 {
		item.Priority = 1
	}

	if item.Priority > queueDef.PriorityLevels {
		item.Priority = queueDef.PriorityLevels
	}

	item.Status = models.ItemStatusWaiting
	item.JoinedAt = time.Now()

	// Ensure JSONB fields contain valid JSON for PostgreSQL.
	if item.Metadata == "" {
		item.Metadata = "{}"
	}

	// Generate ticket number if not provided.
	if item.TicketNo == "" {
		item.TicketNo = generateTicketNo()
	}

	// Wrap capacity check + insert in a single transaction to prevent TOCTOU races.
	db := m.itemRepo.Pool().DB(ctx, false)

	txErr := db.Transaction(func(tx *gorm.DB) error {
		if queueDef.MaxCapacity > 0 {
			// Lock the queue definition row to serialize capacity checks.
			if lockErr := tx.Raw(
				`SELECT id FROM queue_definitions WHERE id = ? FOR UPDATE`,
				item.QueueID,
			).Scan(&struct{ ID string }{}).Error; lockErr != nil {
				return fmt.Errorf("lock queue definition: %w", lockErr)
			}

			var count int64
			if countErr := tx.Raw(
				`SELECT COUNT(*) FROM queue_items WHERE queue_id = ? AND status = ? AND deleted_at IS NULL`,
				item.QueueID, models.ItemStatusWaiting,
			).Scan(&count).Error; countErr != nil {
				return fmt.Errorf("check capacity: %w", countErr)
			}

			if count >= int64(queueDef.MaxCapacity) {
				return ErrQueueFull
			}
		}

		if createErr := tx.Create(item).Error; createErr != nil {
			return fmt.Errorf("persist queue item: %w", createErr)
		}

		return nil
	})

	if txErr != nil {
		if errors.Is(txErr, ErrQueueFull) {
			queueFullCounter.Add(ctx, 1, attrs)
		} else {
			enqueueErrorCounter.Add(ctx, 1, attrs)
		}

		return txErr
	}

	enqueueCounter.Add(ctx, 1, attrs)
	enqueueHistogram.Record(ctx, float64(time.Since(start).Milliseconds()), attrs)
	m.stats.InvalidateCache(ctx, item.QueueID)

	log.Info("item enqueued",
		"item_id", item.ID,
		"queue_id", item.QueueID,
		"ticket", item.TicketNo,
		"priority", item.Priority,
	)

	return nil
}

func (m *queueManager) GetItem(ctx context.Context, id string) (*models.QueueItem, error) {
	item, err := m.itemRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrQueueItemNotFound, err)
	}

	return item, nil
}

func (m *queueManager) GetItemPosition(ctx context.Context, id string) (int, error) {
	item, err := m.itemRepo.GetByID(ctx, id)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrQueueItemNotFound, err)
	}

	if item.Status != models.ItemStatusWaiting {
		return 0, ErrItemNotWaiting
	}

	return m.itemRepo.GetPosition(ctx, item)
}

func (m *queueManager) ListWaitingItems(
	ctx context.Context,
	queueID string,
	limit, offset int,
) ([]*models.QueueItem, error) {
	return m.itemRepo.ListWaiting(ctx, queueID, limit, offset)
}

func (m *queueManager) CancelItem(ctx context.Context, id string) error {
	item, err := m.itemRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrQueueItemNotFound, err)
	}

	if transitionErr := item.TransitionTo(models.ItemStatusCancelled); transitionErr != nil {
		return fmt.Errorf("%w: %w", ErrInvalidTransition, transitionErr)
	}

	// Atomically update item + free counter in a single transaction.
	db := m.itemRepo.Pool().DB(ctx, false)

	txErr := db.Transaction(func(tx *gorm.DB) error {
		if saveErr := tx.Save(item).Error; saveErr != nil {
			return fmt.Errorf("update item: %w", saveErr)
		}

		// If this item was being served, free the counter using raw SQL
		// to avoid GORM scope interference across different models.
		if item.CounterID != "" {
			if counterErr := tx.Exec(
				`UPDATE queue_counters SET current_item_id = '' WHERE id = ? AND deleted_at IS NULL`,
				item.CounterID,
			).Error; counterErr != nil {
				return fmt.Errorf("free counter: %w", counterErr)
			}
		}

		return nil
	})

	if txErr != nil {
		return txErr
	}

	cancelCounter.Add(ctx, 1)
	m.stats.InvalidateCache(ctx, item.QueueID)

	return nil
}

func (m *queueManager) NoShowItem(ctx context.Context, id string) error {
	item, err := m.itemRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrQueueItemNotFound, err)
	}

	counterIDToFree := item.CounterID

	if transitionErr := item.TransitionTo(models.ItemStatusNoShow); transitionErr != nil {
		return fmt.Errorf("%w: %w", ErrInvalidTransition, transitionErr)
	}

	item.CounterID = ""
	item.ServedBy = ""

	// Atomically update item + free counter in a single transaction.
	db := m.itemRepo.Pool().DB(ctx, false)

	txErr := db.Transaction(func(tx *gorm.DB) error {
		if saveErr := tx.Save(item).Error; saveErr != nil {
			return fmt.Errorf("update item: %w", saveErr)
		}

		if counterIDToFree != "" {
			if counterErr := tx.Exec(
				`UPDATE queue_counters SET current_item_id = '' WHERE id = ? AND deleted_at IS NULL`,
				counterIDToFree,
			).Error; counterErr != nil {
				return fmt.Errorf("free counter: %w", counterErr)
			}
		}

		return nil
	})

	if txErr != nil {
		return txErr
	}

	noShowCounter.Add(ctx, 1)
	m.stats.InvalidateCache(ctx, item.QueueID)

	return nil
}

func (m *queueManager) RequeueItem(ctx context.Context, id string) error {
	item, err := m.itemRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrQueueItemNotFound, err)
	}

	if item.Status != models.ItemStatusNoShow {
		return ErrItemNotNoShow
	}

	if transitionErr := item.TransitionTo(models.ItemStatusWaiting); transitionErr != nil {
		return fmt.Errorf("%w: %w", ErrInvalidTransition, transitionErr)
	}

	item.CounterID = ""
	item.ServedBy = ""
	item.CalledAt = nil
	item.JoinedAt = time.Now()

	if updateErr := m.itemRepo.Update(ctx, item); updateErr != nil {
		return fmt.Errorf("update item: %w", updateErr)
	}

	m.stats.InvalidateCache(ctx, item.QueueID)

	return nil
}

func (m *queueManager) TransferItem(ctx context.Context, id, newQueueID string) error {
	item, err := m.itemRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrQueueItemNotFound, err)
	}

	// Only allow transfer from waiting or serving status.
	if item.Status != models.ItemStatusWaiting && item.Status != models.ItemStatusServing {
		return fmt.Errorf("%w: can only transfer items that are waiting or serving", ErrInvalidTransition)
	}

	// Validate target queue exists.
	if _, lookupErr := m.defRepo.GetByID(ctx, newQueueID); lookupErr != nil {
		return fmt.Errorf("%w: target queue: %w", ErrQueueNotFound, lookupErr)
	}

	counterIDToFree := item.CounterID
	oldQueueID := item.QueueID
	item.QueueID = newQueueID
	item.Status = models.ItemStatusWaiting
	item.CounterID = ""
	item.ServedBy = ""
	item.CalledAt = nil
	item.ServiceStart = nil
	item.JoinedAt = time.Now()

	// Atomically update item + free counter in a single transaction.
	db := m.itemRepo.Pool().DB(ctx, false)

	txErr := db.Transaction(func(tx *gorm.DB) error {
		if saveErr := tx.Save(item).Error; saveErr != nil {
			return fmt.Errorf("update item: %w", saveErr)
		}

		// Free the counter using raw SQL to avoid GORM scope interference across different models.
		if counterIDToFree != "" {
			if counterErr := tx.Exec(
				`UPDATE queue_counters SET current_item_id = '' WHERE id = ? AND deleted_at IS NULL`,
				counterIDToFree,
			).Error; counterErr != nil {
				return fmt.Errorf("free counter: %w", counterErr)
			}
		}

		return nil
	})

	if txErr != nil {
		return txErr
	}

	transferCounter.Add(ctx, 1)
	m.stats.InvalidateCache(ctx, oldQueueID)
	m.stats.InvalidateCache(ctx, newQueueID)

	return nil
}

// --- Counters ---

func (m *queueManager) CreateCounter(ctx context.Context, counter *models.QueueCounter) error {
	log := util.Log(ctx)

	// Validate queue exists.
	if _, err := m.defRepo.GetByID(ctx, counter.QueueID); err != nil {
		return fmt.Errorf("%w: %w", ErrQueueNotFound, err)
	}

	counter.Status = models.CounterStatusClosed

	// Ensure JSONB fields contain valid JSON for PostgreSQL.
	if counter.Categories == "" {
		counter.Categories = "{}"
	}

	if err := m.counterRepo.Create(ctx, counter); err != nil {
		return fmt.Errorf("persist counter: %w", err)
	}

	log.Info("counter created", "counter_id", counter.ID, "name", counter.Name)

	return nil
}

func (m *queueManager) GetCounter(ctx context.Context, id string) (*models.QueueCounter, error) {
	counter, err := m.counterRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCounterNotFound, err)
	}

	return counter, nil
}

func (m *queueManager) ListCounters(ctx context.Context, queueID string) ([]*models.QueueCounter, error) {
	return m.counterRepo.ListByQueueID(ctx, queueID)
}

func (m *queueManager) OpenCounter(ctx context.Context, id, staffID string) error {
	counter, err := m.counterRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrCounterNotFound, err)
	}

	if transitionErr := counter.TransitionTo(models.CounterStatusOpen); transitionErr != nil {
		return fmt.Errorf("%w: %w", ErrInvalidTransition, transitionErr)
	}

	counter.ServedBy = staffID

	if updateErr := m.counterRepo.Update(ctx, counter); updateErr != nil {
		return fmt.Errorf("update counter: %w", updateErr)
	}

	m.stats.InvalidateCache(ctx, counter.QueueID)

	return nil
}

func (m *queueManager) CloseCounter(ctx context.Context, id string) error {
	counter, err := m.counterRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrCounterNotFound, err)
	}

	if transitionErr := counter.TransitionTo(models.CounterStatusClosed); transitionErr != nil {
		return fmt.Errorf("%w: %w", ErrInvalidTransition, transitionErr)
	}

	itemIDToRequeue := counter.CurrentItemID
	counter.CurrentItemID = ""
	counter.ServedBy = ""

	// Atomically update counter + re-queue item in a single transaction.
	db := m.itemRepo.Pool().DB(ctx, false)

	txErr := db.Transaction(func(tx *gorm.DB) error {
		if saveErr := tx.Save(counter).Error; saveErr != nil {
			return fmt.Errorf("update counter: %w", saveErr)
		}

		// Re-queue the item using raw SQL to avoid GORM scope interference across different models.
		if itemIDToRequeue != "" {
			now := time.Now()
			if itemErr := tx.Exec(
				`UPDATE queue_items SET status = ?, counter_id = '', served_by = '',
				 called_at = NULL, service_start = NULL, joined_at = ?
				 WHERE id = ? AND status = ? AND deleted_at IS NULL`,
				models.ItemStatusWaiting, now, itemIDToRequeue, models.ItemStatusServing,
			).Error; itemErr != nil {
				return fmt.Errorf("re-queue item: %w", itemErr)
			}
		}

		return nil
	})

	if txErr != nil {
		return txErr
	}

	m.stats.InvalidateCache(ctx, counter.QueueID)

	return nil
}

func (m *queueManager) PauseCounter(ctx context.Context, id string) error {
	counter, err := m.counterRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrCounterNotFound, err)
	}

	if transitionErr := counter.TransitionTo(models.CounterStatusPaused); transitionErr != nil {
		return fmt.Errorf("%w: %w", ErrInvalidTransition, transitionErr)
	}

	if updateErr := m.counterRepo.Update(ctx, counter); updateErr != nil {
		return fmt.Errorf("update counter: %w", updateErr)
	}

	m.stats.InvalidateCache(ctx, counter.QueueID)

	return nil
}

// CallNext atomically finds and assigns the next waiting item to the counter.
// Uses a database transaction to ensure item and counter updates are atomic.
//
//nolint:funlen // call-next logic is intentionally explicit for correctness
func (m *queueManager) CallNext(ctx context.Context, counterID string) (*models.QueueItem, error) {
	log := util.Log(ctx)
	start := time.Now()

	counter, err := m.counterRepo.GetByID(ctx, counterID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCounterNotFound, err)
	}

	if counter.Status != models.CounterStatusOpen {
		return nil, ErrCounterNotOpen
	}

	if counter.CurrentItemID != "" {
		return nil, ErrCounterBusy
	}

	// Parse counter's category filter.
	var categories []string
	if counter.Categories != "" {
		_ = json.Unmarshal([]byte(counter.Categories), &categories)
	}

	// Use a transaction to atomically find item + update item + update counter.
	db := m.itemRepo.Pool().DB(ctx, false)

	var calledItem *models.QueueItem

	txErr := db.Transaction(func(tx *gorm.DB) error {
		// Find next waiting item with FOR UPDATE SKIP LOCKED.
		var item models.QueueItem

		var query string

		var args []any

		if len(categories) > 0 {
			query = `SELECT * FROM queue_items
				 WHERE queue_id = ? AND status = ? AND deleted_at IS NULL AND category = ANY(?)
				 ORDER BY priority DESC, joined_at ASC
				 LIMIT 1
				 FOR UPDATE SKIP LOCKED`
			args = []any{counter.QueueID, models.ItemStatusWaiting, pq.Array(categories)}
		} else {
			query = `SELECT * FROM queue_items
				 WHERE queue_id = ? AND status = ? AND deleted_at IS NULL
				 ORDER BY priority DESC, joined_at ASC
				 LIMIT 1
				 FOR UPDATE SKIP LOCKED`
			args = []any{counter.QueueID, models.ItemStatusWaiting}
		}

		if scanErr := tx.Raw(query, args...).Scan(&item).Error; scanErr != nil {
			return fmt.Errorf("find next waiting: %w", scanErr)
		}

		if item.ID == "" {
			return ErrNoWaitingItems
		}

		// Transition item to serving.
		if transErr := item.TransitionTo(models.ItemStatusServing); transErr != nil {
			return fmt.Errorf("%w: %w", ErrInvalidTransition, transErr)
		}

		now := time.Now()
		item.CounterID = counterID
		item.ServedBy = counter.ServedBy
		item.CalledAt = &now

		if saveErr := tx.Save(&item).Error; saveErr != nil {
			return fmt.Errorf("update item: %w", saveErr)
		}

		// Update counter's current item using raw SQL to avoid GORM scope interference.
		counterResult := tx.Exec(
			`UPDATE queue_counters SET current_item_id = ? WHERE id = ? AND deleted_at IS NULL`,
			item.ID, counterID,
		)
		if counterResult.Error != nil {
			return fmt.Errorf("update counter: %w", counterResult.Error)
		}

		if counterResult.RowsAffected == 0 {
			return fmt.Errorf("update counter: no rows affected for counter %s", counterID)
		}

		calledItem = &item

		return nil
	})

	if txErr != nil {
		dequeueErrorCounter.Add(ctx, 1)
		return nil, txErr
	}

	dequeueCounter.Add(ctx, 1)
	dequeueHistogram.Record(ctx, float64(time.Since(start).Milliseconds()))
	m.stats.InvalidateCache(ctx, counter.QueueID)

	log.Info("item called",
		"item_id", calledItem.ID,
		"counter_id", counterID,
		"ticket", calledItem.TicketNo,
	)

	return calledItem, nil
}

// BeginService marks that actual service has started for the current item.
func (m *queueManager) BeginService(ctx context.Context, counterID string) error {
	counter, err := m.counterRepo.GetByID(ctx, counterID)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrCounterNotFound, err)
	}

	if counter.CurrentItemID == "" {
		return ErrCounterNotServing
	}

	item, err := m.itemRepo.GetByID(ctx, counter.CurrentItemID)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrQueueItemNotFound, err)
	}

	now := time.Now()
	item.ServiceStart = &now

	if updateErr := m.itemRepo.Update(ctx, item); updateErr != nil {
		return fmt.Errorf("update item: %w", updateErr)
	}

	return nil
}

// CompleteService atomically marks the current item as completed and frees the counter.
func (m *queueManager) CompleteService(ctx context.Context, counterID string) error {
	log := util.Log(ctx)

	counter, err := m.counterRepo.GetByID(ctx, counterID)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrCounterNotFound, err)
	}

	if counter.CurrentItemID == "" {
		return ErrCounterNotServing
	}

	item, err := m.itemRepo.GetByID(ctx, counter.CurrentItemID)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrQueueItemNotFound, err)
	}

	if transitionErr := item.TransitionTo(models.ItemStatusCompleted); transitionErr != nil {
		return fmt.Errorf("%w: %w", ErrInvalidTransition, transitionErr)
	}

	now := time.Now()
	item.ServiceEnd = &now

	// Use transaction to atomically update item and counter.
	db := m.itemRepo.Pool().DB(ctx, false)

	txErr := db.Transaction(func(tx *gorm.DB) error {
		if saveErr := tx.Save(item).Error; saveErr != nil {
			return fmt.Errorf("update item: %w", saveErr)
		}

		// Update counter using raw SQL to avoid GORM scope interference across different models.
		counterResult := tx.Exec(
			`UPDATE queue_counters SET current_item_id = '', total_served = total_served + 1
			 WHERE id = ? AND deleted_at IS NULL`,
			counterID,
		)
		if counterResult.Error != nil {
			return fmt.Errorf("update counter: %w", counterResult.Error)
		}

		if counterResult.RowsAffected == 0 {
			return fmt.Errorf("update counter: no rows affected for counter %s", counterID)
		}

		return nil
	})

	if txErr != nil {
		return txErr
	}

	completeCounter.Add(ctx, 1)
	m.stats.InvalidateCache(ctx, counter.QueueID)

	log.Info("service completed",
		"item_id", item.ID,
		"counter_id", counterID,
		"ticket", item.TicketNo,
	)

	return nil
}

func tenantMetricAttributes(ctx context.Context) metric.MeasurementOption {
	claims := security.ClaimsFromContext(ctx)
	if claims == nil {
		return metric.WithAttributes()
	}

	tenantID := claims.GetTenantID()
	if tenantID == "" {
		return metric.WithAttributes()
	}

	return metric.WithAttributes(attribute.String("tenant_id", tenantID))
}

const ticketNoRandomLen = 8

// generateTicketNo creates a short human-readable ticket number.
// Uses 8 random alphanumeric characters for collision resistance.
func generateTicketNo() string {
	return fmt.Sprintf("T-%s", util.RandomAlphaNumericString(ticketNoRandomLen))
}
