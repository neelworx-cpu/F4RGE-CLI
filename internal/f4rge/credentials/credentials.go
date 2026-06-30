package credentials

import (
	"fmt"
	"sync"
	"time"

	"github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/controlplane"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/modelcatalog"
	f4rgesession "github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/session"
)

const defaultTTLSeconds = 600

type IssueRequest struct {
	ModelID       string
	PromptMode    string
	RunID         string
	ThreadID      string
	TTLSeconds    int
	Catalog       *modelcatalog.Bundle
	RuntimePolicy string
}

type Secret struct {
	Kind   string `json:"kind"`
	Value  string `json:"value"`
	EnvKey string `json:"envKey,omitempty"`
}

type Route struct {
	BaseURL     string            `json:"baseUrl,omitempty"`
	Endpoint    string            `json:"endpoint,omitempty"`
	APIVersion  string            `json:"apiVersion,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	ExtraParams map[string]string `json:"extraParams,omitempty"`
}

type Lease struct {
	SchemaVersion        int                `json:"schemaVersion"`
	LeaseID              string             `json:"leaseId"`
	Status               string             `json:"status"`
	Surface              string             `json:"surface"`
	OrganizationID       string             `json:"organizationId"`
	TeamID               string             `json:"teamId,omitempty"`
	ProjectID            string             `json:"projectId,omitempty"`
	SessionID            string             `json:"sessionId"`
	RunID                string             `json:"runId,omitempty"`
	ThreadID             string             `json:"threadId,omitempty"`
	RequestedModelID     string             `json:"requestedModelId"`
	ResolvedModelID      string             `json:"resolvedModelId"`
	Provider             string             `json:"provider"`
	APIFamily            string             `json:"apiFamily"`
	ProviderModelID      string             `json:"providerModelId"`
	DeploymentName       string             `json:"deploymentName,omitempty"`
	CredentialCapability string             `json:"credentialCapability"`
	Secret               Secret             `json:"secret"`
	Route                Route              `json:"route"`
	CatalogVersion       string             `json:"catalogVersion,omitempty"`
	CatalogHash          string             `json:"catalogHash,omitempty"`
	PolicyVersion        string             `json:"policyVersion,omitempty"`
	IssuedAt             string             `json:"issuedAt"`
	ExpiresAt            string             `json:"expiresAt"`
	Metadata             map[string]any     `json:"metadata,omitempty"`
	ResolvedModel        modelcatalog.Model `json:"-"`
}

type issueResponse struct {
	Lease Lease `json:"lease"`
}

type issuePayload struct {
	SchemaVersion  int            `json:"schemaVersion"`
	Surface        string         `json:"surface"`
	OrganizationID string         `json:"organizationId,omitempty"`
	SessionID      string         `json:"sessionId,omitempty"`
	RunID          string         `json:"runId,omitempty"`
	ThreadID       string         `json:"threadId,omitempty"`
	ModelID        string         `json:"modelId"`
	PromptMode     string         `json:"promptMode,omitempty"`
	TTLSeconds     int            `json:"ttlSeconds,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type revokePayload struct {
	Surface        string         `json:"surface"`
	OrganizationID string         `json:"organizationId,omitempty"`
	LeaseID        string         `json:"leaseId"`
	Reason         string         `json:"reason,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type cachedLease struct {
	lease     Lease
	expiresAt time.Time
}

var cache = struct {
	sync.Mutex
	byKey map[string]cachedLease
}{
	byKey: map[string]cachedLease{},
}

func Issue(session *f4rgesession.ManagedSession, request IssueRequest) (*Lease, error) {
	if !f4rgesession.IsRuntimeSessionUsable(session) {
		return nil, fmt.Errorf("F4RGE runtime session required")
	}
	if request.ModelID == "" {
		return nil, fmt.Errorf("F4RGE model is required")
	}
	ttl := request.TTLSeconds
	if ttl <= 0 {
		ttl = defaultTTLSeconds
	}
	key := cacheKey(session, request)
	if lease := cached(key); lease != nil {
		return lease, nil
	}
	var response issueResponse
	err := controlplane.New().PostJSON(session, "/api/runtime/credentials/issue", issuePayload{
		SchemaVersion:  1,
		Surface:        "cli",
		OrganizationID: session.OrganizationID,
		SessionID:      session.RuntimeSessionID,
		RunID:          request.RunID,
		ThreadID:       request.ThreadID,
		ModelID:        request.ModelID,
		PromptMode:     promptMode(request.PromptMode),
		TTLSeconds:     ttl,
		Metadata: map[string]any{
			"catalogVersion": request.CatalogVersion(),
			"policyVersion":  request.RuntimePolicy,
		},
	}, &response)
	if err != nil {
		return nil, err
	}
	if response.Lease.Secret.Value == "" {
		return nil, fmt.Errorf("F4RGE runtime credential lease did not include a usable secret")
	}
	if request.Catalog != nil {
		if model, ok := request.Catalog.ModelByID(response.Lease.ResolvedModelID); ok {
			response.Lease.ResolvedModel = model
		}
	}
	store(key, response.Lease)
	return &response.Lease, nil
}

func Revoke(session *f4rgesession.ManagedSession, leaseID string, reason string) error {
	if leaseID == "" {
		return nil
	}
	if !f4rgesession.IsRuntimeSessionUsable(session) {
		return nil
	}
	var target any
	err := controlplane.New().PostJSON(session, "/api/runtime/credentials/revoke", revokePayload{
		Surface:        "cli",
		OrganizationID: session.OrganizationID,
		LeaseID:        leaseID,
		Reason:         reason,
	}, &target)
	return err
}

func ClearMemoryCache(session *f4rgesession.ManagedSession, reason string) {
	cache.Lock()
	leases := make([]Lease, 0, len(cache.byKey))
	for key, cached := range cache.byKey {
		leases = append(leases, cached.lease)
		delete(cache.byKey, key)
	}
	cache.Unlock()
	for _, lease := range leases {
		_ = Revoke(session, lease.LeaseID, reason)
	}
}

func (request IssueRequest) CatalogVersion() string {
	if request.Catalog == nil {
		return ""
	}
	return request.Catalog.CatalogVersion
}

func cacheKey(session *f4rgesession.ManagedSession, request IssueRequest) string {
	return session.RuntimeSessionID + ":" + request.ModelID + ":" + promptMode(request.PromptMode)
}

func cached(key string) *Lease {
	cache.Lock()
	defer cache.Unlock()
	entry, ok := cache.byKey[key]
	if !ok {
		return nil
	}
	if time.Until(entry.expiresAt) <= 30*time.Second {
		delete(cache.byKey, key)
		return nil
	}
	lease := entry.lease
	return &lease
}

func store(key string, lease Lease) {
	expiresAt, err := time.Parse(time.RFC3339, lease.ExpiresAt)
	if err != nil {
		expiresAt = time.Now().Add(defaultTTLSeconds * time.Second)
	}
	cache.Lock()
	cache.byKey[key] = cachedLease{lease: lease, expiresAt: expiresAt}
	cache.Unlock()
}

func promptMode(value string) string {
	switch value {
	case "ask", "plan", "agent":
		return value
	default:
		return "agent"
	}
}
