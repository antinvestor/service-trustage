package business

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/pitabwire/frame/cache"

	"github.com/antinvestor/service-trustage/apps/queue/service/models"
	"github.com/antinvestor/service-trustage/apps/queue/service/repository"
)

// minInvalidateInterval is the minimum time between cache invalidations for the
// same queue. This prevents cache thrashing under sustained enqueue/dequeue traffic
// where every operation would otherwise trigger 8 sequential database queries.
const minInvalidateInterval = 5 * time.Second

// QueueStats holds computed statistics for a queue.
type QueueStats struct {
	TotalWaiting   int64   `json:"total_waiting"`
	TotalServing   int64   `json:"total_serving"`
	AvgWaitMinutes float64 `json:"avg_wait_minutes"`
	EstimatedWait  float64 `json:"estimated_wait_minutes"`
	LongestWait    float64 `json:"longest_wait_minutes"`
	TodayServed    int64   `json:"today_served"`
	TodayCancelled int64   `json:"today_cancelled"`
	TodayNoShow    int64   `json:"today_no_show"`
	OpenCounters   int64   `json:"open_counters"`
}

// QueueStatsService computes and caches queue statistics.
type QueueStatsService interface {
	GetStats(ctx context.Context, queueID string) (*QueueStats, error)
	InvalidateCache(ctx context.Context, queueID string)
}

type queueStatsService struct {
	itemRepo    repository.QueueItemRepository
	counterRepo repository.QueueCounterRepository
	rawCache    cache.RawCache
	cacheTTL    time.Duration

	// lastInvalidate tracks the last invalidation time per queue to prevent thrashing.
	lastInvalidate   map[string]time.Time
	lastInvalidateMu sync.Mutex
}

// NewQueueStatsService creates a new QueueStatsService.
func NewQueueStatsService(
	itemRepo repository.QueueItemRepository,
	counterRepo repository.QueueCounterRepository,
	rawCache cache.RawCache,
	cacheTTLSeconds int,
) QueueStatsService {
	return &queueStatsService{
		itemRepo:       itemRepo,
		counterRepo:    counterRepo,
		rawCache:       rawCache,
		cacheTTL:       time.Duration(cacheTTLSeconds) * time.Second,
		lastInvalidate: make(map[string]time.Time),
	}
}

func statsCacheKey(queueID string) string {
	return fmt.Sprintf("queue:stats:%s", queueID)
}

func (s *queueStatsService) GetStats(ctx context.Context, queueID string) (*QueueStats, error) {
	cacheKey := statsCacheKey(queueID)

	// Try cache first.
	if cached, found, err := s.rawCache.Get(ctx, cacheKey); err == nil && found && len(cached) > 0 {
		var stats QueueStats
		if jsonErr := json.Unmarshal(cached, &stats); jsonErr == nil {
			return &stats, nil
		}
	}

	// Compute stats.
	stats, err := s.computeStats(ctx, queueID)
	if err != nil {
		return nil, err
	}

	// Cache result.
	if data, marshalErr := json.Marshal(stats); marshalErr == nil {
		_ = s.rawCache.Set(ctx, cacheKey, data, s.cacheTTL)
	}

	return stats, nil
}

func (s *queueStatsService) InvalidateCache(ctx context.Context, queueID string) {
	s.lastInvalidateMu.Lock()
	last, ok := s.lastInvalidate[queueID]
	now := time.Now()

	if ok && now.Sub(last) < minInvalidateInterval {
		s.lastInvalidateMu.Unlock()
		return
	}

	s.lastInvalidate[queueID] = now
	s.lastInvalidateMu.Unlock()

	cacheKey := statsCacheKey(queueID)
	_ = s.rawCache.Delete(ctx, cacheKey)
}

func (s *queueStatsService) computeStats(ctx context.Context, queueID string) (*QueueStats, error) {
	stats := &QueueStats{}

	var err error

	stats.TotalWaiting, err = s.itemRepo.CountByStatus(ctx, queueID, models.ItemStatusWaiting)
	if err != nil {
		return nil, fmt.Errorf("count waiting: %w", err)
	}

	stats.TotalServing, err = s.itemRepo.CountByStatus(ctx, queueID, models.ItemStatusServing)
	if err != nil {
		return nil, fmt.Errorf("count serving: %w", err)
	}

	todayStart := time.Now().Truncate(24 * time.Hour)

	stats.AvgWaitMinutes, err = s.itemRepo.AvgWaitMinutes(ctx, queueID, todayStart)
	if err != nil {
		return nil, fmt.Errorf("avg wait: %w", err)
	}

	stats.LongestWait, err = s.itemRepo.LongestWaitMinutes(ctx, queueID)
	if err != nil {
		return nil, fmt.Errorf("longest wait: %w", err)
	}

	stats.TodayServed, err = s.itemRepo.CountByStatusSince(ctx, queueID, models.ItemStatusCompleted, todayStart)
	if err != nil {
		return nil, fmt.Errorf("today served: %w", err)
	}

	stats.TodayCancelled, err = s.itemRepo.CountByStatusSince(ctx, queueID, models.ItemStatusCancelled, todayStart)
	if err != nil {
		return nil, fmt.Errorf("today cancelled: %w", err)
	}

	stats.TodayNoShow, err = s.itemRepo.CountByStatusSince(ctx, queueID, models.ItemStatusNoShow, todayStart)
	if err != nil {
		return nil, fmt.Errorf("today no-show: %w", err)
	}

	stats.OpenCounters, err = s.counterRepo.CountOpen(ctx, queueID)
	if err != nil {
		return nil, fmt.Errorf("open counters: %w", err)
	}

	// Little's Law estimate: TotalWaiting * AvgServiceTime / OpenCounters.
	if stats.OpenCounters > 0 && stats.AvgWaitMinutes > 0 {
		stats.EstimatedWait = float64(stats.TotalWaiting) * stats.AvgWaitMinutes / float64(stats.OpenCounters)
	}

	return stats, nil
}
