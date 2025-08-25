package respawn

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vinser/haunteed/internal/render"
)

const respawnPeriod = 5 * time.Second

type Model struct {
	width      int
	height     int
	termWidth  int
	termHeight int

	lives        int
	respawnUntil time.Time
}

// TickMsg is a tick message.
type TickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

type TimedoutMsg struct{}

func timedoutCmd() tea.Cmd {
	return func() tea.Msg {
		return TimedoutMsg{}
	}
}

func New(lives, width, height int) Model {
	if width < lipgloss.Width(footer) {
		width = lipgloss.Width(footer)
	}
	return Model{
		width:  width,
		height: height,

		lives:        lives,
		respawnUntil: time.Now().Add(respawnPeriod),
	}
}

func (m *Model) SetSize(width, height int) {
	m.termWidth = width
	m.termHeight = height
}

func (m Model) Init() tea.Cmd {
	return tick()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if time.Now().After(m.respawnUntil) {
		return m, timedoutCmd()
	}
	return m, tick()
}

const footer = ""

func (m Model) View() string {
	flash := ""
	if (time.Now().UnixNano()/int64(time.Millisecond)/500)%2 == 0 {
		flash = "Respawning..."
	}
	return render.Page(flash, m.renderContent(), footer, m.width, m.height, m.termWidth, m.termHeight)
}

func (m Model) renderContent() string {
	return fmt.Sprintf("Lives left: %d\nGet ready to continue", m.lives)
}
