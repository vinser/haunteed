package play

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vinser/haunteed/internal/dweller"
	floor "github.com/vinser/haunteed/internal/floor"
	"github.com/vinser/haunteed/internal/score"
	"github.com/vinser/haunteed/internal/sound"
	"github.com/vinser/haunteed/internal/state"
	"github.com/vinser/haunteed/internal/style"
)

const (
	frightenedPeriod    = 10 * time.Second
	autoRepeatThreshold = 100 * time.Millisecond // Anticheat
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
	paused            bool
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
	Score int
}

func gameOverCmd(score int) tea.Cmd {
	return func() tea.Msg {
		return GameOverMsg{
			Score: score,
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

// New returns a new play model.
func New(s *state.State, f *floor.Floor, sc *score.Score, h *dweller.Haunteed, floorVisibility bool) Model {
	rng := rand.New(rand.NewSource(s.FloorSeeds[f.Index]))
	ghosts := dweller.PlaceGhosts(f.Index, s.SpriteSize, s.GameMode, f.Maze.Width(), f.Maze.Height(), f.Maze.DenWidth(), f.Maze.DenHeight(), rng)
	ghostTick := f.GhostTickInterval
	m := Model{
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
		fullVisibility:    floorVisibility,
	}

	if m.shouldPlayFuseSound() {
		m.state.SoundManager.PlayLoopWithVolume(sound.FUSE_ARC, 2)
	}

	return m
}

func (m Model) shouldPlayFuseSound() bool {
	isLimitedVisibilityFloor := m.floor.VisibilityRadius < floor.FullFloorVisibilityRadius
	return m.state.GameMode == state.ModeCrazy && !m.fullVisibility && isLimitedVisibilityFloor
}

func (m Model) Init() tea.Cmd {
	return tickGhosts()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	// Handle pause/resume toggling first. This should work regardless of the paused state.
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "p": // Toggle pause
			m.paused = !m.paused
			if m.paused {
				m.state.SoundManager.PlayLoopWithVolume(sound.PAUSE_GAME, 0)
				return m, nil // Game is paused, no more ticks
			} else {
				m.state.SoundManager.StopListed(sound.PAUSE_GAME)
			}
			return m, tickGhosts() // Game is resumed, start ticking again
		}
	}

	// If paused, ignore all other messages and updates.
	if m.paused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// sound.ClearSpeaker()
		// Distinguish between a real key press and an auto-repeat.
		// An auto-repeat event is just the same key coming in very fast.
		isAutoRepeat := msg.Type == m.lastKeyMsg.Type &&
			msg.String() == m.lastKeyMsg.String() &&
			time.Since(m.lastKeyTime) < autoRepeatThreshold

		// Update last key press info for the next event.
		m.lastKeyMsg = msg
		m.lastKeyTime = time.Now()

		if isAutoRepeat {
			return m, nil // Ignore auto-repeat events
		}

		m.haunteed.HandleInput(msg.String())

		nextPos := m.haunteed.NextPos()
		if m.haunteed.Dir() != dweller.No {
			tile, err := m.floor.ItemAt(nextPos.X, nextPos.Y)
			canMove := false
			if err == nil {
				if tile == floor.CrumblingWall {
					if m.powerMode {
						m.floor.BreakWall(nextPos.X, nextPos.Y)
						m.state.SoundManager.Play(sound.WALL_BREAK)
						canMove = true
					}
				} else if tile != floor.Wall {
					canMove = true
				}
			}
			if canMove {
				m.haunteed.SetPos(nextPos)
				m.state.SoundManager.Play(sound.STEP_CREAKY)
			} else {
				m.state.SoundManager.Play(sound.STEP_BUMP)
			}
		}

		pos := m.haunteed.Pos()
		tile := m.floor.EatItem(pos.X, pos.Y)

		switch tile {
		case floor.Dot:
			if m.state.GameMode == state.ModeNoisy {
				m.score.Add(20)
			} else {
				m.score.Add(10)
			}
			m.state.SoundManager.PlayWithVolume(sound.PICK_CRUMB, -1.5)
		case floor.PowerPellet:
			m.state.SoundManager.Play(sound.EAT_PELLET)
			m.score.Add(50)
			m.powerMode = true
			m.powerModeUntil = time.Now().Add(frightenedPeriod)
			m.ghostTickInterval = m.floor.GhostTickInterval * 2 // slow down ghosts
			for _, g := range m.ghosts {
				g.SetState(dweller.Frightened)
			}
		case floor.Fuse:
			m.fullVisibility = !m.fullVisibility
			m.state.SoundManager.Play(sound.FUSE_TOGGLE)
			if m.shouldPlayFuseSound() {
				m.state.SoundManager.PlayLoopWithVolume(sound.FUSE_ARC, 2)
			} else {
				m.state.SoundManager.StopListed(sound.FUSE_ARC)
			}
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
				m.state.SoundManager.Play(sound.KILL_GHOST)
				m.score.AddGhostPoints()
				g.SetState(dweller.Eaten)
			case dweller.Chase: // lose a life
				m.haunteed.LoseLife()
				if m.haunteed.IsDead() { // game over
					score := m.score.Get()
					return m, gameOverCmd(score)
				}
				// enter respawn mode
				m.state.SoundManager.PlayWithVolume(sound.LOSE_LIFE, 2)
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
	if m.paused {
		header = "\n\nPAUSED\n"
	} else if m.state.GameMode != state.ModeCrazy {
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
			m.state.NightOption,
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
	m.sb.WriteString("\n← ↑ ↓ → — move, p — pause/resume, q — quit\n")
	return m.sb.String()
}

func (m *Model) renderView() {
	isLage := m.state.SpriteSize == state.SpriteLarge
	f := m.floor
	h := m.haunteed
	g := m.ghosts

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
			if m.notVisible(pos, htPos) {
				sprite = f.Sprites[floor.Empty]
			} else {
				if sp, ok := dwellerSprites[pos]; ok {
					sprite = sp
				} else {
					item, _ := f.ItemAt(x, y)
					if item == floor.CrumblingWall {
						if m.powerMode {
							sprite = f.Sprites[floor.CrumblingWall]
						} else {
							// Render as a normal wall when not in power mode
							sprite = f.Sprites[floor.Wall]
						}
					} else if item == floor.Fuse && !m.fullVisibility {
						sprite = f.DimFuseSprite
					} else {
						sprite = f.Sprites[item]
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

// notVisible checks if the sprite is not visible to the haunteed.
// In "Crazy" mode, basement floors are always dark.
// If "NightOption" is set to "never", upper floors are always lit.
// Upper floors are dark only if "NightOption" option is set to "always" or "real".
// If "NightOption" is set to "always", the upper floor is always dark.
// If "NightOption" is set to "real", the upper floor is dark only during the real night,
// in dawn and dusk it is lit but has reduced visibility and in daylight it is fully lit.
func (m Model) notVisible(spritePos, hauntedPos dweller.Position) bool {
	isLimitedVisibilityActive := (m.state.GameMode == state.ModeCrazy) && (m.floor.Index < 0 || m.state.NightOption == state.NightAlways || m.state.NightOption == state.NightReal)
	return isLimitedVisibilityActive && !m.fullVisibility && distance(spritePos, hauntedPos) > m.floor.VisibilityRadius

}
