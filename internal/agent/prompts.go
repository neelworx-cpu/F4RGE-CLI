package agent

import (
	"context"
	_ "embed"
	"strings"

	"github.com/neelworx-cpu/F4RGE-CLI/internal/agent/prompt"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/config"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/promptbundle"
	f4rgesession "github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/session"
)

//go:embed templates/coder.md.tpl
var coderPromptTmpl []byte

//go:embed templates/task.md.tpl
var taskPromptTmpl []byte

//go:embed templates/initialize.md.tpl
var initializePromptTmpl []byte

func coderPrompt(opts ...prompt.Option) (*prompt.Prompt, error) {
	// The cloud catalog is the source of truth for the coder prompt: when a
	// managed bundle renders a non-empty agent stack it REPLACES the embedded
	// template (the seeded catalog content carries the same Go template
	// placeholders). The embedded template remains the offline fallback.
	template := string(coderPromptTmpl)
	if managed := managedCoderTemplate(); managed != "" {
		template = managed
	}
	systemPrompt, err := prompt.NewPrompt("coder", template, opts...)
	if err != nil {
		return nil, err
	}
	return systemPrompt, nil
}

// managedCoderTemplate resolves the managed prompt template for the agent
// mode: live fetch first, last-good cached bundle when the control plane is
// unreachable, empty string when neither is available or valid.
func managedCoderTemplate() string {
	session, err := f4rgesession.Load()
	if err != nil {
		return ""
	}
	bundle, fetchErr := promptbundle.Fetch(session)
	if fetchErr != nil || bundle == nil {
		cached, cacheErr := promptbundle.LoadCached()
		if cacheErr != nil || cached == nil {
			return ""
		}
		bundle = cached
	}
	if err := promptbundle.Validate(bundle, session); err != nil {
		return ""
	}
	return strings.TrimSpace(bundle.RenderMode("agent"))
}

func taskPrompt(opts ...prompt.Option) (*prompt.Prompt, error) {
	systemPrompt, err := prompt.NewPrompt("task", string(taskPromptTmpl), opts...)
	if err != nil {
		return nil, err
	}
	return systemPrompt, nil
}

func InitializePrompt(cfg *config.ConfigStore) (string, error) {
	systemPrompt, err := prompt.NewPrompt("initialize", string(initializePromptTmpl))
	if err != nil {
		return "", err
	}
	return systemPrompt.Build(context.Background(), "", "", cfg)
}
