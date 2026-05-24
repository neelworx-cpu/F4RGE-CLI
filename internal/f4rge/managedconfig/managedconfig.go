package managedconfig

import (
	"charm.land/catwalk/pkg/catwalk"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/config"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/modelcatalog"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/runtimebundle"
	f4rgesession "github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/session"
)

const ProviderID = "f4rge-gateway"

func Apply(store *config.ConfigStore) bool {
	session, err := f4rgesession.Load()
	if err != nil || !f4rgesession.IsUsable(session) {
		return false
	}
	runtime, _ := runtimebundle.Fetch(session)
	bundle, err := modelcatalog.Fetch(session)
	if err != nil || bundle == nil || len(bundle.Models) == 0 {
		return false
	}
	models := make([]catwalk.Model, 0, len(bundle.Models))
	for _, model := range bundle.Models {
		models = append(models, catwalk.Model{
			ID:                     model.ID,
			Name:                   model.Label,
			ContextWindow:          int64(model.ContextWindow),
			CanReason:              true,
			DefaultMaxTokens:       defaultMaxTokens(model),
			DefaultReasoningEffort: defaultReasoningEffort(model),
		})
	}
	cfg := store.Config()
	cfg.Providers.Set(ProviderID, config.ProviderConfig{
		ID:      ProviderID,
		Name:    "F4RGE Gateway",
		Type:    ProviderID,
		Models:  models,
		Disable: false,
	})
	if cfg.Models == nil {
		cfg.Models = map[config.SelectedModelType]config.SelectedModel{}
	}
	largeModel := roleDefault(bundle, runtime, "agent")
	smallModel := roleDefault(bundle, runtime, "subAgent")
	cfg.Models[config.SelectedModelTypeLarge] = selectedModel(largeModel, models)
	cfg.Models[config.SelectedModelTypeSmall] = selectedModel(smallModel, models)
	cfg.SetupAgents()
	return true
}

func EnsureSelectedModels(cfg *config.Config) bool {
	provider, ok := cfg.Providers.Get(ProviderID)
	if !ok || len(provider.Models) == 0 {
		return false
	}
	if cfg.Models == nil {
		cfg.Models = map[config.SelectedModelType]config.SelectedModel{}
	}

	largeModel := provider.Models[0]
	if current, ok := cfg.Models[config.SelectedModelTypeLarge]; ok {
		if model := cfg.GetModel(ProviderID, current.Model); model != nil {
			largeModel = *model
		}
	}
	smallModel := largeModel
	if current, ok := cfg.Models[config.SelectedModelTypeSmall]; ok {
		if model := cfg.GetModel(ProviderID, current.Model); model != nil {
			smallModel = *model
		}
	}
	cfg.Models[config.SelectedModelTypeLarge] = config.SelectedModel{
		Provider:        ProviderID,
		Model:           largeModel.ID,
		MaxTokens:       largeModel.DefaultMaxTokens,
		ReasoningEffort: largeModel.DefaultReasoningEffort,
	}
	cfg.Models[config.SelectedModelTypeSmall] = config.SelectedModel{
		Provider:        ProviderID,
		Model:           smallModel.ID,
		MaxTokens:       smallModel.DefaultMaxTokens,
		ReasoningEffort: smallModel.DefaultReasoningEffort,
	}
	return true
}

func roleDefault(bundle *modelcatalog.Bundle, runtime *runtimebundle.Bundle, role string) string {
	if runtime != nil {
		switch role {
		case "agent":
			if runtime.Defaults.Agent != "" {
				return runtime.Defaults.Agent
			}
			if runtime.Defaults.ModelID != "" {
				return runtime.Defaults.ModelID
			}
		case "subAgent":
			if runtime.Defaults.SubAgent != "" {
				return runtime.Defaults.SubAgent
			}
			if runtime.Defaults.Ask != "" {
				return runtime.Defaults.Ask
			}
		}
	}
	switch role {
	case "agent":
		if bundle.Defaults.Agent != "" {
			return bundle.Defaults.Agent
		}
	case "subAgent":
		if bundle.Defaults.SubAgent != "" {
			return bundle.Defaults.SubAgent
		}
		if bundle.Defaults.Ask != "" {
			return bundle.Defaults.Ask
		}
	}
	for _, model := range bundle.Models {
		if hasRuntimeRole(model, role) {
			return model.ID
		}
	}
	if len(bundle.Models) > 0 {
		return bundle.Models[0].ID
	}
	return ""
}

func hasRuntimeRole(model modelcatalog.Model, role string) bool {
	if len(model.RuntimeRoles) == 0 {
		return true
	}
	for _, candidate := range model.RuntimeRoles {
		if candidate == role {
			return true
		}
	}
	return false
}

func defaultMaxTokens(model modelcatalog.Model) int64 {
	if model.MaxOutputTokens > 0 {
		return int64(model.MaxOutputTokens)
	}
	return 4096
}

func defaultReasoningEffort(model modelcatalog.Model) string {
	for _, role := range model.RuntimeRoles {
		if role == "title" || role == "summarize" || role == "subAgent" {
			return "low"
		}
	}
	return "medium"
}

func selectedModel(modelID string, models []catwalk.Model) config.SelectedModel {
	selected := config.SelectedModel{Provider: ProviderID, Model: modelID}
	for _, model := range models {
		if model.ID == modelID {
			selected.MaxTokens = model.DefaultMaxTokens
			selected.ReasoningEffort = model.DefaultReasoningEffort
			return selected
		}
	}
	return selected
}
