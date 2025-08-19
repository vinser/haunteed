package quit

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const quitPeriod = 5 * time.Second

type Model struct {
	quitUntil time.Time
}

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

func New() Model {
	return Model{
		quitUntil: time.Now().Add(quitPeriod),
	}
}

func (m Model) Init() tea.Cmd {
	return tick()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if time.Now().After(m.quitUntil) {
		return m, timedoutCmd()
	}
	return m, tick()
}

func (m Model) View() string {
	return "\nIt's a pity you gave up :(\nBye!\n"
}
