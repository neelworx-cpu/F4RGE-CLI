package runtimebundle

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
	"github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/modelcatalog"
	f4rgesession "github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/session"
)

const cacheFileName = "f4rge-runtime-bundle.json"

type Bundle struct {
	SchemaVersion  int                  `json:"schemaVersion"`
	GeneratedAt    string               `json:"generatedAt"`
	ExpiresAt      string               `json:"expiresAt,omitempty"`
	Source         string               `json:"source"`
	Surface        string               `json:"surface"`
	OrganizationID string               `json:"organizationId"`
	Session        *Session             `json:"session,omitempty"`
	RuntimeConfig  RuntimeConfig        `json:"runtimeConfig"`
	ModelCatalog   BundleRef            `json:"modelCatalog"`
	PromptBundle   BundleRef            `json:"promptBundle"`
	Policy         Policy               `json:"policy"`
	Defaults       Defaults             `json:"defaults"`
	Capabilities   []string             `json:"capabilities"`
	Entitlements   []string             `json:"entitlements"`
	Models         []modelcatalog.Model `json:"models,omitempty"`
	CachedAt       time.Time            `json:"cachedAt,omitempty"`
}

type RuntimeConfig struct {
	Version   string `json:"version"`
	UpdatedAt string `json:"updatedAt"`
	Reason    string `json:"reason,omitempty"`
}

type Session struct {
	SessionID      string   `json:"sessionId"`
	Surface        string   `json:"surface"`
	Kind           string   `json:"kind"`
	OrganizationID string   `json:"organizationId"`
	SubjectUserID  string   `json:"subjectUserId,omitempty"`
	OrganizationName string `json:"organizationName,omitempty"`
	DisplayName    string   `json:"displayName,omitempty"`
	Email          string   `json:"email,omitempty"`
	Scopes         []string `json:"scopes"`
	Status         string   `json:"status"`
	ExpiresAt      string   `json:"expiresAt"`
	IssuedAt       string   `json:"issuedAt"`
}

type BundleRef struct {
	SnapshotID     string `json:"snapshotId"`
	ActivationID   string `json:"activationId,omitempty"`
	CanonicalHash  string `json:"canonicalHash"`
	SignatureKeyID string `json:"signatureKeyId,omitempty"`
}

type Policy struct {
	Version               string              `json:"version,omitempty"`
	Enforcement           string              `json:"enforcement,omitempty"`
	MaxModeAllowed        bool                `json:"maxModeAllowed"`
	MaxModeEnabled        bool                `json:"maxModeEnabled"`
	ContextCapByModel     map[string]int      `json:"contextCapByModel,omitempty"`
	BlockedReasonsByModel map[string][]string `json:"blockedReasonsByModel,omitempty"`
}

type Defaults struct {
	Mode           string   `json:"mode"`
	ModelID        string   `json:"modelId,omitempty"`
	Agent          string   `json:"agent,omitempty"`
	Ask            string   `json:"ask,omitempty"`
	Plan           string   `json:"plan,omitempty"`
	Review         string   `json:"review,omitempty"`
	Title          string   `json:"title,omitempty"`
	Summarize      string   `json:"summarize,omitempty"`
	SubAgent       string   `json:"subAgent,omitempty"`
	FallbackChain  []string `json:"fallbackChain"`
	MaxModeEnabled bool     `json:"maxModeEnabled"`
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
		return nil, fmt.Errorf("read F4RGE runtime bundle cache: %w", err)
	}
	var bundle Bundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return nil, fmt.Errorf("parse F4RGE runtime bundle cache: %w", err)
	}
	return &bundle, nil
}

func AllowDevFallback() bool {
	return os.Getenv("F4RGE_CLI_ALLOW_DEV_FALLBACK") == "1" || os.Getenv("F4RGE_ALLOW_EMBEDDED_PROMPT_FALLBACK") == "1"
}

func Save(bundle Bundle) error {
	bundle.CachedAt = time.Now()
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return fmt.Errorf("encode F4RGE runtime bundle cache: %w", err)
	}
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create F4RGE runtime bundle cache directory: %w", err)
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func Fetch(session *f4rgesession.ManagedSession) (*Bundle, error) {
	if !f4rgesession.IsRuntimeSessionUsable(session) {
		if AllowDevFallback() {
			return LoadCached()
		}
		return nil, fmt.Errorf("F4RGE sign-in is incomplete: missing %s", strings.Join(f4rgesession.MissingRuntimeSessionFields(session), ", "))
	}
	var bundle Bundle
	if err := controlplane.New().GetJSON(session, "/api/runtime/bundles/effective", map[string]string{
		"surface":        "cli",
		"organizationId": session.OrganizationID,
	}, &bundle); err != nil {
		return nil, fmt.Errorf("fetch runtime bundle: %w", err)
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
		return fmt.Errorf("runtime bundle is missing")
	}
	if bundle.OrganizationID != session.OrganizationID {
		return fmt.Errorf("runtime bundle organization mismatch")
	}
	if bundle.ModelCatalog.CanonicalHash == "" || bundle.ModelCatalog.SnapshotID == "" {
		return fmt.Errorf("runtime bundle missing model catalog reference")
	}
	if bundle.PromptBundle.CanonicalHash == "" || bundle.PromptBundle.SnapshotID == "" {
		return fmt.Errorf("runtime bundle missing prompt bundle reference")
	}
	if bundle.RuntimeConfig.Version == "" {
		return fmt.Errorf("runtime bundle missing runtime config version")
	}
	if bundle.Policy.Enforcement == "" {
		return fmt.Errorf("runtime bundle missing policy enforcement")
	}
	if bundle.Defaults.ModelID == "" && bundle.Defaults.Agent == "" {
		return fmt.Errorf("runtime bundle missing model defaults")
	}
	return nil
}

func Ready() bool {
	session, err := f4rgesession.Load()
	if err != nil || !f4rgesession.IsUsable(session) {
		return false
	}
	if !f4rgesession.HasRuntimeScope(session, "runtime.credentials.issue") {
		return false
	}
	bundle, err := LoadCached()
	return err == nil && Validate(bundle, session) == nil
}
