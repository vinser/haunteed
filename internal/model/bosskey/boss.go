package bosskey

import (
	"encoding/json"
	"math/rand"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vinser/haunteed/internal/embeddata"
)

// Model describe app states
type Model struct {
	msgs      []string
	bossLines []string
	rng       *rand.Rand
	width     int
	height    int
}

// TickMsg refreshes boss key
type TickMsg time.Time

type bossMessages struct {
	Tips []string `json:"tips"`
}

func New() Model {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Read and parse MOTD messages
	var msgs []string
	bossBytes, err := embeddata.ReadBoss()
	if err == nil {
		var bossData bossMessages
		if json.Unmarshal(bossBytes, &bossData) == nil {
			msgs = bossData.Tips
		}
	}
	if len(msgs) == 0 {
		msgs = []string{"[WARNING] If it’s quiet… maybe too quiet."} // Fallback
	}

	return Model{
		msgs:      msgs,
		bossLines: randomLines(msgs...),
		rng:       rng,
	}
}

// Init starts the 1-st tick
func (m Model) Init() tea.Cmd {
	return Tick()
}

// Tick every 10 second
func Tick() tea.Cmd {
	return tea.Tick(10*time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Update
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg.(type) {
	case TickMsg:
		m.bossLines = randomLines(m.msgs...) // update lines
		return m, Tick()
	}

	return m, nil
}

// View renders the screen
func (m Model) View() string {
	header := "Haunteed Night Shift Monitor\n============================\n\n"
	events := ""
	for _, line := range m.bossLines {
		events += line + "\n"
	}
	histogram := buildHistogram(m.bossLines)
	footer := "\n[Press 'b' to return]"
	content := lipgloss.JoinVertical(lipgloss.Left, header, events, histogram, footer)

	style := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Background(lipgloss.Color("0")).
		Foreground(lipgloss.Color("10")).
		Padding(0, 0, 0, 3)
		// Align(lipgloss.Center)

	return style.Render(content)
}

// randomLines
func randomLines(lines ...string) []string {
	// choose 5 random lines
	rand.Shuffle(len(lines), func(i, j int) { lines[i], lines[j] = lines[j], lines[i] })
	return lines[:5]
}

// buildHistogram builds a 5-row histogram with columns aligned so that
// the first █ of each column is exactly under the third letter of its label.
func buildHistogram(bossLines []string) string {
	categories := []string{"INFO", "WARN", "ERROR", "DEBUG", "NOTE", "HUMOR", "HINT"}
	histHeight := 5

	// Count occurrences of each category in current bossLines
	counts := map[string]int{}
	for _, cat := range categories {
		counts[cat] = 0
	}
	for _, line := range bossLines {
		for _, cat := range categories {
			if len(line) >= len(cat)+2 && line[1:1+len(cat)] == cat {
				counts[cat]++
				break
			}
		}
	}

	// Precompute column positions: first █ above the third letter of label
	labelTemplate := "[INFO]  [WARN]  [ERROR] [DEBUG] [NOTE]  [HUMOR] [HINT]"
	colPositions := make(map[string]int)
	for _, cat := range categories {
		idx := 0
		if pos := indexThirdLetter(labelTemplate, cat); pos >= 0 {
			colPositions[cat] = pos
		} else {
			colPositions[cat] = idx
		}
	}

	// Prepare histogram lines
	lines := make([][]rune, histHeight)
	for i := range lines {
		lines[i] = []rune(labelTemplate) // start with spaces same length as labels
		for j := range lines[i] {
			lines[i][j] = ' '
		}
	}

	// Fill columns with █ according to counts
	for _, cat := range categories {
		pos := colPositions[cat]
		for row := histHeight - counts[cat]; row < histHeight; row++ {
			for i := 0; i < 3; i++ { // width = 3
				lines[row][pos+i] = '█'
			}
		}
	}

	// Build output string
	out := ""
	for _, l := range lines {
		out += string(l) + "\n"
	}
	out += labelTemplate + "\n"
	return out
}

// indexThirdLetter finds the index of the third letter of the category label
func indexThirdLetter(template, cat string) int {
	idx := 0
	for i := 0; i+len(cat)+2 <= len(template); i++ {
		if template[i] == '[' && template[i+1:i+1+len(cat)] == cat {
			return i + 2 // third letter position
		}
	}
	return idx
}
