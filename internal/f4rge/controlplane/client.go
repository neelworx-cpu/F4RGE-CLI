package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	f4rgesession "github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/session"
)

const defaultPlatformURL = "https://api.4rged.ai"

type Client struct {
	BaseURL string
	HTTP    *http.Client
	Stream  *http.Client
}

type RuntimeSessionRequest struct {
	Surface        string `json:"surface"`
	DeviceID       string `json:"deviceId,omitempty"`
	DeviceLabel    string `json:"deviceLabel"`
	Platform       string `json:"platform"`
	ClientVersion  string `json:"clientVersion,omitempty"`
	OrganizationID string `json:"organizationId,omitempty"`
}

type RuntimeSessionResponse struct {
	Session RuntimeSession `json:"session"`
}

type CLIAuthStartResponse struct {
	DeviceCode       string `json:"deviceCode"`
	UserCode         string `json:"userCode"`
	VerificationPath string `json:"verificationPath"`
	ExpiresIn        int    `json:"expiresIn"`
	Interval         int    `json:"interval"`
}

type CLIAuthPollResponse struct {
	Status         string         `json:"status"`
	RuntimeSession RuntimeSession `json:"runtimeSession"`
	OrganizationID string         `json:"organizationId"`
	CompletedAt    string         `json:"completedAt"`
}

type RuntimeSession struct {
	SessionID        string   `json:"sessionId"`
	Surface          string   `json:"surface"`
	Kind             string   `json:"kind"`
	OrganizationID   string   `json:"organizationId"`
	SubjectUserID    string   `json:"subjectUserId,omitempty"`
	DeviceID         string   `json:"deviceId,omitempty"`
	DeviceLabel      string   `json:"deviceLabel,omitempty"`
	ClientVersion    string   `json:"clientVersion,omitempty"`
	Scopes           []string `json:"scopes"`
	Status           string   `json:"status"`
	Token            string   `json:"token"`
	ExpiresAt        string   `json:"expiresAt"`
	ModelCatalogHash string   `json:"modelCatalogHash,omitempty"`
}

func New() Client {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("F4RGE_PLATFORM_URL")), "/")
	if baseURL == "" {
		baseURL = defaultPlatformURL
	}
	return Client{
		BaseURL: baseURL,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
		Stream:  &http.Client{},
	}
}

func (c Client) endpoint(path string) (string, error) {
	base, err := url.Parse(c.BaseURL)
	if err != nil {
		return "", fmt.Errorf("parse F4RGE platform URL: %w", err)
	}
	relative, err := url.Parse(path)
	if err != nil {
		return "", err
	}
	return base.ResolveReference(relative).String(), nil
}

func (c Client) RegisterRuntimeSession(session *f4rgesession.ManagedSession, request RuntimeSessionRequest) (*RuntimeSessionResponse, error) {
	if session == nil || session.AccessToken == "" {
		return nil, fmt.Errorf("F4RGE sign-in required")
	}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	endpoint, err := c.endpoint("/api/runtime/sessions")
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

	var response RuntimeSessionResponse
	if err := c.doJSON(httpRequest, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c Client) StartCLIAuth(request RuntimeSessionRequest) (*CLIAuthStartResponse, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	endpoint, err := c.endpoint("/api/runtime/cli-auth/start")
	if err != nil {
		return nil, err
	}
	httpRequest, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Content-Type", "application/json")
	var response CLIAuthStartResponse
	if err := c.doJSON(httpRequest, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c Client) PollCLIAuth(deviceCode string) (*CLIAuthPollResponse, error) {
	var response CLIAuthPollResponse
	if err := c.GetJSONWithToken("", "/api/runtime/cli-auth/poll", map[string]string{
		"deviceCode": deviceCode,
	}, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c Client) GetJSON(session *f4rgesession.ManagedSession, path string, query map[string]string, target any) error {
	if session == nil || session.AccessToken == "" {
		return fmt.Errorf("F4RGE sign-in required")
	}
	endpoint, err := c.endpoint(path)
	if err != nil {
		return err
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	values := parsed.Query()
	for key, value := range query {
		if value != "" {
			values.Set(key, value)
		}
	}
	parsed.RawQuery = values.Encode()
	httpRequest, err := http.NewRequestWithContext(context.Background(), http.MethodGet, parsed.String(), nil)
	if err != nil {
		return err
	}
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+session.AccessToken)
	return c.doJSON(httpRequest, target)
}

func (c Client) GetJSONWithToken(token string, path string, query map[string]string, target any) error {
	endpoint, err := c.endpoint(path)
	if err != nil {
		return err
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	values := parsed.Query()
	for key, value := range query {
		if value != "" {
			values.Set(key, value)
		}
	}
	parsed.RawQuery = values.Encode()
	httpRequest, err := http.NewRequestWithContext(context.Background(), http.MethodGet, parsed.String(), nil)
	if err != nil {
		return err
	}
	httpRequest.Header.Set("Accept", "application/json")
	if token != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+token)
	}
	return c.doJSON(httpRequest, target)
}

func (c Client) doJSON(request *http.Request, target any) error {
	response, err := c.HTTP.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("F4RGE platform returned %s: %s", response.Status, strings.TrimSpace(string(data)))
	}
	if target == nil {
		return nil
	}
	return json.NewDecoder(response.Body).Decode(target)
}
