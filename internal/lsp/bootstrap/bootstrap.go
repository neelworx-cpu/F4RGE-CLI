package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/neelworx-cpu/F4RGE-CLI/internal/config"
)

type StatusKind string

const (
	StatusAvailable   StatusKind = "available"
	StatusInstalling  StatusKind = "installing"
	StatusInstalled   StatusKind = "installed"
	StatusUnavailable StatusKind = "unavailable"
	StatusFailed      StatusKind = "failed"
)

type Status struct {
	Name       string
	Command    string
	Resolved   string
	Kind       StatusKind
	Message    string
	UpdatedAt  time.Time
	Managed    bool
	InstallCmd []string
}

type Installer interface {
	Install(ctx context.Context, spec Spec) error
}

type commandInstaller struct{}

type Manager struct {
	root      string
	binDir    string
	installer Installer
	mu        sync.Mutex
	statuses  map[string]Status
}

type Spec struct {
	Name       string
	Command    string
	BinaryName string
	Strategy   string
	Args       []string
	Packages   []string
	Env        map[string]string
}

var specs = map[string]Spec{
	"typescript-language-server": {
		Name:       "typescript",
		Command:    "typescript-language-server",
		BinaryName: executableName("typescript-language-server"),
		Strategy:   "npm",
		Args:       []string{"install", "--prefix", ".", "typescript", "typescript-language-server"},
		Packages:   []string{"typescript", "typescript-language-server"},
	},
	"vtsls": {
		Name:       "typescript",
		Command:    "vtsls",
		BinaryName: executableName("vtsls"),
		Strategy:   "npm",
		Args:       []string{"install", "--prefix", ".", "typescript", "@vtsls/language-server"},
		Packages:   []string{"typescript", "@vtsls/language-server"},
	},
	"pyright": {
		Name:       "python",
		Command:    "pyright-langserver",
		BinaryName: executableName("pyright-langserver"),
		Strategy:   "npm",
		Args:       []string{"install", "--prefix", ".", "pyright"},
		Packages:   []string{"pyright"},
	},
	"pyright-langserver": {
		Name:       "python",
		Command:    "pyright-langserver",
		BinaryName: executableName("pyright-langserver"),
		Strategy:   "npm",
		Args:       []string{"install", "--prefix", ".", "pyright"},
		Packages:   []string{"pyright"},
	},
	"gopls": {
		Name:       "go",
		Command:    "gopls",
		BinaryName: executableName("gopls"),
		Strategy:   "go",
		Args:       []string{"install", "golang.org/x/tools/gopls@latest"},
		Packages:   []string{"golang.org/x/tools/gopls@latest"},
	},
}

var defaultManager = NewManager(DefaultRoot(), commandInstaller{})

func DefaultRoot() string {
	return filepath.Join(config.GlobalCacheDir(), "lsp")
}

func NewManager(root string, installer Installer) *Manager {
	if installer == nil {
		installer = commandInstaller{}
	}
	return &Manager{
		root:      root,
		binDir:    filepath.Join(root, "bin"),
		installer: installer,
		statuses:  map[string]Status{},
	}
}

func Resolve(ctx context.Context, name string, command string) (string, bool, Status) {
	return defaultManager.Resolve(ctx, name, command)
}

func Statuses() []Status {
	return defaultManager.Statuses()
}

func KnownCommand(command string) bool {
	_, ok := specs[filepath.Base(command)]
	return ok
}

