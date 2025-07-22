package intro

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const introPeriod = 3 * time.Second

type Model struct {
	index      int
	introUntil time.Time
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

func New(index int) Model {
	return Model{
		index:      index,
		introUntil: time.Now().Add(introPeriod),
	}
}

func (m Model) Init() tea.Cmd {
	return tick()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if time.Now().After(m.introUntil) {
		return m, timedoutCmd()
	}
	return m, tick()
}

func (m Model) View() string {
	var flash string
	if (time.Now().UnixNano()/int64(time.Millisecond)/500)%2 == 0 {
		flash = fmt.Sprintf("Going to Next Floor %d", m.index)
	}
	return fmt.Sprintf("\n%s\nGet ready...\n", flash)
}
