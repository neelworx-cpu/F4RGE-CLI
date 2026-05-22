package dialog

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/ui/common"
)

// SplitChatID is the identifier for the split-chat choice dialog.
const SplitChatID = "split_chat"

// SplitChat asks what should be shown in the second chat pane.
type SplitChat struct {
	com                *common.Common
	targetPaneName     string
	replacingPane      bool
	selectedNewSession bool
	keyMap             struct {
		LeftRight,
		EnterSpace,
		NewSession,
		ExistingSession,
		Tab,
		Close key.Binding
	}
}

var _ Dialog = (*SplitChat)(nil)

// NewSplitChat creates a split-chat choice dialog.
func NewSplitChat(com *common.Common, targetPaneName string, replacingPane bool) *SplitChat {
	s := &SplitChat{
		com:                com,
		targetPaneName:     targetPaneName,
		replacingPane:      replacingPane,
		selectedNewSession: true,
	}
	s.keyMap.LeftRight = key.NewBinding(
		key.WithKeys("left", "right"),
		key.WithHelp("left/right", "switch options"),
	)
	s.keyMap.EnterSpace = key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter/space", "confirm"),
	)
	s.keyMap.NewSession = key.NewBinding(
		key.WithKeys("n", "N"),
		key.WithHelp("n", "new session"),
	)
	s.keyMap.ExistingSession = key.NewBinding(
		key.WithKeys("s", "S", "e", "E"),
		key.WithHelp("s/e", "sessions"),
	)
	s.keyMap.Tab = key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch options"),
	)
	s.keyMap.Close = CloseKey
	return s
}

// ID implements [Dialog].
func (*SplitChat) ID() string {
	return SplitChatID
}

// HandleMsg implements [Dialog].
func (s *SplitChat) HandleMsg(msg tea.Msg) Action {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, s.keyMap.Close):
			return ActionClose{}
		case key.Matches(msg, s.keyMap.LeftRight, s.keyMap.Tab):
			s.selectedNewSession = !s.selectedNewSession
		case key.Matches(msg, s.keyMap.NewSession):
			return ActionSplitChatNewSession{}
		case key.Matches(msg, s.keyMap.ExistingSession):
			return ActionSplitChatExistingSession{}
		case key.Matches(msg, s.keyMap.EnterSpace):
			if s.selectedNewSession {
				return ActionSplitChatNewSession{}
			}
			return ActionSplitChatExistingSession{}
		}
	}
	return nil
}

// Draw implements [Dialog].
func (s *SplitChat) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	question := "Open split chat with:"
	descriptionText := "Choose what to place in the second pane. The sidebar stays on the right."
	if s.replacingPane {
		question = "Replace " + s.targetPaneName + " chat with:"
		descriptionText = "The other split pane stays open. Duplicate sessions are not allowed."
	} else if s.targetPaneName != "" {
		question = "Open " + s.targetPaneName + " chat with:"
	}
	description := s.com.Styles.Dialog.Quit.Content.Render(
		descriptionText,
	)
	buttons := common.ButtonGroup(s.com.Styles, []common.ButtonOpts{
		{Text: "New Session", Selected: s.selectedNewSession, Padding: 2},
		{Text: "Sessions", Selected: !s.selectedNewSession, Padding: 2},
	}, " ")
	content := s.com.Styles.Dialog.Quit.Content.Render(
		lipgloss.JoinVertical(
			lipgloss.Center,
			question,
			"",
			description,
			"",
			buttons,
		),
	)
	view := s.com.Styles.Dialog.Quit.Frame.Render(content)
	DrawCenter(scr, area, view)
	return nil
}
