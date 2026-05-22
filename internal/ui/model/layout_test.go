package model

import (
	"strconv"
	"strings"
	"testing"

	"charm.land/bubbles/v2/textarea"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/session"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/ui/chat"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/ui/common"
)

// testMessageItem is a minimal chat item used to populate the chat list
// without pulling in full message rendering machinery.
type testMessageItem struct {
	id   string
	text string
}

func (m testMessageItem) ID() string           { return m.id }
func (m testMessageItem) Render(int) string    { return m.text }
func (m testMessageItem) RawRender(int) string { return m.text }
func (m testMessageItem) Version() uint64      { return 0 }
func (m testMessageItem) Finished() bool       { return true }

var _ chat.MessageItem = testMessageItem{}

// newTestUI builds a focused uiChat model with dynamic textarea sizing enabled.
// It intentionally keeps dependencies minimal so layout behavior can be tested
// in isolation.
func newTestUI() *UI {
	com := common.DefaultCommon(nil)

	ta := textarea.New()
	ta.SetStyles(com.Styles.Editor.Textarea)
	ta.ShowLineNumbers = false
	ta.CharLimit = -1
	ta.SetVirtualCursor(false)
	ta.DynamicHeight = true
	ta.MinHeight = TextareaMinHeight
	ta.MaxHeight = TextareaMaxHeight
	ta.Focus()

	u := &UI{
		com:      com,
		status:   NewStatus(com, nil),
		chat:     NewChat(com),
		textarea: ta,
		state:    uiChat,
		focus:    uiFocusEditor,
		width:    140,
		height:   45,
	}

	return u
}

func TestUpdateLayoutAndSize_EditorGrowthShrinksChat(t *testing.T) {
	t.Parallel()

	// Baseline layout at min textarea height.
	u := newTestUI()
	u.updateLayoutAndSize()

	initialEditorHeight := u.layout.editor.Dy()
	initialChatHeight := u.layout.main.Dy()

	// Increase textarea content enough to trigger growth, then run the
	// same resize hook used in the real update path.
	prevHeight := u.textarea.Height()
	u.textarea.SetValue(strings.Repeat("line\n", 8))
	u.textarea.MoveToEnd()
	_ = u.handleTextareaHeightChange(prevHeight)

	if got := u.layout.editor.Dy(); got <= initialEditorHeight {
		t.Fatalf("expected editor to grow: got %d, want > %d", got, initialEditorHeight)
	}

	if got := u.layout.main.Dy(); got >= initialChatHeight {
		t.Fatalf("expected chat to shrink: got %d, want < %d", got, initialChatHeight)
	}
}

func TestHandleTextareaHeightChange_FollowModeStaysAtBottom(t *testing.T) {
	t.Parallel()

	// Use enough messages to make the chat scrollable so AtBottom/Follow
	// assertions are meaningful.
	u := newTestUI()

	msgs := make([]chat.MessageItem, 0, 60)
	for i := range 60 {
		msgs = append(msgs, testMessageItem{
			id:   "m-" + strconv.Itoa(i),
			text: "message " + strconv.Itoa(i),
		})
	}
	u.chat.SetMessages(msgs...)
	u.updateLayoutAndSize()

	// Enter follow mode and verify we're anchored at the bottom first.
	u.chat.ScrollToBottom()
	if !u.chat.AtBottom() {
		t.Fatal("expected chat to start at bottom")
	}

	// Grow the editor; follow mode should keep the chat pinned to the end
	// even as the chat viewport shrinks.
	prevHeight := u.textarea.Height()
	u.textarea.SetValue(strings.Repeat("line\n", 10))
	u.textarea.MoveToEnd()
	_ = u.handleTextareaHeightChange(prevHeight)

	if !u.chat.Follow() {
		t.Fatal("expected follow mode to remain enabled")
	}
	if !u.chat.AtBottom() {
		t.Fatal("expected chat to remain at bottom after editor resize in follow mode")
	}
}

func TestGenerateLayout_SplitModeRendersBothPanesInCompactMode(t *testing.T) {
	t.Parallel()

	u := newTestUI()
	u.splitMode = true
	ta := textarea.New()
	ta.SetStyles(u.com.Styles.Editor.Textarea)
	ta.ShowLineNumbers = false
	ta.CharLimit = -1
	ta.SetVirtualCursor(false)
	ta.DynamicHeight = true
	ta.MinHeight = TextareaMinHeight
	ta.MaxHeight = TextareaMaxHeight
	u.secondaryPane = &chatPane{
		chat:     NewChat(u.com),
		textarea: ta,
	}
	u.isCompact = true
	u.updateLayoutAndSize()

	if u.layout.main.Dx() <= 0 || u.layout.secondaryMain.Dx() <= 0 {
		t.Fatalf("expected both split chat panes to have width: left=%d right=%d", u.layout.main.Dx(), u.layout.secondaryMain.Dx())
	}
	if u.layout.editor.Dx() <= 0 || u.layout.secondaryEditor.Dx() <= 0 {
		t.Fatalf("expected both split editors to have width: left=%d right=%d", u.layout.editor.Dx(), u.layout.secondaryEditor.Dx())
	}
	if u.layout.secondaryMain.Min.X <= u.layout.main.Min.X {
		t.Fatalf("expected right split pane to be after left pane: left=%v right=%v", u.layout.main, u.layout.secondaryMain)
	}
}

func TestSplitChatWithNewSessionKeepsBlankPaneUntilSend(t *testing.T) {
	t.Parallel()

	u := newTestUI()
	u.state = uiLanding
	ta := textarea.New()
	ta.SetStyles(u.com.Styles.Editor.Textarea)
	u.secondaryPane = &chatPane{
		chat:     NewChat(u.com),
		textarea: ta,
	}

	cmd := u.splitChatWithNewSession(secondaryPane)
	if cmd == nil {
		t.Fatal("expected focus command")
	}
	if u.state != uiChat {
		t.Fatalf("expected split from landing to enter chat state, got %v", u.state)
	}
	if !u.splitMode || u.secondaryPane == nil {
		t.Fatal("expected split mode with a secondary pane")
	}
	if u.secondaryPane.session != nil {
		t.Fatal("expected blank split pane to avoid creating a persisted session")
	}
	if u.activePane != secondaryPane {
		t.Fatalf("expected blank right pane to be focused, got pane %d", u.activePane)
	}
}

func TestActiveSidebarPaneFollowsFocusedSplitPane(t *testing.T) {
	t.Parallel()

	u := newTestUI()
	u.session = &session.Session{ID: "left", Title: "Left Chat"}
	u.sessionFiles = []SessionFile{{Additions: 1}}
	u.secondaryPane = &chatPane{
		session:  &session.Session{ID: "right", Title: "Right Chat"},
		chat:     NewChat(u.com),
		textarea: textarea.New(),
	}
	u.secondaryPaneSessionFiles = []SessionFile{{Additions: 2}}
	u.splitMode = true

	u.activePane = primaryPane
	if got := u.activeSidebarPane().session.ID; got != "left" {
		t.Fatalf("expected left sidebar session, got %q", got)
	}
	if got := u.activeSidebarSessionFiles()[0].Additions; got != 1 {
		t.Fatalf("expected left sidebar files, got additions=%d", got)
	}

	u.activePane = secondaryPane
	if got := u.activeSidebarPane().session.ID; got != "right" {
		t.Fatalf("expected right sidebar session, got %q", got)
	}
	if got := u.activeSidebarSessionFiles()[0].Additions; got != 2 {
		t.Fatalf("expected right sidebar files, got additions=%d", got)
	}
}
