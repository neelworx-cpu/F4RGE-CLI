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
	SchemaVersion  int        `json:"schemaVersion"`
	CatalogVersion string     `json:"catalogVersion"`
	CanonicalHash  string     `json:"canonicalHash"`
	GeneratedAt    string     `json:"generatedAt"`
	OrganizationID string     `json:"organizationId"`
	SubjectUserID  string     `json:"subjectUserId"`
	Surface        string     `json:"surface"`
	Providers      []Provider `json:"providers"`
	Models         []Model    `json:"models"`
	Defaults       Defaults   `json:"defaults"`
	CachedAt       time.Time  `json:"cachedAt,omitempty"`
}

type Provider struct {
	ID             string   `json:"id"`
	Label          string   `json:"label"`
	Status         string   `json:"status"`
	CredentialMode string   `json:"credentialMode"`
	Surfaces       []string `json:"surfaces"`
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
	ID                   string           `json:"id"`
	Provider             string           `json:"provider"`
	ProviderModelID      string           `json:"providerModelId"`
	Label                string           `json:"label"`
	Description          string           `json:"description"`
	Status               string           `json:"status"`
	Capabilities         []string         `json:"capabilities"`
	RuntimeRoles         []string         `json:"runtimeRoles"`
	ContextWindow        int              `json:"contextWindow"`
	DefaultContextWindow int              `json:"defaultContextWindow,omitempty"`
	MaxOutputTokens      int              `json:"maxOutputTokens"`
	RequestProfile       RequestProfile   `json:"requestProfile"`
	ParameterOptions     ParameterOptions `json:"parameterOptions"`
	CostClass            string           `json:"costClass"`
	RiskClass            string           `json:"riskClass"`
}

type RequestProfile struct {
	APIFamily         string         `json:"apiFamily"`
	ProviderModelID   string         `json:"providerModelId"`
	DeploymentName    string         `json:"deploymentName,omitempty"`
	AcceptedParams    []string       `json:"acceptedParameters"`
	DefaultParameters map[string]any `json:"defaultParameters,omitempty"`
}

// ParameterOptions is the catalog-delivered adjustable parameter surface. The
// CLI never lets users adjust these; it consumes the default-flagged option of
// each group. Mirrors F4rgeModelParameterOptions in F4RGE-Web platform-contracts.
type ParameterOptions struct {
	ContextWindows   []ContextWindowOption  `json:"contextWindows,omitempty"`
	ReasoningEfforts []ReasoningEffortOption `json:"reasoningEfforts,omitempty"`
	Toggles          []ToggleOption          `json:"toggles,omitempty"`
}

type ContextWindowOption struct {
	Value   int    `json:"value"`
	Label   string `json:"label"`
	Default bool   `json:"default,omitempty"`
}

type ReasoningEffortOption struct {
	Value   string `json:"value"`
	Label   string `json:"label"`
	Default bool   `json:"default,omitempty"`
}

type ToggleOption struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Value   bool   `json:"value"`
	Default bool   `json:"default,omitempty"`
}

// ResolvedDefaults is the effective default parameter selection for a model,
// derived purely from the catalog ParameterOptions.
type ResolvedDefaults struct {
	ReasoningEffort string
	ContextWindow   int
	Toggles         map[string]bool
}

// ResolveDefaults mirrors resolveModelParameterDefaults in platform-contracts:
// it returns the default-flagged option per group (falling back to the first),
// so CLI/Cloud apply identical defaults to what the catalog/console define.
func (m Model) ResolveDefaults() ResolvedDefaults {
	resolved := ResolvedDefaults{Toggles: map[string]bool{}}

	if len(m.ParameterOptions.ReasoningEfforts) > 0 {
		chosen := m.ParameterOptions.ReasoningEfforts[0]
		for _, option := range m.ParameterOptions.ReasoningEfforts {
			if option.Default {
				chosen = option
				break
			}
		}
		resolved.ReasoningEffort = chosen.Value
	}

	if len(m.ParameterOptions.ContextWindows) > 0 {
		chosen := m.ParameterOptions.ContextWindows[0]
		for _, option := range m.ParameterOptions.ContextWindows {
			if option.Default {
				chosen = option
				break
			}
		}
		resolved.ContextWindow = chosen.Value
	} else if m.DefaultContextWindow > 0 {
		resolved.ContextWindow = m.DefaultContextWindow
	}

	for _, toggle := range m.ParameterOptions.Toggles {
		resolved.Toggles[toggle.ID] = toggle.Default || toggle.Value
	}

	return resolved
}

// ThinkingEnabledByDefault reports whether the catalog defaults turn thinking on.
func (m Model) ThinkingEnabledByDefault() bool {
	return m.ResolveDefaults().Toggles["thinking"]
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

func (b *Bundle) ModelByID(modelID string) (Model, bool) {
	if b == nil {
		return Model{}, false
	}
	for _, model := range b.Models {
		if model.ID == modelID {
			return model, true
		}
	}
	return Model{}, false
}

func (b *Bundle) ProviderByID(providerID string) (Provider, bool) {
	if b == nil {
		return Provider{}, false
	}
	for _, provider := range b.Providers {
		if provider.ID == providerID {
			return provider, true
		}
	}
	return Provider{}, false
}

func (b *Bundle) ProviderLabel(providerID string) string {
	if provider, ok := b.ProviderByID(providerID); ok && provider.Label != "" {
		return provider.Label
	}
	switch providerID {
	case "openai":
		return "OpenAI"
	case "anthropic":
		return "Anthropic"
	case "google":
		return "Google"
	case "azure":
		return "Azure OpenAI"
	case "f4rge":
		return "4RGE-AI"
	default:
		return providerID
	}
}

func (b *Bundle) GroupedModels() []ProviderGroup {
	if b == nil {
		return nil
	}
	groupByProvider := map[string][]Model{}
	order := make([]string, 0, len(b.Providers))
	seen := map[string]bool{}
	for _, provider := range b.Providers {
		if provider.Status == "disabled" {
			continue
		}
		order = append(order, provider.ID)
		seen[provider.ID] = true
	}
	for _, model := range b.Models {
		if !seen[model.Provider] {
			order = append(order, model.Provider)
			seen[model.Provider] = true
		}
		groupByProvider[model.Provider] = append(groupByProvider[model.Provider], model)
	}
	groups := make([]ProviderGroup, 0, len(order))
	for _, providerID := range order {
		models := groupByProvider[providerID]
		if len(models) == 0 {
			continue
		}
		groups = append(groups, ProviderGroup{
			ProviderID: providerID,
			Label:      b.ProviderLabel(providerID),
			Models:     models,
		})
	}
	return groups
}

type ProviderGroup struct {
	ProviderID string
	Label      string
	Models     []Model
}
