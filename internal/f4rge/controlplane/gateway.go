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
	Role       string `json:"role"`
	Content    string `json:"content"`
	ToolCallID string `json:"toolCallId,omitempty"`
}

type InferenceRequest struct {
	SchemaVersion  int                `json:"schemaVersion"`
	RequestID      string             `json:"requestId"`
	Surface        string             `json:"surface"`
	OrganizationID string             `json:"organizationId"`
	SessionID      string             `json:"sessionId,omitempty"`
	RunID          string             `json:"runId,omitempty"`
	ThreadID       string             `json:"threadId,omitempty"`
	ModelID        string             `json:"modelId"`
	PromptMode     string             `json:"promptMode"`
	Messages       []InferenceMessage `json:"messages"`
	Metadata       map[string]any     `json:"metadata,omitempty"`
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

	response, err := c.HTTP.Do(httpRequest)
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
