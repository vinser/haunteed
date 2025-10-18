package next

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vinser/haunteed/internal/render"
)

const nextPeriod = 5 * time.Second

type Model struct {
	width      int
	height     int
	termWidth  int
	termHeight int

	index     int
	nextUntil time.Time
}

// TickMsg is a tick message for periodic updates.
type TickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// TimedoutMsg signals the end of the transition period.
type TimedoutMsg struct{}

func timedoutCmd() tea.Cmd {
	return func() tea.Msg {
		return TimedoutMsg{}
	}
}

func New(index, width, height int) Model {
	if width < lipgloss.Width(footer) {
		width = lipgloss.Width(footer)
	}
	return Model{
		width:  width,
		height: height,

		index:     index,
		nextUntil: time.Now().Add(nextPeriod),
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
	switch msg.(type) {
	case tea.KeyMsg:
		// Ignore all keyboard events during the transition to prevent input queue buildup.
		return m, nil
	case TickMsg:
		if time.Now().After(m.nextUntil) {
			// Send TimedoutMsg to return to gameplay, relying on input queue reset.
			return m, timedoutCmd()
		}
		return m, tick()
	}
	return m, tick()
}

const footer = ""

func (m Model) View() string {
	flash := ""
	if (time.Now().UnixNano()/int64(time.Millisecond)/500)%2 == 0 {
		flash = fmt.Sprintf("Going to Floor # %d", m.index)
	}
	return render.Page(flash, m.renderContent(), footer, m.width, m.height, m.termWidth, m.termHeight)
}

func (m Model) renderContent() string {
	return "\nGet ready...\n"
}
