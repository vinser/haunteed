package splash

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vinser/haunteed/internal/state"
	"github.com/vinser/haunteed/internal/style"
)

const (
	haunteedOpen = `
███   ███        
███▄▄▄███         
███▀▀▀███HAUNTEED 
▀▀▀   ▀▀▀▄▄▄▄▄▄▄▄▄
 HAUNTEED▀▀▀███▀▀▀
            ███   
            ███   
`
	haunteedClosed = `
   ███   ███         
   ███▄▄▄███         
   ███▀▀▀███NTEED    
   ▀▀▀▄▄▄███▄▄▄
 HAUNT▀▀▀███▀▀▀
         ███   
         ███   
`
	haunteedFront = `
                  
                  
         HAUNTEED 
██████████████████
 HAUNTEED         
                  
                  
`
	curly = `
▄█████▄           
██                
██▄▄▄▄▄ CURLY     
 ▀▀▀▀▀ ▄▄▄▄▄▄ 
 CURLY ██▀▀▀██ 
       ██████▀ 
       ██  ▀█▄
`
	lofty = `
██              
██               
██▄▄▄▄ LOFTY   
▀▀▀▀▀▀ ▄▄▄▄▄▄  
 LOFTY ██▀▀▀▀  
       █████   
       ██      
`
	fluffy = `
██████          
██▄▄▄            
██▀▀▀ FLUFFY  
▀▀     ▄▄▄▄▄▄
FLUFFY ██▀▀▀▀  
       █████
       ██      
`

	virty = `
██   ██        
██   ██          
 ██▄██  VIRTY   
  ▀█▀  ▄▄▄▄▄▄  
 VIRTY ▀▀██▀▀  
         ██    
         ██    
`
)

const (
	spriteWidth  = 18
	spriteHeight = 7

	middlePause      = 3 * time.Second
	moveTickDuration = 100 * time.Millisecond
	chewTickDuration = 500 * time.Millisecond
)

var ghostSprites = []string{curly, lofty, fluffy, virty}

type movingGhost struct {
	index      int
	pos        int
	paused     bool
	pauseUntil time.Time
}

type Model struct {
	state *state.State

	width  int
	height int

	pos        int
	open       bool
	pauseUntil time.Time
	dots       []bool

	ghostIndex    int
	ghostPos      int
	showGhosts    bool
	movingGhosts  []movingGhost // ghosts currently moving
	ghostsStarted int

	grid           [][]rune // grid is the display grid for the splash screen
	ghostColorGrid [][]int  // parallel grid for ghost color indices, -1 means no ghost
	sb             *strings.Builder
}

type MoveMsg struct{}

func moveCmd() tea.Cmd {
	return tea.Tick(moveTickDuration, func(t time.Time) tea.Msg {
		return MoveMsg{}
	})
}

type ChewMsg struct{}

