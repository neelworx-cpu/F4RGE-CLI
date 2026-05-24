package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoginCmd_Aliases(t *testing.T) {
	t.Parallel()

	require.Equal(t, "auth", loginCmd.Aliases[0])
}

func TestLoginCmd_ForceFlag(t *testing.T) {
	t.Parallel()

	flag := loginCmd.Flags().Lookup("force")
	require.NotNil(t, flag)
	require.Equal(t, "f", flag.Shorthand)
}

func TestLoginCmd_DefaultsToF4RGE(t *testing.T) {
	t.Parallel()

	require.Contains(t, loginCmd.ValidArgs, "f4rge")
	require.Contains(t, loginCmd.Short, "F4RGE")
}
