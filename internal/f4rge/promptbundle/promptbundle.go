package promptbundle

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
	"github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/runtimebundle"
	f4rgesession "github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/session"
)

const cacheFileName = "f4rge-prompt-bundle.json"

type Bundle struct {
	Snapshot      Snapshot  `json:"snapshot"`
	CanonicalHash string    `json:"canonicalHash"`
	Source        string    `json:"source"`
	CachedAt      time.Time `json:"cachedAt,omitempty"`
}

type Snapshot struct {
	SnapshotID     string      `json:"snapshotId"`
	VersionName    string      `json:"versionName"`
	RolloutChannel string      `json:"rolloutChannel"`
	RuntimeSurface string      `json:"surface"`
	Modules        []Module    `json:"modules"`
	ModeStacks     []ModeStack `json:"modeStacks"`
}

type Module struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Content string `json:"content"`
}

type ModeStack struct {
	Mode       string      `json:"mode"`
	ModuleRefs []ModuleRef `json:"moduleRefs"`
}

type ModuleRef struct {
	ID       string `json:"id"`
	Required bool   `json:"required"`
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
		return nil, fmt.Errorf("read F4RGE prompt bundle cache: %w", err)
	}
	var bundle Bundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return nil, fmt.Errorf("parse F4RGE prompt bundle cache: %w", err)
	}
	return &bundle, nil
}

func Save(bundle Bundle) error {
	bundle.CachedAt = time.Now()
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return fmt.Errorf("encode F4RGE prompt bundle cache: %w", err)
	}
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create F4RGE prompt bundle cache directory: %w", err)
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func Fetch(session *f4rgesession.ManagedSession) (*Bundle, error) {
	if !f4rgesession.IsRuntimeSessionUsable(session) {
		return nil, fmt.Errorf("F4RGE sign-in is incomplete: missing %s", strings.Join(f4rgesession.MissingRuntimeSessionFields(session), ", "))
	}
	var bundle Bundle
	if err := controlplane.New().GetJSON(session, "/api/runtime/prompt-bundles/effective", map[string]string{
		"surface":        "cli",
		"organizationId": session.OrganizationID,
	}, &bundle); err != nil {
		return nil, fmt.Errorf("fetch prompt bundle: %w", err)
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
		return fmt.Errorf("prompt bundle is missing")
	}
	if bundle.CanonicalHash == "" || bundle.Snapshot.SnapshotID == "" {
		return fmt.Errorf("prompt bundle version is missing")
	}
	if bundle.Snapshot.RuntimeSurface != "" && bundle.Snapshot.RuntimeSurface != "cli" {
		return fmt.Errorf("prompt bundle is for %s, not cli", bundle.Snapshot.RuntimeSurface)
	}
	if bundle.Snapshot.RuntimeSurface == "" {
		return fmt.Errorf("prompt bundle missing runtime surface")
	}
	if bundle.Source == "bundled-fallback" && !runtimebundle.AllowDevFallback() {
		return fmt.Errorf("CLI prompt bundle is not published")
	}
	return nil
}

func (b Bundle) RenderMode(mode string) string {
	stack := ModeStack{}
	for _, candidate := range b.Snapshot.ModeStacks {
		if candidate.Mode == mode {
			stack = candidate
			break
		}
	}
	if len(stack.ModuleRefs) == 0 {
		for _, candidate := range b.Snapshot.ModeStacks {
			if candidate.Mode == "agent" {
				stack = candidate
				break
			}
		}
	}
	moduleByID := make(map[string]string, len(b.Snapshot.Modules))
	for _, module := range b.Snapshot.Modules {
		moduleByID[module.ID] = module.Content
	}
	var sections []string
	for _, ref := range stack.ModuleRefs {
		if content := strings.TrimSpace(moduleByID[ref.ID]); content != "" {
			sections = append(sections, content)
		}
	}
	return strings.Join(sections, "\n\n")
}
