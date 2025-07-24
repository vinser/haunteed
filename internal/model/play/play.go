package play

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vinser/haunteed/internal/dweller"
	floor "github.com/vinser/haunteed/internal/floor"
	"github.com/vinser/haunteed/internal/score"
	"github.com/vinser/haunteed/internal/state"
	"github.com/vinser/haunteed/internal/style"
)

const (
	frightenedPeriod = 10 * time.Second
)

type Model struct {
	state             *state.State
	floor             *floor.Floor
	score             *score.Score
	haunteed          *dweller.Haunteed
	ghosts            []*dweller.Ghost
	lastKeyMsg        tea.KeyMsg
	lastKeyTime       time.Time
	lastGhostMove     time.Time
	powerMode         bool
	powerModeUntil    time.Time
	ghostController   *dweller.GhostController
	ghostTickInterval time.Duration
	justArrived       bool // To prevent immediate floor transition
	sb                *strings.Builder
	fullVisibility    bool
}

// GhostTickMsg is a tick message.
// It is used to trigger ghost movement updates at regular intervals.
// This message is sent by the tickGhosts function to the play model.
// The tick interval is defined by the ghostTickInterval in the play model.
type GhostTickMsg time.Time

func tickGhosts() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return GhostTickMsg(t)
	})
}

// NextFloorMsg is a message to transition to the next floor.
// It contains the index of the next floor.
// This is used to handle the transition logic in the main application.
// The floor index is used to retrieve the next floor from the cache or create it if it doesn't exist.
type NextFloorMsg struct {
	Floor int
}

func nextFloorCmd(floor int) tea.Cmd {
	return func() tea.Msg {
		return NextFloorMsg{
			Floor: floor,
		}
	}
}

// PrevFloorMsg is a message to transition to the previous floor.
// It contains the index of the previous floor.
// This is used to handle the transition logic in the main application.
// The floor index is used to retrieve the previous floor from the cache or create it if it doesn't exist.
type PrevFloorMsg struct {
	Floor int
}

func prevFloorCmd(floor int) tea.Cmd {
	return func() tea.Msg {
		return PrevFloorMsg{
			Floor: floor,
		}
	}
}

// GameOverMsg is a message sent when the game is over.
// This message is used to display the game over screen and handle any necessary cleanup or state updates.
// It contains the game mode, current score, and high score.
type GameOverMsg struct {
	Mode      string
	Score     int
	HighScore int
}

func gameOverCmd(mode string, score int, highScore int) tea.Cmd {
	return func() tea.Msg {
		return GameOverMsg{
			Mode:      mode,
			Score:     score,
			HighScore: highScore,
		}
	}
}

// RespawnMsg is a message sent when the haunteed respawns after losing a life.
type RespawnMsg struct {
	Lives int
}

func respawnCmd(lives int) tea.Cmd {
	return func() tea.Msg {
		return RespawnMsg{
			Lives: lives,
		}
	}
}

// VisibilityToggledMsg is a message sent when the visibility of a floor is toggled when the haunteed steps on a fuse.
type VisibilityToggledMsg struct {
	FloorIndex int
	IsVisible  bool
}

func toggleVisibilityCmd(floorIndex int, isVisible bool) tea.Cmd {
	return func() tea.Msg {
		return VisibilityToggledMsg{
			FloorIndex: floorIndex,
			IsVisible:  isVisible,
		}
	}
}

func New(s *state.State, f *floor.Floor, sc *score.Score, h *dweller.Haunteed, initialVisibility bool) Model {
	rng := rand.New(rand.NewSource(s.FloorSeeds[f.Index]))
	ghosts := dweller.PlaceGhosts(f.Index, s.SpriteSize, s.GameMode, f.Maze.Width(), f.Maze.Height(), f.Maze.DenWidth(), f.Maze.DenHeight(), rng)
	ghostTick := f.GhostTickInterval
	return Model{
		state:             s,
		floor:             f,
		score:             sc,
		haunteed:          h,
		ghosts:            ghosts,
		lastGhostMove:     time.Now(),
		powerMode:         false,
		powerModeUntil:    time.Now(),
		ghostController:   dweller.NewGhostController(),
		ghostTickInterval: ghostTick,
		justArrived:       true,
		sb:                &strings.Builder{},
		fullVisibility:    initialVisibility,
	}
}