func (m *Manager) Resolve(ctx context.Context, name string, command string) (string, bool, Status) {
	if command == "" {
		status := Status{Name: name, Command: command, Kind: StatusUnavailable, Message: "missing LSP command", UpdatedAt: time.Now()}
		m.setStatus(name, status)
		return "", false, status
	}
	if path, err := exec.LookPath(command); err == nil {
		status := Status{Name: name, Command: command, Resolved: path, Kind: StatusAvailable, Message: "available on PATH", UpdatedAt: time.Now()}
		m.setStatus(name, status)
		return path, true, status
	}

	spec, ok := specs[filepath.Base(command)]
	if !ok {
		status := Status{Name: name, Command: command, Kind: StatusUnavailable, Message: "not available on PATH and not managed by F4RGE", UpdatedAt: time.Now()}
		m.setStatus(name, status)
		return "", false, status
	}

	managedPath := m.managedPath(spec)
	if isExecutable(managedPath) {
		status := Status{Name: name, Command: command, Resolved: managedPath, Kind: StatusInstalled, Message: "available from F4RGE managed tools", UpdatedAt: time.Now(), Managed: true}
		m.setStatus(name, status)
		return managedPath, true, status
	}

	status := Status{Name: name, Command: command, Resolved: managedPath, Kind: StatusInstalling, Message: "installing F4RGE managed language server", UpdatedAt: time.Now(), Managed: true, InstallCmd: spec.InstallCommand(m.root)}
	m.setStatus(name, status)
	if err := m.installer.Install(ctx, spec.withRoot(m.root, m.binDir)); err != nil {
		status.Kind = StatusFailed
		status.Message = err.Error()
		status.UpdatedAt = time.Now()
		m.setStatus(name, status)
		return "", false, status
	}
	if isExecutable(managedPath) {
		status.Kind = StatusInstalled
		status.Message = "installed F4RGE managed language server"
		status.UpdatedAt = time.Now()
		m.setStatus(name, status)
		return managedPath, true, status
	}
	status.Kind = StatusFailed
	status.Message = "managed install completed but language server binary was not found"
	status.UpdatedAt = time.Now()
	m.setStatus(name, status)
	return "", false, status
}

func (m *Manager) managedPath(spec Spec) string {
	switch spec.Strategy {
	case "npm":
		return filepath.Join(m.root, "node_modules", ".bin", spec.BinaryName)
	default:
		return filepath.Join(m.binDir, spec.BinaryName)
	}
}

func (m *Manager) Statuses() []Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Status, 0, len(m.statuses))
	for _, status := range m.statuses {
		out = append(out, status)
	}
	slices.SortFunc(out, func(a, b Status) int {
		return strings.Compare(a.Name, b.Name)
	})
	return out
}

func (m *Manager) setStatus(name string, status Status) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statuses[name] = status
}

func (spec Spec) withRoot(root string, binDir string) Spec {
	spec.Env = map[string]string{}
	switch spec.Strategy {
	case "npm":
		spec.Env["npm_config_prefix"] = root
	case "go":
		spec.Env["GOBIN"] = binDir
	}
	return spec
}

func (spec Spec) InstallCommand(root string) []string {
	switch spec.Strategy {
	case "npm":
		return append([]string{"npm"}, spec.Args...)
	case "go":
		return append([]string{"go"}, spec.Args...)
	default:
		return nil
	}
}

func (commandInstaller) Install(ctx context.Context, spec Spec) error {
	if err := os.MkdirAll(filepath.Join(specRoot(spec), "bin"), 0o700); err != nil {
		return err
	}
	var command string
	switch spec.Strategy {
	case "npm":
		command = "npm"
	case "go":
		command = "go"
	default:
		return fmt.Errorf("unsupported LSP install strategy %q", spec.Strategy)
	}
	if _, err := exec.LookPath(command); err != nil {
		if spec.Strategy == "go" {
			if goPath := discoverGoToolchain(); goPath != "" {
				command = goPath
			} else {
				return fmt.Errorf("go is required to install %s", spec.Command)
			}
		} else {
			return fmt.Errorf("%s is required to install %s", command, spec.Command)
		}
	}
	installCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(installCtx, command, spec.Args...)
	cmd.Dir = specRoot(spec)
	cmd.Env = append(os.Environ(), envPairs(spec.Env)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(installCtx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("timed out installing %s", spec.Command)
		}
		slog.Debug("LSP install failed", "command", command, "args", spec.Args, "output", string(output))
		return fmt.Errorf("failed to install %s: %w", spec.Command, err)
	}
	return nil
}

func specRoot(spec Spec) string {
	if prefix := spec.Env["npm_config_prefix"]; prefix != "" {
		return prefix
	}
	if gobin := spec.Env["GOBIN"]; gobin != "" {
		return filepath.Dir(gobin)
	}
	return DefaultRoot()
}

func envPairs(env map[string]string) []string {
	out := make([]string, 0, len(env))
	for key, value := range env {
		out = append(out, key+"="+value)
	}
	return out
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode().Perm()&0o111 != 0
}

func discoverGoToolchain() string {
	candidates := []string{
		filepath.Join(os.TempDir(), "go", "bin", executableName("go")),
		"/usr/local/go/bin/go",
		"/opt/homebrew/bin/go",
		"/usr/local/bin/go",
	}
	for _, candidate := range candidates {
		if isExecutable(candidate) {
			return candidate
		}
	}
	return ""
}

func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".cmd"
	}
	return name
}
