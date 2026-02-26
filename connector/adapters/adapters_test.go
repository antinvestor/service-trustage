package adapters

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/antinvestor/service-trustage/connector"
)

func TestAdapterFixtures_ValidateAll(t *testing.T) {
	registry := connector.NewRegistry()
	client := &http.Client{}
	adapters := []connector.Adapter{
		NewWebhookAdapter(client),
		NewHTTPAdapter(client),
		NewNotificationSendAdapter(client),
		NewNotificationStatusAdapter(client),
		NewPaymentInitiateAdapter(client),
		NewPaymentVerifyAdapter(client),
		NewDataTransformAdapter(),
		NewLogEntryAdapter(),
		NewFormValidateAdapter(),
		NewApprovalRequestAdapter(client),
		NewAIChatAdapter(),
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
		{"webhook missing url", NewWebhookAdapter(client), map[string]any{}, map[string]any{}},
		{"http missing url", NewHTTPAdapter(client), map[string]any{"method": "GET"}, map[string]any{}},
		{"notification missing recipient", NewNotificationSendAdapter(client), map[string]any{"channel": "email", "body": "x"}, map[string]any{}},
		{"notification status missing id", NewNotificationStatusAdapter(client), map[string]any{}, map[string]any{}},
		{"payment initiate missing amount", NewPaymentInitiateAdapter(client), map[string]any{"currency": "USD", "recipient": "x", "reference": "r"}, map[string]any{}},
		{"payment verify missing id", NewPaymentVerifyAdapter(client), map[string]any{}, map[string]any{}},
		{"transform missing source", NewDataTransformAdapter(), map[string]any{"expression": "payload.x"}, map[string]any{}},
		{"log missing level", NewLogEntryAdapter(), map[string]any{"message": "x"}, map[string]any{}},
		{"form validate missing fields", NewFormValidateAdapter(), map[string]any{"required_fields": []any{"a"}}, map[string]any{}},
		{"approval missing approver", NewApprovalRequestAdapter(client), map[string]any{"title": "x"}, map[string]any{}},
		{"ai chat missing messages", NewAIChatAdapter(), map[string]any{}, map[string]any{"provider": "openai", "model": "gpt-4o"}},
	}

	for _, c := range cases {
		req := &connector.ExecuteRequest{Input: c.input, Config: c.config}
		if err := c.adapter.Validate(req); err == nil {
			t.Fatalf("expected validation error for %s", c.name)
		}
	}
}
