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

package config //nolint:testpackage // white-box test: exercises unexported injectQueryParam

import (
	"net/url"
	"testing"
)

func TestInjectQueryParam_ReplacesExisting(t *testing.T) {
	raw := "nats://host:4222?jetstream=true&consumer_max_ack_pending=500"
	got := injectQueryParam(raw, "consumer_max_ack_pending", 1000)
	u, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get("consumer_max_ack_pending") != "1000" {
		t.Fatalf("expected consumer_max_ack_pending=1000, got %q", u.Query().Get("consumer_max_ack_pending"))
	}
	if u.Query().Get("jetstream") != "true" {
		t.Fatalf("other params dropped: %q", got)
	}
}

func TestInjectQueryParam_AddsMissing(t *testing.T) {
	raw := "nats://host:4222?jetstream=true"
	got := injectQueryParam(raw, "consumer_max_ack_pending", 1000)
	if got == raw {
		t.Fatalf("expected query param to be added, got %q", got)
	}
	u, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get("consumer_max_ack_pending") != "1000" {
		t.Fatalf("expected consumer_max_ack_pending=1000, got %q", u.Query().Get("consumer_max_ack_pending"))
	}
}

func TestInjectQueryParam_ZeroValueSkips(t *testing.T) {
	raw := "nats://host:4222?consumer_max_ack_pending=500"
	got := injectQueryParam(raw, "consumer_max_ack_pending", 0)
	if got != raw {
		t.Fatalf("zero should skip; got %q", got)
	}
}

func TestInjectQueryParam_NegativeValueSkips(t *testing.T) {
	raw := "nats://host:4222?consumer_max_ack_pending=500"
	got := injectQueryParam(raw, "consumer_max_ack_pending", -1)
	if got != raw {
		t.Fatalf("negative value should skip; got %q", got)
	}
}

func TestInjectQueryParam_InvalidURLFallsBack(t *testing.T) {
	raw := ":::not-a-url"
	got := injectQueryParam(raw, "k", 1)
	if got != raw {
		t.Fatalf("invalid URL should return unchanged; got %q", got)
	}
}