func (m Model) Init() tea.Cmd {
	return tickGhosts()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// if msg == m.lastKeyMsg && time.Since(m.lastKeyTime) < 10*time.Millisecond {
		// 	return m, nil
		// }
		// m.lastKeyMsg = msg
		// m.lastKeyTime = time.Now()

		m.haunteed.HandleInput(msg)
		m.haunteed.Move(m.floor)

		pos := m.haunteed.Pos()
		tile := m.floor.EatItem(pos.X, pos.Y)

		switch tile {
		case floor.Dot:
			if m.state.GameMode == state.ModeNoisy {
				m.score.Add(20)
			} else {
				m.score.Add(10)
			}
		case floor.PowerPellet:
			m.score.Add(50)
			m.powerMode = true
			m.powerModeUntil = time.Now().Add(frightenedPeriod)
			m.ghostTickInterval = m.floor.GhostTickInterval * 2 // slow down ghosts
			for _, g := range m.ghosts {
				g.SetState(dweller.Frightened)
			}
		case floor.Fuse:
			m.fullVisibility = !m.fullVisibility
			return m, toggleVisibilityCmd(m.floor.Index, m.fullVisibility)
		case floor.Start:
			if !m.justArrived {
				return m, prevFloorCmd(m.floor.Index - 1)
			}
		case floor.End:
			if !m.justArrived {
				return m, nextFloorCmd(m.floor.Index + 1)
			}
		}

		if m.justArrived {
			m.justArrived = false
		}
	}

	// update power mode
	if m.powerMode && time.Now().After(m.powerModeUntil) {
		m.powerMode = false
		m.ghostTickInterval = m.floor.GhostTickInterval // reset ghost speed
		m.score.ResetGhostStreak()
		for _, g := range m.ghosts {
			if g.State() == dweller.Frightened {
				g.SetState(dweller.Chase)
			}
		}
	}

	switch msg.(type) {
	case GhostTickMsg:
		if time.Since(m.lastGhostMove) >= m.ghostTickInterval {
			m.ghostController.Update(m.ghosts)
			dweller.MoveGhosts(m.ghosts, m.floor, m.powerMode, m.haunteed.Pos(), m.haunteed.Dir())
			m.lastGhostMove = time.Now()
		}
	}

	// check haunteed collisions with ghosts
	htPos := m.haunteed.Pos()
	for _, g := range m.ghosts {
		if htPos == g.Pos() {
			switch g.State() {
			case dweller.Frightened: // eat the ghost
				m.score.AddGhostPoints()
				g.SetState(dweller.Eaten)
			case dweller.Chase: // lose a life
				m.haunteed.LoseLife()
				if m.haunteed.IsDead() { // game over
					score := m.score.Get()
					highScore := m.state.GetHighScore()
					if err := m.state.UpdateAndSave(m.floor.Index, score, m.floor.Seed); err != nil {
						log.Fatal(err)
					}
					return m, gameOverCmd(m.state.GameMode, score, highScore)
				}
				// enter respawn mode
				return m, respawnCmd(m.haunteed.Lives())
			}
		}
	}

	return m, tickGhosts()
}

func (m Model) Haunteed() *dweller.Haunteed {
	return m.haunteed
}

func (m Model) FloorIndex() int {
	return m.floor.Index
}

func (m Model) Score() *score.Score {
	return m.score
}

