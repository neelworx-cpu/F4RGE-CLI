package cmd

import (
	"cmp"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/x/ansi"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/client"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/config"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/controlplane"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/managedlogin"
	f4rgesession "github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/session"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/oauth"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/oauth/copilot"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/oauth/hyper"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/version"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Aliases: []string{"auth"},
	Use:     "login [platform]",
	Short:   "Sign in to F4RGE",
	Long: `Sign in to F4RGE and connect this CLI to your organization.

The managed F4RGE login flow is the default product path. Provider-specific
login modes remain available for internal or legacy development flows while the
F4RGE Auth device flow is wired to the Web control plane.`,
	Example: `
# Sign in to F4RGE
4rged login

# Legacy/internal provider login
4rged login hyper
4rged login copilot

# Force re-authentication even if already logged in
4rged login -f
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
		if len(args) > 0 {
			provider = args[0]
		}
		force, _ := cmd.Flags().GetBool("force")
		switch provider {
		case "f4rge":
			return loginF4RGE(force)
		case "hyper":
			if os.Getenv("F4RGE_CLI_ENABLE_LEGACY_PROVIDER_AUTH") != "1" {
				return fmt.Errorf("provider-specific login is disabled for F4RGE managed CLI")
			}
			return loginHyper(c, ws.ID, force)
		case "copilot", "github", "github-copilot":
			if os.Getenv("F4RGE_CLI_ENABLE_LEGACY_PROVIDER_AUTH") != "1" {
				return fmt.Errorf("provider-specific login is disabled for F4RGE managed CLI")
			}
			return loginCopilot(c, ws.ID, force)
		default:
			return fmt.Errorf("unknown platform: %s", args[0])
		}
	},
}

func init() {
	loginCmd.Flags().BoolP("force", "f", false, "Force re-authentication even if already logged in")
}

func loginF4RGE(force bool) error {
	if !force {
		session, err := f4rgesession.Load()
		if err != nil {
			return err
		}
		if f4rgesession.IsUsable(session) {
			fmt.Println("You are already signed in to F4RGE.")
			fmt.Println("Use --force to start a fresh sign-in.")
			return nil
		}
	}

	deviceAuth := f4rgesession.StartDeviceAuth()
	controlPlane := controlplane.New()
	started, startErr := controlPlane.StartCLIAuth(controlplane.RuntimeSessionRequest{
		Surface:       "cli",
		DeviceLabel:   "4RGED CLI",
		Platform:      runtime.GOOS + "/" + runtime.GOARCH,
		ClientVersion: version.Version,
	})
	if startErr == nil && started != nil {
		deviceAuth.DeviceCode = started.DeviceCode
		deviceAuth.UserCode = started.UserCode
		deviceAuth.ExpiresIn = started.ExpiresIn
		deviceAuth.Interval = started.Interval
		deviceAuth.VerificationURL = f4rgesession.AuthURL() + "?device_code=" + url.QueryEscape(started.DeviceCode) + "&user_code=" + url.QueryEscape(started.UserCode)
	}
	fmt.Println("Opening F4RGE sign-in...")
	fmt.Println()
	fmt.Println("If your browser does not open, visit:")
	fmt.Println(deviceAuth.VerificationURL)
	fmt.Println()
	fmt.Println("Code:", lipgloss.NewStyle().Bold(true).Render(deviceAuth.UserCode))
	if err := browser.OpenURL(deviceAuth.VerificationURL); err != nil {
		fmt.Println()
		fmt.Println("Could not open your browser automatically.")
	}
	if startErr == nil && started != nil {
		deadline := time.Now().Add(time.Duration(deviceAuth.ExpiresIn) * time.Second)
		interval := time.Duration(max(deviceAuth.Interval, 1)) * time.Second
		for time.Now().Before(deadline) {
			time.Sleep(interval)
			poll, err := controlPlane.PollCLIAuth(deviceAuth.DeviceCode)
			if err != nil {
				continue
			}
			if poll.Status == "completed" && poll.RuntimeSession.Token != "" {
				if err := managedlogin.Finalize(controlPlane, poll); err != nil {
					return err
				}
				fmt.Println()
				fmt.Println("Signed in to F4RGE.")
				return nil
			}
		}
		return fmt.Errorf("sign-in timed out; please try again")
	}
	return startErr
}

func loginHyper(c *client.Client, wsID string, force bool) error {
	ctx := getLoginContext()

	if !force {
		cfg, err := c.GetConfig(ctx, wsID)
		if err == nil && cfg != nil {
			if pc, ok := cfg.Providers.Get("hyper"); ok && pc.OAuthToken != nil {
				fmt.Println("You are already logged in to Hyper.")
				fmt.Println("Use --force to re-authenticate.")
				return nil
			}
		}
	}

	resp, err := hyper.InitiateDeviceAuth(ctx)
	if err != nil {
		return err
	}

	if clipboard.WriteAll(resp.UserCode) == nil {
		fmt.Println("The following code should be on clipboard already:")
	} else {
		fmt.Println("Copy the following code:")
	}

	fmt.Println()
	fmt.Println(lipgloss.NewStyle().Bold(true).Render(resp.UserCode))
	fmt.Println()
	fmt.Println("Press enter to open this URL, and then paste it there:")
	fmt.Println()
	fmt.Println(lipgloss.NewStyle().Hyperlink(resp.VerificationURL, "id=hyper").Render(resp.VerificationURL))
	fmt.Println()
	waitEnter()
	if err := browser.OpenURL(resp.VerificationURL); err != nil {
		fmt.Println("Could not open the URL. You'll need to manually open the URL in your browser.")
	}

	fmt.Println("Exchanging authorization code...")
	refreshToken, err := hyper.PollForToken(ctx, resp.DeviceCode, resp.ExpiresIn)
	if err != nil {
		return err
	}

	fmt.Println("Exchanging refresh token for access token...")
	token, err := hyper.ExchangeToken(ctx, refreshToken)
	if err != nil {
		return err
	}

	fmt.Println("Verifying access token...")
	introspect, err := hyper.IntrospectToken(ctx, token.AccessToken)
	if err != nil {
		return fmt.Errorf("token introspection failed: %w", err)
	}
	if !introspect.Active {
		return fmt.Errorf("access token is not active")
	}

	if err := cmp.Or(
		c.SetConfigField(ctx, wsID, config.ScopeGlobal, "providers.hyper.api_key", token.AccessToken),
		c.SetConfigField(ctx, wsID, config.ScopeGlobal, "providers.hyper.oauth", token),
	); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("You're now authenticated with Hyper!")
	return nil
}

func loginCopilot(c *client.Client, wsID string, force bool) error {
	loginCtx := getLoginContext()

	if !force {
		cfg, err := c.GetConfig(loginCtx, wsID)
		if err == nil && cfg != nil {
			if pc, ok := cfg.Providers.Get("copilot"); ok && pc.OAuthToken != nil {
				fmt.Println("You are already logged in to GitHub Copilot.")
				fmt.Println("Use --force to re-authenticate.")
				return nil
			}
		}
	}

	diskToken, hasDiskToken := copilot.RefreshTokenFromDisk()
	var token *oauth.Token

	switch {
	case hasDiskToken:
		fmt.Println("Found existing GitHub Copilot token on disk. Using it to authenticate...")

		t, err := copilot.RefreshToken(loginCtx, diskToken)
		if err != nil {
			return fmt.Errorf("unable to refresh token from disk: %w", err)
		}
		token = t
	default:
		fmt.Println("Requesting device code from GitHub...")
		dc, err := copilot.RequestDeviceCode(loginCtx)
		if err != nil {
			return err
		}

		fmt.Println()
		fmt.Println("Open the following URL and follow the instructions to authenticate with GitHub Copilot:")
		fmt.Println()
		fmt.Println(lipgloss.NewStyle().Hyperlink(dc.VerificationURI, "id=copilot").Render(dc.VerificationURI))
		fmt.Println()
		fmt.Println("Code:", lipgloss.NewStyle().Bold(true).Render(dc.UserCode))
		fmt.Println()
		fmt.Println("Waiting for authorization...")

		t, err := copilot.PollForToken(loginCtx, dc)
		if err == copilot.ErrNotAvailable {
			fmt.Println()
			fmt.Println("GitHub Copilot is unavailable for this account. To signup, go to the following page:")
			fmt.Println()
			fmt.Println(lipgloss.NewStyle().Hyperlink(copilot.SignupURL, "id=copilot-signup").Render(copilot.SignupURL))
			fmt.Println()
			fmt.Println("You may be able to request free access if eligible. For more information, see:")
			fmt.Println()
			fmt.Println(lipgloss.NewStyle().Hyperlink(copilot.FreeURL, "id=copilot-free").Render(copilot.FreeURL))
		}
		if err != nil {
			return err
		}
		token = t
	}

	if err := cmp.Or(
		c.SetConfigField(loginCtx, wsID, config.ScopeGlobal, "providers.copilot.api_key", token.AccessToken),
		c.SetConfigField(loginCtx, wsID, config.ScopeGlobal, "providers.copilot.oauth", token),
	); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("You're now authenticated with GitHub Copilot!")
	return nil
}

func getLoginContext() context.Context {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	go func() {
		<-ctx.Done()
		cancel()
		os.Exit(1)
	}()
	return ctx
}

func waitEnter() {
	_, _ = fmt.Scanln()
}
