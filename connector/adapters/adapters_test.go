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

package adapters_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/antinvestor/service-trustage/connector"
	"github.com/antinvestor/service-trustage/connector/adapters"
)

func TestAdapterFixtures_ValidateAll(t *testing.T) {
	registry := connector.NewRegistry()
	client := &http.Client{}
	adapters := []connector.Adapter{
		adapters.NewWebhookAdapter(client),
		adapters.NewHTTPAdapter(client),
		adapters.NewNotificationSendAdapter(client),
		adapters.NewNotificationStatusAdapter(client),
		adapters.NewPaymentInitiateAdapter(client),
		adapters.NewPaymentVerifyAdapter(client),
		adapters.NewDataTransformAdapter(),
		adapters.NewLogEntryAdapter(),
		adapters.NewFormValidateAdapter(),
		adapters.NewApprovalRequestAdapter(client),
		adapters.NewAIChatAdapter(),
	}

	for _, a := range adapters {
		if err := registry.Register(a); err != nil {
			t.Fatalf("register %s: %v", a.Type(), err)
		}

		fixture, ok := connector.AdapterFixtures[a.Type()]
		if !ok {
			t.Fatalf("missing fixture for adapter %s", a.Type())
		}

		if err := json.Unmarshal(a.InputSchema(), &map[string]any{}); err != nil {
			t.Fatalf("invalid input schema for %s: %v", a.Type(), err)
		}
		if err := json.Unmarshal(a.ConfigSchema(), &map[string]any{}); err != nil {
			t.Fatalf("invalid config schema for %s: %v", a.Type(), err)
		}
		if err := json.Unmarshal(a.OutputSchema(), &map[string]any{}); err != nil {
			t.Fatalf("invalid output schema for %s: %v", a.Type(), err)
		}

		req := &connector.ExecuteRequest{
			Input:       fixture.Input,
			Config:      fixture.Config,
			Credentials: fixture.Credentials,
		}
		if err := a.Validate(req); err != nil {
			t.Fatalf("validate failed for %s: %v", a.Type(), err)
		}
	}
}

func TestAdapters_ValidateMissingRequiredFields(t *testing.T) {
	client := &http.Client{}

	cases := []struct {
		name    string
		adapter connector.Adapter
		input   map[string]any
		config  map[string]any
	}{
		{"webhook missing url", adapters.NewWebhookAdapter(client), map[string]any{}, map[string]any{}},
		{"http missing url", adapters.NewHTTPAdapter(client), map[string]any{"method": "GET"}, map[string]any{}},
		{
			"notification missing recipient",
			adapters.NewNotificationSendAdapter(client),
			map[string]any{"channel": "email", "body": "x"},
			map[string]any{},
		},
		{
			"notification status missing id",
			adapters.NewNotificationStatusAdapter(client),
			map[string]any{},
			map[string]any{},
		},
		{
			"payment initiate missing amount",
			adapters.NewPaymentInitiateAdapter(client),
			map[string]any{"currency": "USD", "recipient": "x", "reference": "r"},
			map[string]any{},
		},
		{"payment verify missing id", adapters.NewPaymentVerifyAdapter(client), map[string]any{}, map[string]any{}},
		{
			"transform missing source",
			adapters.NewDataTransformAdapter(),
			map[string]any{"expression": "payload.x"},
			map[string]any{},
		},
		{"log missing level", adapters.NewLogEntryAdapter(), map[string]any{"message": "x"}, map[string]any{}},
		{
			"form validate missing fields",
			adapters.NewFormValidateAdapter(),
			map[string]any{"required_fields": []any{"a"}},
			map[string]any{},
		},
		{
			"approval missing approver",
			adapters.NewApprovalRequestAdapter(client),
			map[string]any{"title": "x"},
			map[string]any{},
		},
		{
			"ai chat missing messages",
			adapters.NewAIChatAdapter(),
			map[string]any{},
			map[string]any{"provider": "openai", "model": "gpt-4o"},
		},
	}

	for _, c := range cases {
		req := &connector.ExecuteRequest{Input: c.input, Config: c.config}
		if err := c.adapter.Validate(req); err == nil {
			t.Fatalf("expected validation error for %s", c.name)
		}
	}
}
