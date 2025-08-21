package over

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vinser/haunteed/internal/render"
	"github.com/vinser/haunteed/internal/style"
)

type Model struct {
	width      int
	height     int
	termWidth  int
	termHeight int

	mode      string
	score     int
	highScore int
}

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

func New(mode string, score, highScore, width, height int) Model {
	if width < lipgloss.Width(footer) {
		width = lipgloss.Width(footer)
	}
	return Model{
		width:  width,
		height: height,

		mode:      mode,
		score:     score,
		highScore: highScore,
	}
}

func (m *Model) SetSize(width, height int) {
	m.termWidth = width
	m.termHeight = height
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

const footer = "a — play again, q — quit"

func (m Model) View() string {
	return render.Page("Game Over!", m.renderContent(), footer, m.width, m.height, m.termWidth, m.termHeight)
}

func (m Model) renderContent() string {
	if m.score > m.highScore {
		return style.HighScore.Render(fmt.Sprintf("New %s High score: %d !!!", m.mode, m.score))
	} else {
		return fmt.Sprintf("Your %s score: %d", m.mode, m.score)
	}
}
