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
	template := string(coderPromptTmpl)
	if session, err := f4rgesession.Load(); err == nil {
		if bundle, fetchErr := promptbundle.Fetch(session); fetchErr == nil && bundle != nil {
			if managed := strings.TrimSpace(bundle.RenderMode("agent")); managed != "" {
				template = managed + "\n\n" + template
			}
		}
	}
	systemPrompt, err := prompt.NewPrompt("coder", template, opts...)
	if err != nil {
		return nil, err
	}
	return systemPrompt, nil
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
