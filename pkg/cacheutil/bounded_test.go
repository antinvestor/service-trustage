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

//nolint:testpackage // package-local tests exercise unexported cache helpers intentionally.
package cacheutil

import "testing"

func TestBoundedCache(t *testing.T) {
	t.Parallel()

	cache := NewBoundedCache[int](2)
	cache.Put("a", 1)
	cache.Put("b", 2)

	if got, ok := cache.Get("a"); !ok || got != 1 {
		t.Fatalf("Get(a) = %v, %v", got, ok)
	}

	cache.Put("c", 3)
	if _, ok := cache.Get("a"); ok {
		t.Fatal("expected oldest item to be evicted")
	}

	cache.Put("b", 20)
	if got, ok := cache.Get("b"); !ok || got != 20 {
		t.Fatalf("Get(b) = %v, %v", got, ok)
	}

	if cache.Len() != 2 {
		t.Fatalf("Len() = %d", cache.Len())
	}
}
