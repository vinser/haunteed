package over

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PlayAgainMsg is a message sent when the user chooses to play again.
type PlayAgainMsg struct{}

func playAgainCmd() tea.Cmd {
	return func() tea.Msg {
		return PlayAgainMsg{}
	}
}

// QuitGameMsg is a message sent when the user chooses to quit from the game over screen.
type QuitGameMsg struct{}

func quitGameCmd() tea.Cmd {
	return func() tea.Msg {
		return QuitGameMsg{}
	}
}

type Model struct {
	mode      string
	score     int
	highScore int
	sb        *strings.Builder
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
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "a":
			return m, playAgainCmd()
		case "q":
			return m, quitGameCmd()
		}
	}
	return m, nil
}

var (
	buttonStyle = lipgloss.NewStyle().Padding(0, 2).Margin(1)
)

func (m Model) View() string {
	m.sb.Reset()
	m.sb.WriteString("\nGame Over!\n\n")
	if m.score > m.highScore {
		m.sb.WriteString(fmt.Sprintf("!!! New %s High Score: ", m.mode))
	} else {
		m.sb.WriteString(fmt.Sprintf("Your %s Score: ", m.mode))
	}
	m.sb.WriteString(fmt.Sprintf("%d\n\n", m.score))

	m.sb.WriteString("\n\na — play again, q — quit\n")
	return m.sb.String()
}
