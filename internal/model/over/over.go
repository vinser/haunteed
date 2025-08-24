package over

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vinser/haunteed/internal/render"
	"github.com/vinser/haunteed/internal/state"
	"github.com/vinser/haunteed/internal/style"
)

type status int

const (
	statusIdle status = iota
	statusEntering
)

type Model struct {
	width      int
	height     int
	termWidth  int
	termHeight int

	status     status
	mode       string
	score      int
	highScores []state.HighScore
	textInput  textinput.Model
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

// SaveHighScoreMsg is a message sent when the user has entered their name for a new high score.
type SaveHighScoreMsg struct {
	Nick string
}

func saveHighScoreCmd(nick string) tea.Cmd {
	return func() tea.Msg {
		return SaveHighScoreMsg{Nick: nick}
	}
}

func New(mode string, score int, highScores []state.HighScore, width, height int) Model {
	if width < lipgloss.Width(footer) {
		width = lipgloss.Width(footer)
	}

	ti := textinput.New()
	ti.Prompt = "Nickname: "
	ti.Placeholder = "Enter Your Nickname"
	ti.CharLimit = 26
	ti.Width = 30

	leftAlign := lipgloss.NewStyle().Align(lipgloss.Left)
	ti.PromptStyle = leftAlign
	ti.TextStyle = leftAlign
	ti.PlaceholderStyle = leftAlign

	status := statusIdle
	// Check if the current score is high enough to make the list
	if len(highScores) < 5 || score > highScores[len(highScores)-1].Score {
		status = statusEntering
		ti.Focus()
	}

	return Model{
		width:      width,
		height:     height,
		status:     status,
		mode:       mode,
		score:      score,
		highScores: highScores,
		textInput:  ti,
	}
}

func (m *Model) SetSize(width, height int) {
	m.termWidth = width
	m.termHeight = height
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.status == statusEntering {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyEnter:
				// Save the score and switch to idle status
				nick := m.textInput.Value()
				m.status = statusIdle
				return m, saveHighScoreCmd(nick)
			case tea.KeyEsc:
				// Cancel entering, save with default name
				m.status = statusIdle
				return m, saveHighScoreCmd("")
			}
		}
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	// Idle status
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
	if m.status == statusEntering {
		var input []string
		input = append(input, style.HighScore.Render(fmt.Sprintf("New %s High score: %d !!!", m.mode, m.score)))

		input = append(input, "") // Add a blank line
		// blockStyle := lipgloss.NewStyle().Inline(false).Align(lipgloss.Left)
		// textInputLine := blockStyle.Render(m.textInput.View())
		// input = append(input, textInputLine)
		input = append(input, m.textInput.View())

		input = append(input, "") // Add a blank line
		input = append(input, "(press Enter to save, Esc to cancel)")

		return lipgloss.JoinVertical(lipgloss.Left, input...)
	}

	var content []string

	if len(m.highScores) == 0 || m.score > m.highScores[len(m.highScores)-1].Score {
		content = append(content, style.HighScore.Render(fmt.Sprintf("New %s High score: %d !!!", m.mode, m.score)))
	} else {
		content = append(content, fmt.Sprintf("Your %s score: %d", m.mode, m.score))
	}

	content = append(content, "") // Add a blank line
	content = append(content, "High Scores:")

	listFormat := fmt.Sprintf("%%d. %%%dd — %%s", calcDidgits(m.highScores))
	for i, hs := range m.highScores {
		content = append(content, fmt.Sprintf(listFormat, i+1, hs.Score, hs.Nick))
	}

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

func calcDidgits(hs []state.HighScore) int {
	digits := 0
	for _, hs := range hs {
		if len(strconv.Itoa(hs.Score)) > digits {
			digits = len(strconv.Itoa(hs.Score))
		}
	}
	return digits
}

func (m Model) Score() int {
	return m.score
}
