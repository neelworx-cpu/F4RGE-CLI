package model

import (
	"os"

	"github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/runtimebundle"
	f4rgesession "github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/session"
)

func (m *UI) managedSession() *f4rgesession.ManagedSession {
	session, err := f4rgesession.Load()
	if err != nil || !f4rgesession.IsUsable(session) {
		return nil
	}
	return session
}

func (m *UI) isManagedSignedIn() bool {
	return m.managedSession() != nil
}

func (m *UI) isManagedRuntimeReady() bool {
	session, err := f4rgesession.Load()
	if err != nil || !f4rgesession.IsRuntimeSessionUsable(session) {
		return false
	}
	if len(session.RuntimeScopes) == 0 {
		return false
	}
	if !f4rgesession.HasRuntimeScope(session, "runtime.credentials.issue") {
		return false
	}
	return runtimebundle.Ready()
}

func (m *UI) canUseLegacyProviderConfig() bool {
	return os.Getenv("F4RGE_CLI_ENABLE_LEGACY_PROVIDER_AUTH") == "1" && m.com.Config().IsConfigured()
}
