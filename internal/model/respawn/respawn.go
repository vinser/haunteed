package respawn

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const respawnPeriod = 3 * time.Second

type Model struct {
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

func New(lives int) Model {
	return Model{
		lives:        lives,
		respawnUntil: time.Now().Add(respawnPeriod),
	}
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

func (m Model) View() string {
	var flash string
	if (time.Now().UnixNano()/int64(time.Millisecond)/500)%2 == 0 {
		flash = "Respawning..."
	}
	return fmt.Sprintf(
		"\n%s\nLives: %d\nGet ready to continue\n", flash, m.lives)
}
