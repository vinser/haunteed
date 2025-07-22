package over

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	mode      string
	score     int
	highScore int
	sb        *strings.Builder
}

// TickMsg is a tick message.
type TickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func New(mode string, score int, highScore int) Model {
	return Model{
		mode:      mode,
		score:     score,
		highScore: highScore,
		sb:        &strings.Builder{},
	}
}

func (m Model) Init() tea.Cmd {
	return tick()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, tea.Quit
}

func (m Model) View() string {
	m.sb.Reset()
	m.sb.WriteString("\nGame Over!\n")
	if m.score > m.highScore {
		m.sb.WriteString(fmt.Sprintf("!!! New %s High Score: ", m.mode))
	} else {
		m.sb.WriteString(fmt.Sprintf("Your %s Score: ", m.mode))
	}
	m.sb.WriteString(fmt.Sprintf("%d\n", m.score))
	return m.sb.String()
}
