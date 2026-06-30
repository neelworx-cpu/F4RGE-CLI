package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/config"
)

const (
	defaultAuthURL  = "https://auth.4rged.ai/cli"
	sessionFileName = "f4rge-session.json"
)

// ManagedSession is the local F4RGE client session snapshot. It is deliberately
// separated from provider config so customer auth does not look like BYOK setup.
type ManagedSession struct {
	AccessToken         string   `json:"access_token,omitempty"`
	RefreshToken        string   `json:"refresh_token,omitempty"`
	RuntimeSessionID    string   `json:"runtime_session_id,omitempty"`
	RuntimeScopes       []string `json:"runtime_scopes,omitempty"`
	SubjectUserID       string   `json:"subject_user_id,omitempty"`
	UserDisplayName     string   `json:"user_display_name,omitempty"`
	UserEmail           string   `json:"user_email,omitempty"`
	OrganizationID      string   `json:"organization_id,omitempty"`
	OrganizationName    string   `json:"organization_name,omitempty"`
	PolicyVersion       string   `json:"policy_version,omitempty"`
	ModelCatalogVersion string   `json:"model_catalog_version,omitempty"`
	PromptBundleVersion string   `json:"prompt_bundle_version,omitempty"`
	RuntimeBundleHash   string   `json:"runtime_bundle_hash,omitempty"`
	GatewayEndpoint     string   `json:"gateway_endpoint,omitempty"`
	PlatformEndpoint    string   `json:"platform_endpoint,omitempty"`
	ExpiresAt           int64    `json:"expires_at,omitempty"`
	CreatedAt           int64    `json:"created_at,omitempty"`
	UpdatedAt           int64    `json:"updated_at,omitempty"`
}

// DeviceAuth describes the customer-facing device/browser sign-in step.
type DeviceAuth struct {
	DeviceCode      string
	UserCode        string
	VerificationURL string
	ExpiresIn       int
	Interval        int
}

func AuthURL() string {
	if value := os.Getenv("F4RGE_AUTH_URL"); value != "" {
		return value
	}
	return defaultAuthURL
}

func Path() string {
	return filepath.Join(filepath.Dir(config.GlobalConfigData()), sessionFileName)
}

func Load() (*ManagedSession, error) {
	data, err := os.ReadFile(Path())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read F4RGE session: %w", err)
	}
	var session ManagedSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("parse F4RGE session: %w", err)
	}
	return &session, nil
}

func Save(session ManagedSession) error {
	now := time.Now().Unix()
	if session.CreatedAt == 0 {
		session.CreatedAt = now
	}
	session.UpdatedAt = now
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("encode F4RGE session: %w", err)
	}
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create F4RGE session directory: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write F4RGE session: %w", err)
	}
	return nil
}

func Clear() error {
	if err := os.Remove(Path()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove F4RGE session: %w", err)
	}
	return nil
}

func IsUsable(session *ManagedSession) bool {
	return IsRuntimeSessionUsable(session) &&
		session.SubjectUserID != "" &&
		session.UserEmail != "" &&
		session.OrganizationName != "" &&
		session.ModelCatalogVersion != "" &&
		session.PromptBundleVersion != "" &&
		session.RuntimeBundleHash != ""
}

func IsRuntimeSessionUsable(session *ManagedSession) bool {
	return session != nil &&
		session.AccessToken != "" &&
		session.RuntimeSessionID != "" &&
		session.OrganizationID != "" &&
		session.OrganizationID != "pending" &&
		!IsExpired(session)
}

func HasRuntimeScope(session *ManagedSession, scope string) bool {
	return session != nil && slices.Contains(session.RuntimeScopes, scope)
}

func MissingReadinessFields(session *ManagedSession) []string {
	if session == nil {
		return []string{"session"}
	}
	var missing []string
	if session.AccessToken == "" {
		missing = append(missing, "access token")
	}
	if session.RuntimeSessionID == "" {
		missing = append(missing, "runtime session")
	}
	if session.SubjectUserID == "" {
		missing = append(missing, "user")
	}
	if session.UserEmail == "" {
		missing = append(missing, "email")
	}
	if session.OrganizationID == "" || session.OrganizationID == "pending" {
		missing = append(missing, "organization")
	}
	if session.OrganizationName == "" {
		missing = append(missing, "organization name")
	}
	if session.ModelCatalogVersion == "" {
		missing = append(missing, "model catalog")
	}
	if session.PromptBundleVersion == "" {
		missing = append(missing, "prompt bundle")
	}
	if session.RuntimeBundleHash == "" {
		missing = append(missing, "runtime bundle")
	}
	if IsExpired(session) {
		missing = append(missing, "valid token")
	}
	return missing
}

func MissingRuntimeSessionFields(session *ManagedSession) []string {
	if session == nil {
		return []string{"session"}
	}
	var missing []string
	if session.AccessToken == "" {
		missing = append(missing, "access token")
	}
	if session.RuntimeSessionID == "" {
		missing = append(missing, "runtime session")
	}
	if session.OrganizationID == "" || session.OrganizationID == "pending" {
		missing = append(missing, "organization")
	}
	if IsExpired(session) {
		missing = append(missing, "valid token")
	}
	return missing
}

func StartDeviceAuth() DeviceAuth {
	return DeviceAuth{
		DeviceCode:      uuid.NewString(),
		UserCode:        "F4RGED-CLI",
		VerificationURL: AuthURL(),
		ExpiresIn:       900,
		Interval:        5,
	}
}

func IsExpired(session *ManagedSession) bool {
	return session != nil && session.ExpiresAt > 0 && time.Now().Unix() >= session.ExpiresAt
}
