package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	f4rgesession "github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/session"
)

type InferenceMessage struct {
	Role       string              `json:"role"`
	Content    string              `json:"content,omitempty"`
	ToolCalls  []InferenceToolCall `json:"toolCalls,omitempty"`
	ToolCallID string              `json:"toolCallId,omitempty"`
}

type InferenceToolCall struct {
	ToolCallID       string          `json:"toolCallId"`
	Name             string          `json:"name"`
	ArgumentsJSON    string          `json:"argumentsJson"`
	ProviderMetadata json.RawMessage `json:"providerMetadata,omitempty"`
}

type InferenceTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
	Risk        string         `json:"risk,omitempty"`
}

type InferenceToolResult struct {
	ToolCallID string         `json:"toolCallId"`
	Name       string         `json:"name"`
	Content    string         `json:"content"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type InferenceRequest struct {
	SchemaVersion  int                   `json:"schemaVersion"`
	RequestID      string                `json:"requestId"`
	Surface        string                `json:"surface"`
	OrganizationID string                `json:"organizationId"`
	SessionID      string                `json:"sessionId,omitempty"`
	RunID          string                `json:"runId,omitempty"`
	ThreadID       string                `json:"threadId,omitempty"`
	ModelID        string                `json:"modelId"`
	PromptMode     string                `json:"promptMode"`
	Messages       []InferenceMessage    `json:"messages"`
	Tools          []InferenceTool       `json:"tools,omitempty"`
	ToolResults    []InferenceToolResult `json:"toolResults,omitempty"`
	Metadata       map[string]any        `json:"metadata,omitempty"`
}

type InferenceSmokeResult struct {
	RequestID string
	Text      string
}

func (c Client) SmokeInference(ctx context.Context, session *f4rgesession.ManagedSession, modelID string) (*InferenceSmokeResult, error) {
	if modelID == "" {
		return nil, fmt.Errorf("model ID is required")
	}
	requestID := "cli_smoke_" + modelID
	body, err := c.StreamInference(ctx, session, InferenceRequest{
		RequestID:      requestID,
		Surface:        "cli",
		OrganizationID: session.OrganizationID,
		SessionID:      session.RuntimeSessionID,
		ModelID:        modelID,
		PromptMode:     "ask",
		Messages: []InferenceMessage{{
			Role:    "user",
			Content: "Reply with exactly: ok",
		}},
	})
	if err != nil {
		return nil, err
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	return &InferenceSmokeResult{
		RequestID: requestID,
		Text:      string(data),
	}, nil
}

func (c Client) StreamInference(ctx context.Context, session *f4rgesession.ManagedSession, request InferenceRequest) (io.ReadCloser, error) {
	if session == nil || session.AccessToken == "" {
		return nil, fmt.Errorf("F4RGE sign-in required")
	}
	if request.Surface == "" {
		request.Surface = "cli"
	}
	if request.OrganizationID == "" {
		request.OrganizationID = session.OrganizationID
	}
	if request.SessionID == "" {
		request.SessionID = session.RuntimeSessionID
	}
	if request.SchemaVersion == 0 {
		request.SchemaVersion = 1
	}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	endpoint, err := c.endpoint("/api/runtime/inference/stream")
	if err != nil {
		return nil, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Accept", "text/event-stream")
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+session.AccessToken)

	client := c.Stream
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		defer response.Body.Close()
		data, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return nil, fmt.Errorf("F4RGE Gateway returned %s: %s", response.Status, string(data))
	}
	return response.Body, nil
}
