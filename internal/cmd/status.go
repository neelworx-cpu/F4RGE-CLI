package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/runtimebundle"
	f4rgesession "github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/session"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/version"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show F4RGE account and runtime status",
	Long:  "Show the signed-in F4RGE account, organization, policy, model catalog, credential lease, and CLI runtime status.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cwd, err := ResolveCwd(cmd)
		if err != nil {
			return err
		}
		dataDir, _ := cmd.Flags().GetString("data-dir")

		fmt.Println("4RGED status")
		fmt.Println()
		fmt.Println("CLI version:        " + version.Version)
		fmt.Println("Workspace:          " + cwd)
		if dataDir == "" {
			fmt.Println("Data directory:     default")
		} else {
			fmt.Println("Data directory:     " + dataDir)
		}
		session, err := f4rgesession.Load()
		if err != nil {
			return err
		}
		if session == nil {
			fmt.Println("F4RGE account:      not signed in")
			fmt.Println("Organization:       unavailable")
			fmt.Println("Policy snapshot:    unavailable")
			fmt.Println("Model catalog:      unavailable")
			fmt.Println("Gateway:            not configured")
			fmt.Println()
			fmt.Println("Next step: open 4RGED and use the F4RGE sign-in dialog.")
			return nil
		}
		if !f4rgesession.IsUsable(session) {
			fmt.Println("F4RGE account:      not ready")
			fmt.Println("Reason:             missing " + strings.Join(f4rgesession.MissingReadinessFields(session), ", "))
			fmt.Println()
			fmt.Println("Next step: open 4RGED and refresh sign-in from the F4RGE sign-in dialog.")
			return fmt.Errorf("F4RGE sign-in is incomplete")
		}

		status := "signed in"
		if f4rgesession.IsExpired(session) {
			status = "expired"
		}
		fmt.Println("F4RGE account:      " + status)
		fmt.Println("User:               " + fallback(session.UserDisplayName, session.UserEmail))
		if session.UserEmail != "" && session.UserEmail != session.UserDisplayName {
			fmt.Println("Email:              " + session.UserEmail)
		}
		fmt.Println("Organization:       " + session.OrganizationName)
		fmt.Println("Policy snapshot:    " + session.PolicyVersion)
		fmt.Println("Model catalog:      " + session.ModelCatalogVersion)
		if bundle, err := runtimebundle.LoadCached(); err == nil && runtimebundle.Validate(bundle, session) == nil {
			fmt.Println("Runtime bundle:     ready (" + fallback(bundle.RuntimeConfig.Version, "current") + ")")
			fmt.Println("Runtime hash:       " + session.RuntimeBundleHash)
		} else {
			fmt.Println("Runtime bundle:     not ready")
		}
		fmt.Println("Model access:       " + modelAccessStatus(session))
		fmt.Println("F4RGE endpoint:     " + fallback(session.PlatformEndpoint, "not configured"))
		if session.ExpiresAt > 0 {
			fmt.Println("Token expires:      " + time.Unix(session.ExpiresAt, 0).Format(time.RFC3339))
		}
		return nil
	},
}

func fallback(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func modelAccessStatus(session *f4rgesession.ManagedSession) string {
	if bundle, err := runtimebundle.LoadCached(); err == nil && runtimebundle.Validate(bundle, session) == nil && len(bundle.Models) > 0 {
		return fmt.Sprintf("%d models available", len(bundle.Models))
	}
	return "unavailable"
}
