package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"github.com/antinvestor/service-trustage/connector"
)

const (
	aiChatType        = "ai.chat"
	aiChatDisplayName = "AI Chat"
)

// AIChatAdapter sends messages to an LLM provider via the OpenAI-compatible API.
// Provider-agnostic: point base_url to any OpenAI-compatible gateway to access different LLMs.
type AIChatAdapter struct{}

// NewAIChatAdapter creates a new AIChatAdapter.
func NewAIChatAdapter() *AIChatAdapter {
	return &AIChatAdapter{}
}

func (a *AIChatAdapter) Type() string        { return aiChatType }
func (a *AIChatAdapter) DisplayName() string { return aiChatDisplayName }

func (a *AIChatAdapter) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["messages"],
		"properties": {
			"messages": {
				"type": "array",
				"items": {
					"type": "object",
					"required": ["role", "content"],
					"properties": {
						"role": {"type": "string", "enum": ["user", "assistant"]},
						"content": {"type": "string"}
					}
				},
				"description": "Conversation message history"
			},
			"system": {
				"type": "string",
				"description": "System prompt for the LLM"
			}
		}
	}`)
}

func (a *AIChatAdapter) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["model"],
		"properties": {
			"model": {
				"type": "string",
				"description": "Model identifier (e.g. gpt-4o, claude-sonnet-4-20250514)"
			},
			"base_url": {
				"type": "string",
				"format": "uri",
				"description": "API base URL (defaults to OpenAI; set to gateway URL for other providers)"
			},
			"temperature": {
				"type": "number",
				"minimum": 0,
				"maximum": 2,
				"description": "Sampling temperature"
			},
			"max_tokens": {
				"type": "integer",
				"minimum": 1,
				"description": "Maximum response tokens"
			}
		}
	}`)
}

