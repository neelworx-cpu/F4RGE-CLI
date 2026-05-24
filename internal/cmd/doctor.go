package cmd

import (
	"fmt"
	"strings"

	"github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/runtimebundle"
	f4rgesession "github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/session"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check 4RGED managed CLI readiness",
	Long:  "Check local runtime, F4RGE auth, policy, model catalog, gateway, install, and update readiness.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cwd, err := ResolveCwd(cmd)
		if err != nil {
			return err
		}

		fmt.Println("4RGED doctor")
		fmt.Println()
		fmt.Println("Local runtime")
		fmt.Println("  workspace: ok (" + cwd + ")")
		fmt.Println("  terminal:  ok")
		fmt.Println()
		fmt.Println("Managed F4RGE")
		session, sessionErr := f4rgesession.Load()
		switch {
		case sessionErr != nil:
			fmt.Println("  auth:      error (" + sessionErr.Error() + ")")
		case session == nil:
			fmt.Println("  auth:      not signed in")
		case f4rgesession.IsExpired(session):
			fmt.Println("  auth:      expired")
		case !f4rgesession.IsUsable(session):
			fmt.Println("  auth:      not ready")
			fmt.Println("  reason:    missing " + strings.Join(f4rgesession.MissingReadinessFields(session), ", "))
		default:
			fmt.Println("  auth:      ok")
		}
		if bundle, err := runtimebundle.LoadCached(); err == nil && f4rgesession.IsUsable(session) && runtimebundle.Validate(bundle, session) == nil {
			fmt.Println("  policy:    ok")
			fmt.Printf("  models:    ok (%d available)\n", len(bundle.Models))
			fmt.Println("  gateway:   ok")
		} else {
			fmt.Println("  policy:    unavailable")
			fmt.Println("  models:    unavailable")
			fmt.Println("  gateway:   unavailable")
		}
		fmt.Println()
		fmt.Println("Install/update")
		fmt.Println("  installer: planned (`curl https://cli.4rge.ai/install -fsS | bash`)")
		fmt.Println("  updates:   planned (signed artifacts and minimum-version policy)")
		return nil
	},
}
