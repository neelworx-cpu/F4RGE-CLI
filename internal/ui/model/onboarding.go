package model

import (
	"fmt"
	"net/url"
	"runtime"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/controlplane"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/managedlogin"
	f4rgesession "github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/session"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/home"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/ui/common"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/ui/dialog"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/ui/util"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/version"
	"github.com/pkg/browser"
)

// updateOnboardingView handles keyboard input for the F4RGE sign-in prompt.
func (m *UI) updateOnboardingView(msg tea.KeyPressMsg) (cmds []tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Initialize.Enter), key.Matches(msg, m.keyMap.Initialize.Yes):
		cmds = append(cmds, m.openF4RGEAuthDialog())
	case key.Matches(msg, m.keyMap.Initialize.No):
		return []tea.Cmd{util.CmdHandler(util.NewInfoMsg("Sign-in is required before using 4RGED."))}
	}
	return cmds
}

func (m *UI) openF4RGEAuthDialog() tea.Cmd {
	if m.dialog.ContainsDialog(dialog.F4RGEAuthID) {
		m.dialog.BringToFront(dialog.F4RGEAuthID)
		return nil
	}
	m.dialog.OpenDialog(dialog.NewF4RGEAuth(m.com))
	return nil
}

func openF4RGEAuthURL() tea.Msg {
	controlPlane := controlplane.New()
	started, err := controlPlane.StartCLIAuth(controlplane.RuntimeSessionRequest{
		Surface:       "cli",
		DeviceLabel:   "4RGED CLI",
		Platform:      runtime.GOOS + "/" + runtime.GOARCH,
		ClientVersion: version.Version,
	})
	if err != nil {
		return util.NewErrorMsg(err)
	}
	verificationURL := f4rgesession.AuthURL() + "?device_code=" + url.QueryEscape(started.DeviceCode) + "&user_code=" + url.QueryEscape(started.UserCode)
	if err := browser.OpenURL(verificationURL); err != nil {
		return util.NewWarnMsg("Open this URL in your browser: " + verificationURL)
	}
	deadline := time.Now().Add(time.Duration(started.ExpiresIn) * time.Second)
	interval := time.Duration(max(started.Interval, 1)) * time.Second
	for time.Now().Before(deadline) {
		time.Sleep(interval)
		poll, err := controlPlane.PollCLIAuth(started.DeviceCode)
		if err != nil {
			continue
		}
		if poll.Status != "completed" || poll.RuntimeSession.Token == "" {
			continue
		}
		if err := managedlogin.Finalize(controlPlane, poll); err != nil {
			return util.NewErrorMsg(err)
		}
		return f4rgeSignInCompletedMsg{}
	}
	return util.NewErrorMsg(fmt.Errorf("sign-in timed out; please try again"))
}

// onboardingView renders the managed first-run sign-in surface.
func (m *UI) onboardingView() string {
	s := m.com.Styles.Initialize

	header := s.Header.Render("Welcome to 4RGED")
	desc := s.Content.Render("Sign in to your F4RGE account to continue.")
	detail := s.Content.Render("Your organization controls available models, access policy, and usage for this CLI.")
	next := s.Content.Render("Press enter to start sign-in.")
	footer := s.Content.Render("Esc quits. Commands unlock after sign-in.")

	width := min(m.layout.main.Dx(), 72)

	return lipgloss.NewStyle().
		Width(width).
		Height(m.layout.main.Dy()).
		PaddingBottom(1).
		AlignVertical(lipgloss.Bottom).
		Render(strings.Join(
			[]string{
				header,
				desc,
				detail,
				next,
				footer,
			},
			"\n\n",
		))
}

// markProjectInitializedCmd marks the current project as initialized in the config.
func (m *UI) markProjectInitializedCmd() tea.Cmd {
	return func() tea.Msg {
		if err := m.com.Workspace.MarkProjectInitialized(); err != nil {
			return util.InfoMsg{
				Type: util.InfoTypeError,
				Msg:  fmt.Sprintf("Failed to mark project as initialized: %v", err),
				TTL:  15 * time.Second,
			}
		}
		return nil
	}
}

// updateInitializeView handles keyboard input for the project initialization prompt.
func (m *UI) updateInitializeView(msg tea.KeyPressMsg) (cmds []tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Initialize.Enter):
		if m.onboarding.yesInitializeSelected {
			cmds = append(cmds, m.initializeProject())
		} else {
			cmds = append(cmds, m.skipInitializeProject())
		}
	case key.Matches(msg, m.keyMap.Initialize.Switch):
		m.onboarding.yesInitializeSelected = !m.onboarding.yesInitializeSelected
	case key.Matches(msg, m.keyMap.Initialize.Yes):
		cmds = append(cmds, m.initializeProject())
	case key.Matches(msg, m.keyMap.Initialize.No):
		cmds = append(cmds, m.skipInitializeProject())
	}
	return cmds
}

// initializeProject starts project initialization and transitions to the landing view.
func (m *UI) initializeProject() tea.Cmd {
	// clear the session
	var cmds []tea.Cmd
	if cmd := m.newSession(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	initialize := func() tea.Msg {
		initPrompt, err := m.com.Workspace.InitializePrompt()
		if err != nil {
			return util.InfoMsg{
				Type: util.InfoTypeError,
				Msg:  fmt.Sprintf("Failed to initialize project: %v", err),
			}
		}
		return sendMessageMsg{Content: initPrompt}
	}
	// Mark the project as initialized
	cmds = append(cmds, initialize, m.markProjectInitializedCmd())

	return tea.Sequence(cmds...)
}

// skipInitializeProject skips project initialization and transitions to the landing view.
func (m *UI) skipInitializeProject() tea.Cmd {
	// TODO: initialize the project
	m.setState(uiLanding, uiFocusEditor)
	// mark the project as initialized
	return m.markProjectInitializedCmd()
}

// initializeView renders the project initialization prompt with Yes/No buttons.
func (m *UI) initializeView() string {
	s := m.com.Styles.Initialize
	cwd := home.Short(m.com.Workspace.WorkingDir())
	initFile := m.com.Config().Options.InitializeAs

	header := s.Header.Render("Would you like to initialize this project?")
	path := s.Accent.PaddingLeft(2).Render(cwd)
	desc := s.Content.Render(fmt.Sprintf("When I initialize your codebase I examine the project and put the result into an %s file which serves as general context.", initFile))
	hint := s.Content.Render("You can also initialize anytime via ") + s.Accent.Render("ctrl+p") + s.Content.Render(".")
	prompt := s.Content.Render("Would you like to initialize now?")

	buttons := common.ButtonGroup(m.com.Styles, []common.ButtonOpts{
		{Text: "Yep!", Selected: m.onboarding.yesInitializeSelected},
		{Text: "Nope", Selected: !m.onboarding.yesInitializeSelected},
	}, " ")

	// max width 60 so the text is compact
	width := min(m.layout.main.Dx(), 60)

	return lipgloss.NewStyle().
		Width(width).
		Height(m.layout.main.Dy()).
		PaddingBottom(1).
		AlignVertical(lipgloss.Bottom).
		Render(strings.Join(
			[]string{
				header,
				path,
				desc,
				hint,
				prompt,
				buttons,
			},
			"\n\n",
		))
}
