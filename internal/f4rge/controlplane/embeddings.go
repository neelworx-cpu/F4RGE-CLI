package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	f4rgesession "github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/session"
)

type EmbeddingChunk struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	Path      string `json:"path,omitempty"`
	StartLine int    `json:"startLine,omitempty"`
	EndLine   int    `json:"endLine,omitempty"`
	SHA256    string `json:"sha256,omitempty"`
}

type EmbeddingRequest struct {
	SchemaVersion  int              `json:"schemaVersion"`
	RequestID      string           `json:"requestId"`
	Surface        string           `json:"surface"`
	OrganizationID string           `json:"organizationId"`
	SessionID      string           `json:"sessionId,omitempty"`
	RepositoryID   string           `json:"repositoryId,omitempty"`
	WorkspaceID    string           `json:"workspaceId,omitempty"`
	StorageMode    string           `json:"storageMode"`
	Purpose        string           `json:"purpose"`
	Chunks         []EmbeddingChunk `json:"chunks"`
}

type EmbeddingResponse struct {
	RequestID string `json:"requestId"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	Vectors   []struct {
		ChunkID    string    `json:"chunkId"`
		Embedding  []float64 `json:"embedding"`
		Dimensions int       `json:"dimensions"`
		Model      string    `json:"model"`
	} `json:"vectors"`
}

func (c Client) Embed(session *f4rgesession.ManagedSession, request EmbeddingRequest) (*EmbeddingResponse, error) {
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
	if request.StorageMode == "" {
		request.StorageMode = "local"
	}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	endpoint, err := c.endpoint("/api/runtime/embeddings")
	if err != nil {
		return nil, err
	}
	httpRequest, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+session.AccessToken)

	var response EmbeddingResponse
	if err := c.doJSON(httpRequest, &response); err != nil {
		return nil, err
	}
	return &response, nil
}
