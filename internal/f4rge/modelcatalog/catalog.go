package modelcatalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/neelworx-cpu/F4RGE-CLI/internal/config"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/controlplane"
	f4rgesession "github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/session"
)

const cacheFileName = "f4rge-model-catalog.json"

type Bundle struct {
	SchemaVersion  int       `json:"schemaVersion"`
	CatalogVersion string    `json:"catalogVersion"`
	CanonicalHash  string    `json:"canonicalHash"`
	GeneratedAt    string    `json:"generatedAt"`
	OrganizationID string    `json:"organizationId"`
	SubjectUserID  string    `json:"subjectUserId"`
	Surface        string    `json:"surface"`
	Models         []Model   `json:"models"`
	Defaults       Defaults  `json:"defaults"`
	CachedAt       time.Time `json:"cachedAt,omitempty"`
}

type Defaults struct {
	Ask           string   `json:"ask,omitempty"`
	Plan          string   `json:"plan,omitempty"`
	Agent         string   `json:"agent,omitempty"`
	Review        string   `json:"review,omitempty"`
	Title         string   `json:"title,omitempty"`
	Summarize     string   `json:"summarize,omitempty"`
	SubAgent      string   `json:"subAgent,omitempty"`
	FallbackChain []string `json:"fallbackChain"`
}

type Model struct {
	ID              string         `json:"id"`
	Provider        string         `json:"provider"`
	ProviderModelID string         `json:"providerModelId"`
	Label           string         `json:"label"`
	Description     string         `json:"description"`
	Status          string         `json:"status"`
	Capabilities    []string       `json:"capabilities"`
	RuntimeRoles    []string       `json:"runtimeRoles"`
	ContextWindow   int            `json:"contextWindow"`
	MaxOutputTokens int            `json:"maxOutputTokens"`
	RequestProfile  RequestProfile `json:"requestProfile"`
	CostClass       string         `json:"costClass"`
	RiskClass       string         `json:"riskClass"`
}

type RequestProfile struct {
	APIFamily       string   `json:"apiFamily"`
	ProviderModelID string   `json:"providerModelId"`
	DeploymentName  string   `json:"deploymentName,omitempty"`
	AcceptedParams  []string `json:"acceptedParameters"`
}

func Path() string {
	return filepath.Join(filepath.Dir(config.GlobalConfigData()), cacheFileName)
}

func LoadCached() (*Bundle, error) {
	data, err := os.ReadFile(Path())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read F4RGE model catalog cache: %w", err)
	}
	var bundle Bundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return nil, fmt.Errorf("parse F4RGE model catalog cache: %w", err)
	}
	return &bundle, nil
}

func Save(bundle Bundle) error {
	bundle.CachedAt = time.Now()
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return fmt.Errorf("encode F4RGE model catalog cache: %w", err)
	}
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create F4RGE model catalog cache directory: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write F4RGE model catalog cache: %w", err)
	}
	return nil
}

func Fetch(session *f4rgesession.ManagedSession) (*Bundle, error) {
	if !f4rgesession.IsRuntimeSessionUsable(session) {
		return nil, fmt.Errorf("F4RGE sign-in is incomplete: missing %s", strings.Join(f4rgesession.MissingRuntimeSessionFields(session), ", "))
	}
	var bundle Bundle
	if err := controlplane.New().GetJSON(session, "/api/runtime/model-catalog/effective", map[string]string{
		"surface":        "cli",
		"organizationId": session.OrganizationID,
	}, &bundle); err != nil {
		return nil, fmt.Errorf("fetch model catalog: %w", err)
	}
	if err := Save(bundle); err != nil {
		return nil, err
	}
	return &bundle, nil
}

func Validate(bundle *Bundle, session *f4rgesession.ManagedSession) error {
	if !f4rgesession.IsUsable(session) {
		return fmt.Errorf("F4RGE sign-in is incomplete: missing %s", strings.Join(f4rgesession.MissingReadinessFields(session), ", "))
	}
	if bundle == nil {
		return fmt.Errorf("model catalog is missing")
	}
	if bundle.OrganizationID != session.OrganizationID {
		return fmt.Errorf("model catalog organization mismatch")
	}
	if bundle.CatalogVersion == "" || bundle.CanonicalHash == "" {
		return fmt.Errorf("model catalog version is missing")
	}
	if len(bundle.Models) == 0 {
		return fmt.Errorf("model catalog has no models")
	}
	if bundle.Defaults.Agent == "" {
		return fmt.Errorf("model catalog missing agent default")
	}
	return nil
}
