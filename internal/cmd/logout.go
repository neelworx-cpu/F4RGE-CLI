package cmd

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"os/signal"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/client"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/config"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/modelcatalog"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/promptbundle"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/runtimebundle"
	f4rgesession "github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/session"
	"github.com/spf13/cobra"
)

var (
	logoutHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	logoutItemStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	logoutPromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("215"))
)

var logoutCmd = &cobra.Command{
	Aliases: []string{"signout"},
	Use:     "logout [platform]",
	Short:   "Sign out of F4RGE",
	Long: `Sign out of F4RGE, removing locally stored managed session credentials.

Provider-specific logout modes remain available for internal or legacy
development flows while the F4RGE Auth device flow is wired to the Web control
plane.`,
	Example: `
# Sign out from F4RGE
4rged logout

# Legacy/internal provider logout
4rged logout hyper
4rged logout copilot
  `,
	ValidArgs: []cobra.Completion{
		"f4rge",
		"hyper",
		"copilot",
		"github",
		"github-copilot",
	},
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, ws, cleanup, err := connectToServer(cmd)
		if err != nil {
			return err
		}
		defer cleanup()

		progressEnabled := ws.Config.Options.Progress == nil || *ws.Config.Options.Progress
		if progressEnabled && supportsProgressBar() {
			_, _ = fmt.Fprintf(os.Stderr, ansi.SetIndeterminateProgressBar)
			defer func() { _, _ = fmt.Fprintf(os.Stderr, ansi.ResetProgressBar) }()
		}

		provider := "f4rge"
		if len(args) == 0 {
			// Managed F4RGE is the default customer-facing path.
		} else {
			provider = args[0]
		}

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Print(logoutPromptStyle.Render(fmt.Sprintf("Are you sure you want to logout %s? (y/N) ", provider)))
			var response string
			_, err := fmt.Scanln(&response)
			if err != nil || (response != "y" && response != "Y" && response != "yes" && response != "Yes" && response != "YES") {
				fmt.Println(logoutHeaderStyle.Render("Logout cancelled."))
				return nil
			}
		}

		switch provider {
		case "f4rge":
			return logoutF4RGE()
		case "hyper":
			if os.Getenv("F4RGE_CLI_ENABLE_LEGACY_PROVIDER_AUTH") != "1" {
				return fmt.Errorf("provider-specific logout is disabled for F4RGE managed CLI")
			}
			return logoutHyper(c, ws.ID)
		case "copilot", "github", "github-copilot":
			if os.Getenv("F4RGE_CLI_ENABLE_LEGACY_PROVIDER_AUTH") != "1" {
				return fmt.Errorf("provider-specific logout is disabled for F4RGE managed CLI")
			}
			return logoutCopilot(c, ws.ID)
		default:
			return fmt.Errorf("unknown platform: %s", provider)
		}
	},
}

func logoutF4RGE() error {
	if err := f4rgesession.Clear(); err != nil {
		return err
	}
	_ = os.Remove(modelcatalog.Path())
	_ = os.Remove(promptbundle.Path())
	_ = os.Remove(runtimebundle.Path())
	fmt.Println(logoutHeaderStyle.Render("Signed out of F4RGE."))
	return nil
}

func logoutHyper(c *client.Client, wsID string) error {
	ctx := getLogoutContext()

	if err := cmp.Or(
		c.RemoveConfigField(ctx, wsID, config.ScopeGlobal, "providers.hyper.api_key"),
		c.RemoveConfigField(ctx, wsID, config.ScopeGlobal, "providers.hyper.oauth"),
	); err != nil {
		return err
	}

	fmt.Println(logoutHeaderStyle.Render("Successfully logged out of Hyper."))
	return nil
}

func logoutCopilot(c *client.Client, wsID string) error {
	ctx := getLogoutContext()

	if err := cmp.Or(
		c.RemoveConfigField(ctx, wsID, config.ScopeGlobal, "providers.copilot.api_key"),
		c.RemoveConfigField(ctx, wsID, config.ScopeGlobal, "providers.copilot.oauth"),
	); err != nil {
		return err
	}

	fmt.Println(logoutHeaderStyle.Render("Successfully logged out of GitHub Copilot."))
	return nil
}

func pickLoggedInProvider(c *client.Client, wsID string) (string, error) {
	ctx := getLogoutContext()

	cfg, err := c.GetConfig(ctx, wsID)
	if err != nil {
		return "", fmt.Errorf("failed to get config: %w", err)
	}

	type loggedInProvider struct {
		id   string
		name string
	}

	var loggedIn []loggedInProvider
	for p := range cfg.Providers.Seq() {
		if p.OAuthToken != nil || p.APIKey != "" {
			name := p.Name
			if name == "" {
				name = p.ID
			}
			loggedIn = append(loggedIn, loggedInProvider{id: p.ID, name: name})
		}
	}

	if len(loggedIn) == 0 {
		fmt.Println(logoutPromptStyle.Render("You are not logged in to any platform."))
		return "", nil
	}

	if len(loggedIn) == 1 {
		return loggedIn[0].id, nil
	}

	fmt.Println(logoutHeaderStyle.Render("Logged-in platforms:"))
	for i, p := range loggedIn {
		fmt.Println(logoutItemStyle.Render(fmt.Sprintf("  %d. %s", i+1, p.name)))
	}
	fmt.Print(logoutPromptStyle.Render(fmt.Sprintf("Select a platform to logout (1-%d): ", len(loggedIn))))

	var choice int
	_, err = fmt.Scanln(&choice)
	if err != nil || choice < 1 || choice > len(loggedIn) {
		fmt.Println(logoutHeaderStyle.Render("Logout cancelled."))
		return "", nil
	}

	return loggedIn[choice-1].id, nil
}

func init() {
	logoutCmd.Flags().BoolP("force", "f", false, "Skip logout confirmation prompt")
}

func getLogoutContext() context.Context {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	go func() {
		<-ctx.Done()
		cancel()
		os.Exit(1)
	}()
	return ctx
}
