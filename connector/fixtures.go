package connector

// AdapterFixture contains minimal valid inputs for adapter validation and test coverage.
type AdapterFixture struct {
	Input       map[string]any
	Config      map[string]any
	Credentials map[string]string
}

// AdapterFixtures provides minimal valid payloads for each supported adapter type.
// These are intentionally small and deterministic so tests and tooling can validate
// adapter input expectations across the codebase.
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
			"amount":    42.5,
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
			"source":     map[string]any{"amount": 15},
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
