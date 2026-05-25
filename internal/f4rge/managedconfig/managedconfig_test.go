package managedconfig

import (
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/config"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/csync"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/modelcatalog"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/runtimebundle"
	"github.com/stretchr/testify/require"
)

func TestEnsureSelectedModelsBackfillsMissingSmallModel(t *testing.T) {
	t.Parallel()

	cfg := newManagedConfig(map[config.SelectedModelType]config.SelectedModel{
		config.SelectedModelTypeLarge: {
			Provider: ProviderID,
			Model:    "f4rge/4rge-2.5",
		},
	})

	require.True(t, EnsureSelectedModels(cfg))
	require.Equal(t, "f4rge/4rge-2.5", cfg.Models[config.SelectedModelTypeLarge].Model)
	require.Equal(t, "f4rge/4rge-2.5", cfg.Models[config.SelectedModelTypeSmall].Model)
}

func TestEnsureSelectedModelsPreservesValidSmallModel(t *testing.T) {
	t.Parallel()

	cfg := newManagedConfig(map[config.SelectedModelType]config.SelectedModel{
		config.SelectedModelTypeLarge: {
			Provider: ProviderID,
			Model:    "anthropic/claude-sonnet-4.5",
		},
		config.SelectedModelTypeSmall: {
			Provider: ProviderID,
			Model:    "openai/gpt-5.4-mini",
		},
	})

	require.True(t, EnsureSelectedModels(cfg))
	require.Equal(t, "anthropic/claude-sonnet-4.5", cfg.Models[config.SelectedModelTypeLarge].Model)
	require.Equal(t, "openai/gpt-5.4-mini", cfg.Models[config.SelectedModelTypeSmall].Model)
}

func TestEnsureSelectedModelsPreservesSelectedAgentModel(t *testing.T) {
	t.Parallel()

	cfg := newManagedConfig(map[config.SelectedModelType]config.SelectedModel{
		config.SelectedModelTypeLarge: {
			Provider: ProviderID,
			Model:    "anthropic/claude-sonnet-4.5",
		},
		config.SelectedModelTypeSmall: {
			Provider: ProviderID,
			Model:    "openai/gpt-5.4-mini",
		},
	})

	require.True(t, EnsureSelectedModels(cfg))
	require.Equal(t, "anthropic/claude-sonnet-4.5", cfg.Models[config.SelectedModelTypeLarge].Model)
	require.Equal(t, "openai/gpt-5.4-mini", cfg.Models[config.SelectedModelTypeSmall].Model)
}

func TestRoleDefaultUsesRuntimeAgentAndSubAgentDefaults(t *testing.T) {
	t.Parallel()

	bundle := &modelcatalog.Bundle{
		Models: []modelcatalog.Model{
			{ID: "large-model", RuntimeRoles: []string{"agent"}},
			{ID: "small-model", RuntimeRoles: []string{"subAgent"}},
		},
	}
	runtime := &runtimebundle.Bundle{
		Defaults: runtimebundle.Defaults{
			Agent:    "large-model",
			SubAgent: "small-model",
		},
	}

	require.Equal(t, "large-model", roleDefault(bundle, runtime, "agent"))
	require.Equal(t, "small-model", roleDefault(bundle, runtime, "subAgent"))
}

func TestApplyManagedSelectionsPreservesSelectedAgentModel(t *testing.T) {
	t.Parallel()

	cfg := newManagedConfig(map[config.SelectedModelType]config.SelectedModel{
		config.SelectedModelTypeLarge: {
			Provider: ProviderID,
			Model:    "anthropic/claude-sonnet-4.5",
		},
		config.SelectedModelTypeSmall: {
			Provider: ProviderID,
			Model:    "openai/gpt-5.4-mini",
		},
	})
	provider, ok := cfg.Providers.Get(ProviderID)
	require.True(t, ok)

	applyManagedSelections(cfg, &modelcatalog.Bundle{
		Models: []modelcatalog.Model{
			{ID: "f4rge/4rge-2.5", RuntimeRoles: []string{"agent"}},
			{ID: "openai/gpt-5.4-mini", RuntimeRoles: []string{"subAgent"}},
		},
		Defaults: modelcatalog.Defaults{
			Agent:    "f4rge/4rge-2.5",
			SubAgent: "openai/gpt-5.4-mini",
		},
	}, nil, provider.Models)

	require.Equal(t, "anthropic/claude-sonnet-4.5", cfg.Models[config.SelectedModelTypeLarge].Model)
	require.Equal(t, "openai/gpt-5.4-mini", cfg.Models[config.SelectedModelTypeSmall].Model)
}

func newManagedConfig(models map[config.SelectedModelType]config.SelectedModel) *config.Config {
	providers := csync.NewMap[string, config.ProviderConfig]()
	providers.Set(ProviderID, config.ProviderConfig{
		ID:   ProviderID,
		Name: "F4RGE Gateway",
		Type: ProviderID,
		Models: []catwalk.Model{
			{
				ID:                     "f4rge/4rge-2.5",
				Name:                   "4RGE 2.5",
				DefaultMaxTokens:       4096,
				DefaultReasoningEffort: "high",
			},
			{
				ID:                     "anthropic/claude-sonnet-4.5",
				Name:                   "Claude Sonnet 4.5",
				DefaultMaxTokens:       8192,
				DefaultReasoningEffort: "medium",
			},
			{
				ID:                     "openai/gpt-5.4-mini",
				Name:                   "GPT-5.4 Mini",
				DefaultMaxTokens:       4096,
				DefaultReasoningEffort: "medium",
			},
		},
	})
	return &config.Config{
		Providers: providers,
		Models:    models,
	}
}
