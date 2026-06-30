package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeInstaller struct {
	err error
}

func (f fakeInstaller) Install(context.Context, Spec) error {
	return f.err
}

func TestKnownCommand(t *testing.T) {
	require.True(t, KnownCommand("typescript-language-server"))
	require.True(t, KnownCommand("vtsls"))
	require.True(t, KnownCommand("pyright-langserver"))
	require.True(t, KnownCommand("gopls"))
	require.False(t, KnownCommand("python"))
}

func TestResolveUsesManagedBinary(t *testing.T) {
	root := t.TempDir()
	manager := NewManager(root, fakeInstaller{})
	bin := filepath.Join(root, "bin")
	require.NoError(t, os.MkdirAll(bin, 0o700))
	path := filepath.Join(bin, executableName("gopls"))
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"), 0o700))

	resolved, ok, status := manager.Resolve(context.Background(), "go", "gopls")

	require.True(t, ok)
	require.Equal(t, path, resolved)
	require.Equal(t, StatusInstalled, status.Kind)
	require.True(t, status.Managed)
}

func TestResolveUsesNpmBinShim(t *testing.T) {
	root := t.TempDir()
	manager := NewManager(root, fakeInstaller{})
	bin := filepath.Join(root, "node_modules", ".bin")
	require.NoError(t, os.MkdirAll(bin, 0o700))
	path := filepath.Join(bin, executableName("typescript-language-server"))
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"), 0o700))

	resolved, ok, status := manager.Resolve(context.Background(), "ts_ls", "typescript-language-server")

	require.True(t, ok)
	require.Equal(t, path, resolved)
	require.Equal(t, StatusInstalled, status.Kind)
	require.True(t, status.Managed)
}

func TestResolveUnknownCommandUnavailable(t *testing.T) {
	manager := NewManager(t.TempDir(), fakeInstaller{})

	_, ok, status := manager.Resolve(context.Background(), "custom", "missing-f4rge-test-command")

	require.False(t, ok)
	require.Equal(t, StatusUnavailable, status.Kind)
}

func TestDiscoverGoToolchainFindsTempGo(t *testing.T) {
	if _, err := os.Stat(filepath.Join(os.TempDir(), "go", "bin", executableName("go"))); err != nil {
		t.Skip("temporary Go toolchain is not installed")
	}
	require.NotEmpty(t, discoverGoToolchain())
}
