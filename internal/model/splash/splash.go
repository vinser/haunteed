package splash

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vinser/haunteed/internal/style"
)

var haunteedOpen = `
███   ███        
███▄▄▄███         
███▀▀▀███HAUNTEED 
▀▀▀   ▀▀▀▄▄▄▄▄▄▄▄▄
 HAUNTEED▀▀▀███▀▀▀
            ███   
            ███   
`

var haunteedClosed = `
   ███   ███         
   ███▄▄▄███         
   ███▀▀▀███NTEED    
   ▀▀▀▄▄▄███▄▄▄
 HAUNT▀▀▀███▀▀▀
         ███   
         ███   
`

var haunteedFront = `
                  
                  
         HAUNTEED 
██████████████████
 HAUNTEED         
                  
                  
`

var (
	haunteedOpenLines   = strings.Split(haunteedOpen, "\n")
	haunteedClosedLines = strings.Split(haunteedClosed, "\n")
	haunteedFrontLines  = strings.Split(haunteedFront, "\n")
)

const (
	spriteWidth  = 18 // Width of the widest part of the sprite
	spriteHeight = 7  // Height of the sprite in lines
)

type Model struct {
	width  int
	height int

	// State
	pos        int
	open       bool
	pauseUntil time.Time
	dots       []bool

	// Reusable buffers to reduce allocations and improve performance.
	grid [][]rune
	sb   *strings.Builder
}

type MoveMsg struct{}

func moveCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return MoveMsg{}
	})
}

type ChewMsg struct{}

func chewCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return ChewMsg{}
	})
}

type MakeSettingsMsg struct{}

func makeSettingsCmd() tea.Cmd {
	return func() tea.Msg {
		return MakeSettingsMsg{}
	}
}

type TimedoutMsg struct{}

func timedoutCmd() tea.Cmd {
	return func() tea.Msg {
		return TimedoutMsg{}
	}
}

func New(width, height int) Model {
	dots := make([]bool, width)
	for i := range dots {
		dots[i] = true
	}

	grid := make([][]rune, height)
	for i := range grid {
		grid[i] = make([]rune, width)
	}

	return Model{
		width:  width,
		height: height,
		pos:    -spriteWidth,
		open:   true,
		dots:   dots,
		grid:   grid,
		sb:     &strings.Builder{},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(moveCmd(), chewCmd())
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case MoveMsg:
		now := time.Now()
		if !m.pauseUntil.IsZero() {
			if now.Before(m.pauseUntil) {
				return m, moveCmd()
			}
			m.pauseUntil = time.Time{}
			m.open = true
			m.pos++
			return m, moveCmd()
		}
		// Pause when the sprite reaches the center of the screen
		if m.pos == (m.width/2 - spriteWidth/2) {
			m.pauseUntil = now.Add(2 * time.Second)
			return m, moveCmd()
		}
		m.pos++
		// "Eat" a dot when the mouth of the sprite passes over it
		mouthPos := m.pos + spriteWidth - 1
		if mouthPos >= 0 && mouthPos < len(m.dots) {
			m.dots[mouthPos] = false
		}
		if m.pos > m.width {
			return m, timedoutCmd()
		}
		return m, moveCmd()

	case ChewMsg:
		if m.pauseUntil.IsZero() {
			m.open = !m.open
		}
		return m, chewCmd()

	case tea.KeyMsg:
		switch msg.String() {
		case "s":
			return m, makeSettingsCmd()
		case "enter", "esc", "space":
			return m, timedoutCmd()
		}
	}
	return m, nil
}

func (m Model) View() string {
	// Clear the reusable grid
	for i := range m.grid {
		for j := range m.grid[i] {
			m.grid[i][j] = ' '
		}
	}

	// Center dots and sprite vertically
	spriteY := (m.height - spriteHeight) / 2
	dotY := spriteY + 4 // Align dots with the sprite's "mouth" line
	for i := 0; i < m.width; i += 4 {
		if m.dots != nil && i < len(m.dots) && m.dots[i] && dotY >= 0 && dotY < m.height {
			m.grid[dotY][i] = '●'
		}
	}

	var spriteLines []string
	switch {
	case !m.pauseUntil.IsZero():
		spriteLines = haunteedFrontLines
	case m.open:
		spriteLines = haunteedOpenLines
	default:
		spriteLines = haunteedClosedLines
	}

	for i, line := range spriteLines {
		y := spriteY + i
		if y < 0 || y >= m.height {
			continue
		}
		for x, r := range []rune(line) {
			sx := m.pos + x
			if sx >= 0 && sx < m.width {
				m.grid[y][sx] = r
			}
		}
	}

	// Reset and reuse the string builder
	m.sb.Reset()
	for _, row := range m.grid {
		for _, r := range row {
			switch r {
			case '●':
				m.sb.WriteString(style.SplashDot.Render(string(r)))
			case ' ', 0:
				m.sb.WriteRune(' ')
			default:
				m.sb.WriteString(style.SplashHaunteed.Render(string(r)))
			}
		}
		m.sb.WriteRune('\n')
	}

	m.sb.WriteString("\ns — settings, m — mute, space — skip, q — quit\n")

	return m.sb.String()
}
