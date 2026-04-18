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

package connector

import (
	"fmt"
	"sync"
)

// Registry holds registered adapter implementations.
type Registry struct {
	mu       sync.RWMutex
	adapters map[string]Adapter
}

// NewRegistry creates a new empty adapter registry.
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]Adapter),
	}
}

// Register adds an adapter to the registry.
func (r *Registry) Register(adapter Adapter) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	t := adapter.Type()
	if _, exists := r.adapters[t]; exists {
		return fmt.Errorf("adapter %q already registered", t)
	}

	r.adapters[t] = adapter

	return nil
}

// Get returns an adapter by type, or an error if not found.
func (r *Registry) Get(adapterType string) (Adapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, ok := r.adapters[adapterType]
	if !ok {
		return nil, fmt.Errorf("adapter %q not registered", adapterType)
	}

	return adapter, nil
}

// List returns the types of all registered adapters.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.adapters))
	for t := range r.adapters {
		types = append(types, t)
	}

	return types
}
