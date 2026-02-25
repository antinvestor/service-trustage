package business

import "errors"

// Sentinel errors for the queue business layer.
var (
	ErrQueueNotFound      = errors.New("queue not found")
	ErrQueueItemNotFound  = errors.New("queue item not found")
	ErrCounterNotFound    = errors.New("counter not found")
	ErrQueueFull          = errors.New("queue is at maximum capacity")
	ErrNoWaitingItems     = errors.New("no waiting items in queue")
	ErrCounterNotOpen     = errors.New("counter is not open")
	ErrCounterBusy        = errors.New("counter is currently serving another item")
	ErrCounterNotServing  = errors.New("counter is not currently serving any item")
	ErrInvalidTransition  = errors.New("invalid status transition")
	ErrDuplicateQueueName = errors.New("queue name already exists")
	ErrItemNotWaiting     = errors.New("item is not in waiting status")
	ErrItemNotNoShow      = errors.New("item is not in no_show status for re-queue")
)
