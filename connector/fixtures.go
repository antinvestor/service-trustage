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

// AdapterFixture contains minimal valid inputs for adapter validation and test coverage.
type AdapterFixture struct {
	Input       map[string]any
	Config      map[string]any
	Credentials map[string]string
}

const (
	fixturePaymentAmount   = 42.5
	fixtureTransformAmount = 15
)

// AdapterFixtures provides minimal valid payloads for each supported adapter type.
// These are intentionally small and deterministic so tests and tooling can validate
// adapter input expectations across the codebase.
//
//nolint:gochecknoglobals // adapter fixtures are static test data
var AdapterFixtures = map[string]AdapterFixture{
	"ai.chat": {
		Input: map[string]any{
			"messages": []any{
				map[string]any{"role": "user", "content": "hello"},
			},
		},
		Config: map[string]any{
			"provider": "openai",
			"model":    "gpt-4o",
		},
	},
	"webhook.call": {
		Input: map[string]any{
			"url":    "https://example.com/hook",
			"method": "POST",
			"body":   map[string]any{"ok": true},
		},
		Config: map[string]any{},
	},
	"http.request": {
		Input: map[string]any{
			"url":    "https://example.com/api",
			"method": "POST",
			"body":   map[string]any{"ping": "pong"},
		},
		Config: map[string]any{},
	},
	"notification.send": {
		Input: map[string]any{
			"recipient": "user@example.com",
			"channel":   "email",
			"subject":   "Hello",
			"body":      "Welcome",
		},
		Config: map[string]any{
			"api_url": "https://notify.example.com",
		},
	},
	"notification.status": {
		Input: map[string]any{
			"notification_id": "notif-123",
		},
		Config: map[string]any{
			"api_url": "https://notify.example.com",
		},
	},
	"payment.initiate": {
		Input: map[string]any{
			"amount":    fixturePaymentAmount,
			"currency":  "USD",
			"recipient": "+1555000111",
			"reference": "ref-001",
			"method":    "mobile_money",
		},
		Config: map[string]any{
			"api_url": "https://payments.example.com",
		},
	},
	"payment.verify": {
		Input: map[string]any{
			"payment_id": "pay-001",
		},
		Config: map[string]any{
			"api_url": "https://payments.example.com",
		},
	},
	"data.transform": {
		Input: map[string]any{
			"source":     map[string]any{"amount": fixtureTransformAmount},
			"expression": "payload.amount > 10",
		},
		Config: map[string]any{},
	},
	"log.entry": {
		Input: map[string]any{
			"level":   "info",
			"message": "hello",
			"data":    map[string]any{"trace": "t1"},
		},
		Config: map[string]any{},
	},
	"form.validate": {
		Input: map[string]any{
			"fields":          map[string]any{"email": "user@example.com"},
			"required_fields": []any{"email"},
			"field_types":     map[string]any{"email": "string"},
		},
		Config: map[string]any{},
	},
	"approval.request": {
		Input: map[string]any{
			"approver":    "approver@example.com",
			"title":       "Approve",
			"description": "Please approve",
		},
		Config: map[string]any{
			"api_url": "https://approval.example.com",
		},
	},
}