func chewCmd() tea.Cmd {
	return tea.Tick(chewTickDuration, func(t time.Time) tea.Msg {
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

func New(state *state.State, width, height int) Model {
	dots := make([]bool, width)
	for i := range dots {
		dots[i] = true
	}

	grid := make([][]rune, height)
	ghostColorGrid := make([][]int, height)
	for i := range grid {
		grid[i] = make([]rune, width)
		ghostColorGrid[i] = make([]int, width)
		for j := range ghostColorGrid[i] {
			ghostColorGrid[i][j] = -1
		}
	}

	return Model{
		state:          state,
		width:          width,
		height:         height,
		pos:            -spriteWidth,
		open:           true,
		dots:           dots,
		grid:           grid,
		ghostColorGrid: ghostColorGrid,
		sb:             &strings.Builder{},
		ghostIndex:     -1,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(moveCmd(), chewCmd())
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case MoveMsg:
		if m.showGhosts {
			return m.updateGhosts()
		}
		return m.updateHaunteed()
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

// --- Sub-functions for Update ---

func (m Model) updateHaunteed() (Model, tea.Cmd) {
	now := time.Now()
	if !m.pauseUntil.IsZero() {
		if now.Before(m.pauseUntil) {
			return m, moveCmd()
		}
		m.pauseUntil = time.Time{}
		m.open = true
		m.pos++
		// Eat dot after pause
		mouthPos := m.pos + spriteWidth - 1
		if mouthPos >= 0 && mouthPos < len(m.dots) {
			m.dots[mouthPos] = false
		}
		return m, moveCmd()
	}
	if m.pos == (m.width/2 - spriteWidth/2) {
		m.pauseUntil = now.Add(middlePause)
		return m, moveCmd()
	}
	m.pos++
	mouthPos := m.pos + spriteWidth - 1
	if mouthPos >= 0 && mouthPos < len(m.dots) {
		m.dots[mouthPos] = false
	}
	if m.pos > m.width {
		m.showGhosts = true
		m.ghostIndex = -1
		m.ghostPos = -spriteWidth
		return m, moveCmd()
	}
	return m, moveCmd()
}

func (m Model) updateGhosts() (Model, tea.Cmd) {
	centerX := m.width/2 - spriteWidth/2

	// Start the first ghost if needed
	if len(m.movingGhosts) == 0 && m.ghostsStarted == 0 {
		m.movingGhosts = append(m.movingGhosts, movingGhost{index: 0, pos: -spriteWidth})
		m.ghostsStarted = 1
	}

	// Start the next ghost if the last started ghost reached center and there are more ghosts
	if len(m.movingGhosts) > 0 {
		last := &m.movingGhosts[len(m.movingGhosts)-1]
		if last.pos == centerX && !last.paused {
			last.paused = true
			last.pauseUntil = time.Now().Add(2 * time.Second)
		}
		if last.pos == centerX && last.paused && time.Now().After(last.pauseUntil) {
			last.paused = false
			if m.ghostsStarted < len(ghostSprites) {
				m.movingGhosts = append(m.movingGhosts, movingGhost{index: m.ghostsStarted, pos: -spriteWidth})
				m.ghostsStarted++
			}
		}
	}

	// Move all ghosts (only if not paused)
	for i := range m.movingGhosts {
		if !m.movingGhosts[i].paused {
			m.movingGhosts[i].pos++
		}
	}

	// Remove ghosts that have fully exited the right border
	remaining := m.movingGhosts[:0]
	for _, g := range m.movingGhosts {
		if g.pos < m.width {
			remaining = append(remaining, g)
		}
	}
	m.movingGhosts = remaining

	// If all ghosts have exited, finish splash
	if len(m.movingGhosts) == 0 && m.ghostsStarted == len(ghostSprites) {
		return m, timedoutCmd()
	}

	// Mark ghostIndex for coloring in View
	if len(m.movingGhosts) > 0 {
		m.ghostIndex = m.movingGhosts[len(m.movingGhosts)-1].index
	}

	return m, moveCmd()
}

// --- View ---

func (m Model) View() string {
	m.clearGrid()
	m.drawDots()
	if m.showGhosts && len(m.movingGhosts) > 0 {
		for _, g := range m.movingGhosts {
			m.drawGhost(g.index, g.pos)
		}
	} else {
		m.drawHaunteed()
	}
	return m.renderGrid()
}

func (m *Model) clearGrid() {
	for i := range m.grid {
		for j := range m.grid[i] {
			m.grid[i][j] = ' '
			m.ghostColorGrid[i][j] = -1
		}
	}
}

func (m *Model) ghostY(ghostIdx int) int {
	switch ghostIdx {
	case 0: // Curly (top)
		return 0
	case 3: // Virty (bottom)
		return m.height - spriteHeight - 1
	default: // Lofty, Fluffy (evenly spaced)
		// Spread evenly between top and bottom
		steps := len(ghostSprites) - 1
		return (ghostIdx * (m.height - spriteHeight)) / steps
	}
}

func (m *Model) drawDots() {
	spriteY := (m.height - spriteHeight) / 2
	dotY := spriteY + spriteHeight/2 + 1 // Align dots with the sprite's "mouth" line
	for i := 0; i < m.width; i += 4 {
		if m.dots != nil && i < len(m.dots) && m.dots[i] && dotY >= 0 && dotY < m.height {
			m.grid[dotY][i] = '●'
		}
	}
}

func (m *Model) drawGhost(ghostIdx, ghostPos int) {
	spriteY := m.ghostY(ghostIdx)
	spriteLines := strings.Split(ghostSprites[ghostIdx], "\n")
	for i, line := range spriteLines {
		y := spriteY + i
		if y < 0 || y >= m.height {
			continue
		}
		for x, r := range []rune(line) {
			sx := ghostPos + x
			if sx >= 0 && sx < m.width && r != ' ' {
				m.grid[y][sx] = r
				m.ghostColorGrid[y][sx] = ghostIdx
			}
		}
	}
}

func (m *Model) drawHaunteed() {
	spriteY := (m.height - spriteHeight) / 2
	var spriteLines []string
	switch {
	case !m.pauseUntil.IsZero():
		spriteLines = strings.Split(haunteedFront, "\n")
	case m.open:
		spriteLines = strings.Split(haunteedOpen, "\n")
	default:
		spriteLines = strings.Split(haunteedClosed, "\n")
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
}

func (m *Model) renderGrid() string {
	m.sb.Reset()
	for y, row := range m.grid {
		for x, r := range row {
			switch r {
			case '●':
				m.sb.WriteString(style.SplashDot.Render(string(r)))
			case ' ', 0:
				m.sb.WriteRune(' ')
			default:
				if m.showGhosts && m.ghostColorGrid[y][x] >= 0 && m.ghostColorGrid[y][x] < len(style.SplashGhosts) {
					idx := m.ghostColorGrid[y][x]
					m.sb.WriteString(style.SplashGhosts[idx].Render(string(r)))
				} else {
					m.sb.WriteString(style.SplashHaunteed.Render(string(r)))
				}
			}
		}
		m.sb.WriteRune('\n')
	}
	m.sb.WriteString("s — settings, m — mute, space — skip, q — quit\n")
	return m.sb.String()
}
