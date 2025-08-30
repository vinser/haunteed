package motd

import (
	"encoding/json"
	"math/rand"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vinser/haunteed/internal/embeddata"
)

type Model struct {
	msgs       []string
	style      lipgloss.Style
	frameWidth int
	repeats    int
	interval   time.Duration

	current    string
	offset     int
	doneCount  int
	lastShown  time.Time
	termWidth  int
	termHeight int
	rng        *rand.Rand
}

type TickMsg struct{}

func Tick() tea.Cmd {
	return tea.Tick(time.Millisecond*200, func(time.Time) tea.Msg {
		return TickMsg{}
	})
}

type motdMessages struct {
	Tips []string `json:"tips"`
}

func New(frameWidth, repeats int, interval time.Duration) Model {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Read and parse MOTD messages
	var msgs []string
	motdBytes, err := embeddata.ReadMOTD()
	if err == nil {
		var motdData motdMessages
		if json.Unmarshal(motdBytes, &motdData) == nil {
			msgs = motdData.Tips
		}
	}
	if len(msgs) == 0 {
		msgs = []string{"Have a spooky day!"} // Fallback
	}

	style := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)

	m := Model{
		msgs:       msgs,
		style:      style,
		frameWidth: frameWidth,
		repeats:    repeats,
		interval:   interval,
		current:    msgs[rng.Intn(len(msgs))],
		lastShown:  time.Now(),
		rng:        rng,
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return Tick()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg.(type) {
	case TickMsg:
		if m.doneCount >= m.repeats {
			if time.Since(m.lastShown) >= m.interval {
				m.current = m.msgs[m.rng.Intn(len(m.msgs))]
				m.lastShown = time.Now()
				m.doneCount = 0
				m.offset = 0
			}
		} else {
			m.offset++
			fullLength := len(m.current) + m.frameWidth
			if m.offset >= fullLength {
				m.offset = 0
				m.doneCount++
			}
		}
		return m, Tick()
	}
	return m, nil
}

func (m Model) View() string {
	if m.frameWidth <= 0 {
		return ""
	}

	displayWidth := m.frameWidth
	spaces := strings.Repeat(" ", displayWidth)
	repeatText := spaces + m.current + spaces

	start := m.offset
	end := start + displayWidth
	if end > len(repeatText) {
		end = len(repeatText)
	}
	if start > len(repeatText) {
		start = len(repeatText)
	}
	visible := repeatText[start:end]

	return m.style.Render(visible)
}

func (m *Model) SetWidth(width int) {
	m.frameWidth = width
}
