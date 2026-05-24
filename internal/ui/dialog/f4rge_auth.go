package dialog

import (
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	f4rgesession "github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/session"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/ui/common"
)

const F4RGEAuthID = "f4rge_auth"

// F4RGEAuth is the first-run managed sign-in dialog.
type F4RGEAuth struct {
	com                  *common.Common
	selectedAuthenticate bool
	help                 help.Model
	keyMap               struct {
		LeftRight,
		Tab,
		Authenticate key.Binding
		Close key.Binding
	}
}

var _ Dialog = (*F4RGEAuth)(nil)

func NewF4RGEAuth(com *common.Common) *F4RGEAuth {
	d := &F4RGEAuth{com: com, selectedAuthenticate: true}
	d.help = help.New()
	d.help.Styles = com.Styles.DialogHelpStyles()
	d.keyMap.LeftRight = key.NewBinding(
		key.WithKeys("left", "right"),
		key.WithHelp("←/→", "choose"),
	)
	d.keyMap.Tab = key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch"),
	)
	d.keyMap.Authenticate = key.NewBinding(
		key.WithKeys("enter", " ", "a", "A"),
		key.WithHelp("enter", "confirm"),
	)
	d.keyMap.Close = key.NewBinding(
		key.WithKeys("esc", "alt+esc", "c", "C"),
		key.WithHelp("esc", "cancel"),
	)
	return d
}

func (*F4RGEAuth) ID() string {
	return F4RGEAuthID
}

func (d *F4RGEAuth) HandleMsg(msg tea.Msg) Action {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, d.keyMap.LeftRight, d.keyMap.Tab):
			d.selectedAuthenticate = !d.selectedAuthenticate
		case key.Matches(msg, d.keyMap.Authenticate):
			if d.selectedAuthenticate {
				return ActionOpenF4RGEAuth{}
			}
			return ActionClose{}
		case key.Matches(msg, d.keyMap.Close):
			return ActionClose{}
		}
	}
	return nil
}

func (d *F4RGEAuth) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	t := d.com.Styles
	width := min(64, max(42, area.Dx()-8))
	contentStyle := t.Dialog.Quit.Content.Width(width - t.Dialog.Quit.Frame.GetHorizontalFrameSize())

	title := t.Dialog.TitleText.Render("Sign in to F4RGE")
	body := contentStyle.Render(strings.Join([]string{
		"Authenticate this CLI with your F4RGE account.",
		"",
		"4RGED will open a browser page where you can sign in. After F4RGE Auth is connected, this flow will return your organization, policy, and available models.",
		"",
		"URL: " + f4rgesession.AuthURL(),
	}, "\n"))
	buttons := common.ButtonGroup(t, []common.ButtonOpts{
		{Text: "Authenticate", Selected: d.selectedAuthenticate, Padding: 2},
		{Text: "Cancel", Selected: !d.selectedAuthenticate, Padding: 2},
	}, " ")
	helpView := t.Dialog.HelpView.Render(d.help.View(d))

	view := t.Dialog.Quit.Frame.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		body,
		"",
		buttons,
		"",
		helpView,
	))
	DrawCenter(scr, area, view)
	return nil
}

func (d *F4RGEAuth) ShortHelp() []key.Binding {
	return []key.Binding{d.keyMap.LeftRight, d.keyMap.Authenticate, d.keyMap.Close}
}

func (d *F4RGEAuth) FullHelp() [][]key.Binding {
	return [][]key.Binding{d.ShortHelp()}
}
