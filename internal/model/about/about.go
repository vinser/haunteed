package about

import (
	"log"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/vinser/haunteed/internal/embeddata"
	"github.com/vinser/haunteed/internal/render"
	"github.com/vinser/haunteed/internal/state"
)

const (
	selectedMode = iota
	selectedCrazyNight
	selectedSpriteSize
	selectedMute
	selectedReset
)

type Model struct {
	width       int
	height      int
	startHeight int
	termWidth   int
	termHeight  int

	viewport viewport.Model
}

type CloseAboutMsg struct{}

func closeAboutCmd() tea.Cmd {
	return func() tea.Msg {
		return CloseAboutMsg{}
	}
}

func New(state *state.State, width, height int) Model {
	if width < lipgloss.Width(footer) {
		width = lipgloss.Width(footer)
	}
	bytes, err := embeddata.ReadAboutMD()
	if err != nil {
		log.Fatal(err)
	}

	vp := viewport.New(width, height)
	vp.Style = lipgloss.NewStyle()
	const glamourGutter = 2
	glam := glamContent(string(bytes), width, vp.Style.GetHorizontalFrameSize(), glamourGutter)
	vp.SetContent(glam)

	return Model{
		width:       width,
		height:      height,
		startHeight: height,

		viewport: vp,
	}
}

func (m *Model) SetSize(width, height int) {
	m.termWidth = width
	m.termHeight = height
	if m.startHeight > m.termHeight-5 {
		m.height = m.termHeight
		m.viewport.Height = m.termHeight - 5
	} else {
		m.height = m.startHeight
		m.viewport.Height = m.startHeight

	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, closeAboutCmd()
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

const footer = "↑ ↓ — scroll, esc — back, q — quit"

func (m Model) View() string {
	return render.Page("About", m.viewport.View(), footer, m.width, m.height, m.termWidth, m.termHeight)
}

func glamContent(content string, width, frame, gutter int) string {
	renderWidth := width - frame - gutter
	r, err := glamour.NewTermRenderer(
		// glamour.WithAutoStyle(),
		glamour.WithStandardStyle("pink"),
		glamour.WithWordWrap(renderWidth),
	)
	if err != nil {
		return content //noop
	}
	str, err := r.Render(content)
	if err != nil {
		return content //noop
	}
	return str
}