func (m Model) Lives() int {
	return m.haunteed.Lives()
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// manhattan returns the Manhattan distance between two points.
func manhattan(a, b dweller.Position) int {
	return abs(a.X-b.X) + abs(a.Y-b.Y)
}

func distance(a, b dweller.Position) int {
	dX := abs(a.X - b.X)
	dY := abs(a.Y - b.Y)

	return int(math.Sqrt(float64(dX*dX + dY*dY)))
}

const (
	minVisibilityRadius = 4
)

var (
	styleHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")) // bright green
)

// RenderMaze returns the complete screen output with game entities and stats.
func (m Model) View() string {
	m.sb.Reset()

	// Draw game header
	header := ""
	if m.state.GameMode != state.ModeCrazy {
		header = fmt.Sprintf("Mode: %s  Floor: %d\nScore: %d  High Score: %d  Lives: %d\n",
			m.state.GameMode,
			m.floor.Index,
			m.score.Get(),
			m.score.GetHigh(),
			m.haunteed.Lives(),
		)
	} else {
		header = fmt.Sprintf("Latitude: %.4f, Longitude: %.4f, Timezone: %s\n", m.state.LocationInfo.Lat, m.state.LocationInfo.Lon, m.state.LocationInfo.Timezone)
		header += fmt.Sprintf("Mode: %s, Night: %s  Floor: %d\nScore: %d  High Score: %d  Lives: %d\n",
			m.state.GameMode,
			m.state.CrazyNight,
			m.floor.Index,
			m.score.Get(),
			m.score.GetHigh(),
			m.haunteed.Lives(),
		)
	}

	m.sb.WriteString(style.PlayHeader.Render(header))
	m.sb.WriteRune('\n')

	m.renderView()

	// Controls footer
	m.sb.WriteString("\n← ↑ ↓ → — move, q — quit\n")
	return m.sb.String()
}

func (m *Model) renderView() {
	isLage := m.state.SpriteSize == state.SpriteLarge
	f := m.floor
	h := m.haunteed
	g := m.ghosts

	// In "Crazy" mode, basement floors are always dark.
	// If "CrazyNight" is set to "never", upper floors are always lit.
	// Upper floors are dark only if "CrazyNight" option is set to "always" or "real".
	// If "CrazyNight" is set to "always", the upper floor is always dark.
	// If "CrazyNight" is set to "real", the upper floor is dark only during the real night,
	// in dawn and dusk it is lit but has reduced visibility and in daylight it is fully lit.
	isCrazyMode := m.state.GameMode == state.ModeCrazy
	isLimitedVisibilityActive := isCrazyMode && (m.floor.Index < 0 || m.state.CrazyNight == state.CrazyNightAlways || m.state.CrazyNight == state.CrazyNightReal)
	htPos := h.Pos()

	// Create a map of dweller positions to their rendered sprites for efficient lookup.
	// Ghosts are added after the haunteed to ensure they are rendered on top
	// if they occupy the same cell, matching the original rendering logic.
	dwellerSprites := make(map[dweller.Position][]string)
	dwellerSprites[h.Pos()] = h.Render(m.state.SpriteSize)
	for _, gh := range g {
		dwellerSprites[gh.Pos()] = gh.Render(m.state.SpriteSize)
	}

	for y := 0; y < m.floor.Maze.Height(); y++ {
		var line1, line2 strings.Builder
		for x := 0; x < f.Maze.Width(); x++ {
			var sprite []string
			pos := dweller.Position{X: x, Y: y}
			if isLimitedVisibilityActive && !m.fullVisibility && distance(pos, htPos) > m.floor.VisibilityRadius {
				sprite = f.Sprites[floor.Empty]
			} else {
				if sp, ok := dwellerSprites[pos]; ok {
					sprite = sp
				} else {
					item, _ := f.ItemAt(x, y)
					if item == floor.Fuse && !m.fullVisibility {
						sprite = f.DimFuseSprite
					} else {
						sprite = f.RenderAt(x, y)
					}
				}
			}
			line1.WriteString(sprite[0])
			if isLage {
				line2.WriteString(sprite[1])
			}
		}
		m.sb.WriteString(line1.String())
		m.sb.WriteRune('\n')
		if isLage {
			m.sb.WriteString(line2.String())
			m.sb.WriteRune('\n')
		}
	}
}
