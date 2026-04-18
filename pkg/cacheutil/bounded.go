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

package cacheutil

import "sync"

// BoundedCache is a thread-safe cache with a maximum size.
// When the cache is full, the oldest entry (by insertion order) is evicted.
type BoundedCache[V any] struct {
	mu      sync.RWMutex
	items   map[string]V
	order   []string // insertion order for eviction
	maxSize int
}

// NewBoundedCache creates a new BoundedCache with the given maximum size.
func NewBoundedCache[V any](maxSize int) *BoundedCache[V] {
	return &BoundedCache[V]{
		items:   make(map[string]V, maxSize),
		order:   make([]string, 0, maxSize),
		maxSize: maxSize,
	}
}

// Get returns the cached value and true, or the zero value and false if not found.
func (c *BoundedCache[V]) Get(key string) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	v, ok := c.items[key]
	return v, ok
}

// Put stores a value in the cache, evicting the oldest entry if the cache is full.
func (c *BoundedCache[V]) Put(key string, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If key already exists, just update the value.
	if _, exists := c.items[key]; exists {
		c.items[key] = value
		return
	}

	// Evict oldest entries if at capacity.
	for len(c.items) >= c.maxSize && len(c.order) > 0 {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.items, oldest)
	}

	c.items[key] = value
	c.order = append(c.order, key)
}

// Len returns the number of entries in the cache.
func (c *BoundedCache[V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}