func (a *AIChatAdapter) OutputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["content", "model"],
		"properties": {
			"content": {"type": "string", "description": "Generated response text"},
			"model": {"type": "string", "description": "Model that produced the response"}
		}
	}`)
}

func (a *AIChatAdapter) Validate(req *connector.ExecuteRequest) error {
	if req.Input["messages"] == nil {
		return errors.New("messages is required")
	}

	if _, ok := req.Input["messages"].([]any); !ok {
		return errors.New("messages must be an array")
	}

	model, _ := req.Config["model"].(string)
	if model == "" {
		return errors.New("config.model is required")
	}

	return nil
}

func (a *AIChatAdapter) Execute(
	ctx context.Context,
	req *connector.ExecuteRequest,
) (*connector.ExecuteResponse, *connector.ExecutionError) {
	model, _ := req.Config["model"].(string)
	apiKey := req.Credentials["api_key"]

	if model == "" {
		return nil, &connector.ExecutionError{
			Class:   connector.ErrorFatal,
			Code:    "CONFIG_ERROR",
			Message: "model is required in config",
		}
	}

	if apiKey == "" {
		return nil, &connector.ExecutionError{
			Class:   connector.ErrorFatal,
			Code:    "CREDENTIALS_ERROR",
			Message: "api_key credential is required",
		}
	}

	// Build client options.
	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
	}

	if baseURL, ok := req.Config["base_url"].(string); ok && baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}

	if httpClient, ok := req.Config["http_client"].(*http.Client); ok {
		opts = append(opts, option.WithHTTPClient(httpClient))
	}

	client := openai.NewClient(opts...)

	// Build messages.
	messages, execErr := buildMessages(req.Input)
	if execErr != nil {
		return nil, execErr
	}

	// Build completion params.
	params := openai.ChatCompletionNewParams{
		Model:    model,
		Messages: messages,
	}

	if temp, ok := req.Config["temperature"].(float64); ok {
		params.Temperature = openai.Float(temp)
	}

	if maxTokens, ok := req.Config["max_tokens"].(float64); ok {
		params.MaxTokens = openai.Int(int64(maxTokens))
	}

	completion, err := client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, classifyLLMError(err)
	}

	content := ""
	if len(completion.Choices) > 0 {
		content = completion.Choices[0].Message.Content
	}

	return &connector.ExecuteResponse{
		Output: map[string]any{
			"content": content,
			"model":   completion.Model,
		},
	}, nil
}

func buildMessages(input map[string]any) ([]openai.ChatCompletionMessageParamUnion, *connector.ExecutionError) {
	rawMessages, _ := input["messages"].([]any)
	systemPrompt, _ := input["system"].(string)

	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(rawMessages)+1)

	if systemPrompt != "" {
		messages = append(messages, openai.SystemMessage(systemPrompt))
	}

	for i, raw := range rawMessages {
		msg, ok := raw.(map[string]any)
		if !ok {
			return nil, &connector.ExecutionError{
				Class:   connector.ErrorFatal,
				Code:    "INPUT_ERROR",
				Message: fmt.Sprintf("messages[%d]: expected object", i),
			}
		}

		content, _ := msg["content"].(string)
		role, _ := msg["role"].(string)

		switch role {
		case "user":
			messages = append(messages, openai.UserMessage(content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(content))
		default:
			return nil, &connector.ExecutionError{
				Class:   connector.ErrorFatal,
				Code:    "INPUT_ERROR",
				Message: fmt.Sprintf("messages[%d]: role must be 'user' or 'assistant', got %q", i, role),
			}
		}
	}

	return messages, nil
}

// classifyLLMError maps OpenAI API errors to connector error classes.
func classifyLLMError(err error) *connector.ExecutionError {
	msg := err.Error()

	switch {
	case containsAny(msg, "authentication", "unauthorized", "invalid api key", "invalid_api_key", "401"):
		return &connector.ExecutionError{
			Class:   connector.ErrorFatal,
			Code:    "AUTH_ERROR",
			Message: fmt.Sprintf("LLM authentication failed: %s", truncateError(msg)),
		}
	case containsAny(msg, "rate limit", "rate_limit", "429", "too many requests"):
		return &connector.ExecutionError{
			Class:   connector.ErrorRetryable,
			Code:    "RATE_LIMITED",
			Message: fmt.Sprintf("LLM rate limited: %s", truncateError(msg)),
		}
	case containsAny(msg, "timeout", "deadline exceeded", "context deadline"):
		return &connector.ExecutionError{
			Class:   connector.ErrorRetryable,
			Code:    "TIMEOUT",
			Message: fmt.Sprintf("LLM request timed out: %s", truncateError(msg)),
		}
	case containsAny(msg, "context canceled"):
		return &connector.ExecutionError{
			Class:   connector.ErrorRetryable,
			Code:    "CANCELLED",
			Message: "LLM request was cancelled",
		}
	case containsAny(msg, "connection refused", "no such host", "dns", "unreachable"):
		return &connector.ExecutionError{
			Class:   connector.ErrorExternalDependency,
			Code:    "CONNECTION_ERROR",
			Message: fmt.Sprintf("LLM provider unreachable: %s", truncateError(msg)),
		}
	case containsAny(msg, "500", "502", "503", "504", "internal server error", "service unavailable", "bad gateway"):
		return &connector.ExecutionError{
			Class:   connector.ErrorExternalDependency,
			Code:    "PROVIDER_ERROR",
			Message: fmt.Sprintf("LLM provider error: %s", truncateError(msg)),
		}
	case containsAny(msg, "invalid", "bad request", "400", "422", "not found", "model"):
		return &connector.ExecutionError{
			Class:   connector.ErrorFatal,
			Code:    "REQUEST_ERROR",
			Message: fmt.Sprintf("LLM request error: %s", truncateError(msg)),
		}
	default:
		return &connector.ExecutionError{
			Class:   connector.ErrorExternalDependency,
			Code:    "LLM_ERROR",
			Message: fmt.Sprintf("LLM call failed: %s", truncateError(msg)),
		}
	}
}

// containsAny checks if s contains any of the given substrings (case-insensitive).
func containsAny(s string, substrs ...string) bool {
	lower := strings.ToLower(s)
	for _, sub := range substrs {
		if strings.Contains(lower, sub) {
			return true
		}
	}
	return false
}

// truncateError limits error messages to 512 characters.
func truncateError(s string) string {
	const maxLen = 512
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
