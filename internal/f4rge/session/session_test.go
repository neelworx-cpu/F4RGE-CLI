package session

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestManagedSessionStoreRoundTrip(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("F4RGED_GLOBAL_DATA", dataDir)

	require.Equal(t, filepath.Join(dataDir, sessionFileName), Path())

	loaded, err := Load()
	require.NoError(t, err)
	require.Nil(t, loaded)

	saved := ManagedSession{
		AccessToken:      "runtime-token",
		RuntimeSessionID: "runtime-session",
		UserEmail:        "user@example.com",
		OrganizationID:   "123",
		OrganizationName: "Example Org",
		ExpiresAt:        time.Now().Add(time.Hour).Unix(),
	}
	require.NoError(t, Save(saved))

	loaded, err = Load()
	require.NoError(t, err)
	require.NotNil(t, loaded)
	require.Equal(t, saved.UserEmail, loaded.UserEmail)
	require.Equal(t, saved.OrganizationName, loaded.OrganizationName)

	require.NoError(t, Clear())
	loaded, err = Load()
	require.NoError(t, err)
	require.Nil(t, loaded)
}

func TestStartDeviceAuth(t *testing.T) {
	auth := StartDeviceAuth()
	require.NotEmpty(t, auth.DeviceCode)
	require.Equal(t, "F4RGED-CLI", auth.UserCode)
	require.Equal(t, AuthURL(), auth.VerificationURL)
	require.Positive(t, auth.ExpiresIn)
	require.Positive(t, auth.Interval)
}
