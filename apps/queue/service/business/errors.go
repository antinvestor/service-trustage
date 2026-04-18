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
