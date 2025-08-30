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
	"github.com/vinser/haunteed/internal/model/motd"
	"github.com/vinser/haunteed/internal/score"
	"github.com/vinser/haunteed/internal/sound"
	"github.com/vinser/haunteed/internal/state"
	"github.com/vinser/haunteed/internal/style"
)

const (
	frightenedPeriod    = 10 * time.Second
	autoRepeatThreshold = 100 * time.Millisecond // Anticheat
)

// Viewport represents the visible area of the maze
type Viewport struct {
	StartX, StartY int // Top-left corner of viewport in maze coordinates
	Width, Height  int // Dimensions of viewport
}

// TerminalDimensions holds the terminal size information
type TerminalDimensions struct {
	Width  int
	Height int
}

type Model struct {
	state             *state.State
	soundManager      *sound.Manager
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
	gotCrumbs         bool
	terminal          TerminalDimensions // Terminal dimensions
	viewport          Viewport           // Current viewport for scrolling
	motd              motd.Model
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

// WindowSizeMsg is a message sent when the terminal is resized.
type WindowSizeMsg struct {
	Width  int
	Height int
}

func windowSizeCmd(width, height int) tea.Cmd {
	return func() tea.Msg {
		return WindowSizeMsg{
			Width:  width,
			Height: height,
		}
	}
}

// New returns a new play model.
func New(s *state.State, sm *sound.Manager, f *floor.Floor, sc *score.Score, h *dweller.Haunteed, floorVisibility bool) Model {
	rng := rand.New(rand.NewSource(s.FloorSeeds[f.Index]))
	ghosts := dweller.PlaceGhosts(f.Index, s.SpriteSize, s.GameMode, f.Maze.Width(), f.Maze.Height(), f.Maze.DenWidth(), f.Maze.DenHeight(), rng)
	ghostTick := f.GhostTickInterval

	// Calculate minimal viewport size based on noisy mode maze size plus header/footer
	minViewportWidth := 31  // ModeNoisyWidth
	minViewportHeight := 21 // ModeNoisyHeight
	headerHeight := 3       // Approximate header height
	footerHeight := 1       // Footer height
	minTerminalHeight := minViewportHeight + headerHeight + footerHeight

	m := Model{
		state:             s,
		soundManager:      sm,
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
		terminal:          TerminalDimensions{Width: 80, Height: minTerminalHeight}, // Default minimal size
		viewport:          Viewport{StartX: 0, StartY: 0, Width: minViewportWidth, Height: minViewportHeight},
		motd:              motd.New(f.Maze.Width()*2, 1, 1*time.Minute),
	}

	if m.shouldPlayFuseSound() {
		m.soundManager.PlayLoopWithVolume(sound.FUSE_ARC, 2)
	}

	return m
}

func (m Model) shouldPlayFuseSound() bool {
	isLimitedVisibilityFloor := m.floor.VisibilityRadius < m.floor.FullVisibilityRadius()
	return m.state.GameMode == state.ModeCrazy && !m.fullVisibility && isLimitedVisibilityFloor
}

func (m Model) Init() tea.Cmd {
	return tickGhosts()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	// Handle MOTD updates separately
	if _, ok := msg.(motd.TickMsg); ok && m.paused {
		newMotd, motdCmd := m.motd.Update(msg)
		m.motd = newMotd
		return m, motdCmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "p": // Toggle pause
			m.paused = !m.paused
			if m.paused {
				m.soundManager.PlayLoopWithVolume(sound.PAUSE_GAME, 0)
				return m, m.motd.Init()
			} else {
				m.soundManager.StopListed(sound.PAUSE_GAME)
				return m, tickGhosts() // Game is resumed, start ticking again
			}
		case "c":
			if !m.paused && !m.gotCrumbs && m.state.GameMode == state.ModeCrazy && m.haunteed.Lives() > 1 {
				m.floor.ShowCrumbs(m.floor.Index, m.state.SpriteSize)
				m.haunteed.LoseLife()
				m.gotCrumbs = true
				return m, tickGhosts()
			}
		}
	case WindowSizeMsg:
		// Handle terminal resize
		m.terminal.Width = msg.Width
		m.terminal.Height = msg.Height
		// Force a complete viewport reset and recalculation
		m.resetViewport()
		m.updateViewport()
		m.motd.SetWidth(m.terminal.Width)
		return m, nil
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
						m.soundManager.Play(sound.WALL_BREAK)
						canMove = true
					}
				} else if tile != floor.Wall {
					canMove = true
				}
			}
			if canMove {
				m.haunteed.SetPos(nextPos)
				m.soundManager.Play(sound.STEP_CREAKY)
				// Update viewport to follow player if using scrolling
				m.centerViewportOnPlayer()
			} else {
				m.soundManager.Play(sound.STEP_BUMP)
			}
		}

		pos := m.haunteed.Pos()
		tile := m.floor.EatItem(pos.X, pos.Y)

		switch tile {
		case floor.Dot:
			points := 0
			switch m.state.GameMode {
			case state.ModeEasy:
				points = 5
			case state.ModeNoisy:
				points = 10
			case state.ModeCrazy:
				points = 15
				if !m.fullVisibility {
					points *= 2
				}
				if m.gotCrumbs {
					points = 5
				}
			}
			m.score.Add(points)
			m.soundManager.PlayWithVolume(sound.PICK_CRUMB, -1.5)
		case floor.PowerPellet:
			m.soundManager.Play(sound.EAT_PELLET)
			m.score.Add(50)
			m.powerMode = true
			m.powerModeUntil = time.Now().Add(frightenedPeriod)
			m.ghostTickInterval = m.floor.GhostTickInterval * 2 // slow down ghosts
			for _, g := range m.ghosts {
				g.SetState(dweller.Frightened)
			}
		case floor.Fuse:
			m.fullVisibility = !m.fullVisibility
			m.soundManager.Play(sound.FUSE_TOGGLE)
			if m.shouldPlayFuseSound() {
				m.soundManager.PlayLoop(sound.FUSE_ARC)
			} else {
				m.soundManager.StopListed(sound.FUSE_ARC)
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
				m.soundManager.Play(sound.KILL_GHOST)
				m.score.AddGhostPoints()
				g.SetState(dweller.Eaten)
			case dweller.Chase: // lose a life
				m.haunteed.LoseLife()
				if m.haunteed.IsDead() { // game over
					score := m.score.Get()
					return m, gameOverCmd(score)
				}
				// enter respawn mode
				m.soundManager.PlayWithVolume(sound.LOSE_LIFE, 2)
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

// max returns the larger of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

// getSpriteCharDims returns (tileWidthChars, tileHeightRows) for the current sprite size
func (m *Model) getSpriteCharDims() (int, int) {
	switch m.state.SpriteSize {
	case state.SpriteSmall:
		return 1, 1
	case state.SpriteLarge:
		return 4, 2
	default: // state.SpriteMedium
		return 2, 1
	}
}

// shouldScrollHorizontally returns true if the maze width is greater than the terminal width.
func (m *Model) shouldScrollHorizontally() bool {
	mazeWidthChars, _ := m.getMazePixelDimensions()
	return mazeWidthChars > m.terminal.Width
}

// shouldScrollVertically returns true if the maze height is greater than the available terminal height.
func (m *Model) shouldScrollVertically() bool {
	_, mazeHeightRows := m.getMazePixelDimensions()
	headerH := m.headerRows()
	footerH := 1
	availableHeight := m.terminal.Height - headerH - footerH
	return mazeHeightRows > availableHeight
}

// shouldScroll check if maze should be scroll horizontally or/and vertically
func (m *Model) shouldScroll() (scrollH, scrollV bool) {
	wChar, hRows := m.getMazePixelDimensions()
	headerH := m.headerRows()
	footerH := 1
	availableHeight := m.terminal.Height - headerH - footerH - 1
	scrollH = wChar > m.terminal.Width
	scrollV = hRows > availableHeight
	return scrollH, scrollV
}

// getMazePixelDimensions returns the maze dimensions in characters and rows.
func (m *Model) getMazePixelDimensions() (int, int) {
	wChar, hRows := m.getSpriteCharDims()
	mazeWidth := m.floor.Maze.Width()
	mazeHeight := m.floor.Maze.Height()
	return mazeWidth * wChar, mazeHeight * hRows
}

// updateViewport recalculates the viewport dimensions and position based on terminal size
func (m *Model) updateViewport() {
	mazeWidth := m.floor.Maze.Width()
	mazeHeight := m.floor.Maze.Height()
	wChar, hRows := m.getSpriteCharDims()
	headerH := m.headerRows()
	footerH := 1
	availableHeightRows := m.terminal.Height - headerH - footerH - 1
	if availableHeightRows < 1 {
		availableHeightRows = 1
	}
	maxCellsWide := m.terminal.Width / wChar
	if maxCellsWide < 1 {
		maxCellsWide = 1
	}
	maxCellsHigh := availableHeightRows / hRows
	if maxCellsHigh < 1 {
		maxCellsHigh = 1
	}

	if m.shouldScrollHorizontally() {
		m.viewport.Width = max(maxCellsWide, 1)
	} else {
		m.viewport.Width = mazeWidth
	}

	if m.shouldScrollVertically() {
		m.viewport.Height = max(maxCellsHigh, 1)
	} else {
		m.viewport.Height = mazeHeight
	}

	m.viewport.Width = min(m.viewport.Width, mazeWidth)
	m.viewport.Height = min(m.viewport.Height, mazeHeight)
	if m.viewport.Width < 1 {
		m.viewport.Width = 1
	}
	if m.viewport.Height < 1 {
		m.viewport.Height = 1
	}
	m.centerViewportOnPlayer()
}

// centerViewportOnPlayer centers the viewport on the player's position
func (m *Model) centerViewportOnPlayer() {
	playerPos := m.haunteed.Pos()
	mazeWidth := m.floor.Maze.Width()
	mazeHeight := m.floor.Maze.Height()

	// Center viewport on player
	m.viewport.StartX = playerPos.X - m.viewport.Width/2
	m.viewport.StartY = playerPos.Y - m.viewport.Height/2

	// Clamp viewport to maze boundaries
	if m.viewport.StartX < 0 {
		m.viewport.StartX = 0
	}
	if m.viewport.StartY < 0 {
		m.viewport.StartY = 0
	}
	if m.viewport.StartX+m.viewport.Width > mazeWidth {
		m.viewport.StartX = mazeWidth - m.viewport.Width
	}
	if m.viewport.StartY+m.viewport.Height > mazeHeight {
		m.viewport.StartY = mazeHeight - m.viewport.Height
	}

	// Ensure player is always visible in the viewport
	if playerPos.X < m.viewport.StartX || playerPos.X >= m.viewport.StartX+m.viewport.Width ||
		playerPos.Y < m.viewport.StartY || playerPos.Y >= m.viewport.StartY+m.viewport.Height {
		// Player is outside viewport, adjust to include player
		if playerPos.X < m.viewport.StartX {
			m.viewport.StartX = max(0, playerPos.X-2) // Show 2 columns before player
		} else if playerPos.X >= m.viewport.StartX+m.viewport.Width {
			m.viewport.StartX = min(mazeWidth-m.viewport.Width, playerPos.X-m.viewport.Width+2) // Show 2 columns after player
		}

		if playerPos.Y < m.viewport.StartY {
			m.viewport.StartY = max(0, playerPos.Y-2) // Show 2 rows before player
		} else if playerPos.Y >= m.viewport.StartY+m.viewport.Height {
			m.viewport.StartY = min(mazeHeight-m.viewport.Height, playerPos.Y-m.viewport.Height+2) // Show 2 rows after player
		}
	}
}

// render renders the maze with the appropriate scrolling and centering.
func (m *Model) render() {
	scrollH, scrollV := m.shouldScroll()

	if scrollH || scrollV {
		m.updateViewport()
	}

	mazeWidth := m.floor.Maze.Width()
	mazeHeight := m.floor.Maze.Height()

	startX, startY := 0, 0
	viewW, viewH := mazeWidth, mazeHeight
	centerH, centerV := !scrollH, !scrollV

	if scrollH {
		startX = m.viewport.StartX
		viewW = m.viewport.Width
	}
	if scrollV {
		startY = m.viewport.StartY
		viewH = m.viewport.Height
	}
	wChar, hRows := m.getSpriteCharDims()
	mazeWidthChars := viewW * wChar
	mazeHeightRows := viewH * hRows

	var horizontalPadding, verticalPadding int
	if centerH {
		horizontalPadding = (m.terminal.Width - mazeWidthChars) / 2
		if horizontalPadding < 0 {
			horizontalPadding = 0
		}
	}
	if centerV {
		headerH := m.headerRows()
		footerH := 1
		availableHeight := m.terminal.Height - headerH - footerH - 1
		verticalPadding = (availableHeight - mazeHeightRows) / 2
		if verticalPadding < 0 {
			verticalPadding = 0
		}
	}

	m.renderTopBar(mazeWidthChars, horizontalPadding)
	for i := 0; i < verticalPadding; i++ {
		m.sb.WriteString("\n")
	}

	m.renderHeader(horizontalPadding)

	m.renderMaze(startX, startY, viewW, viewH, horizontalPadding)

	m.renderMOTD(mazeWidthChars, horizontalPadding)

	// Controls footer (single line)
	m.renderFooter(mazeWidthChars, horizontalPadding)
}

// renderTopBar
func (m *Model) renderTopBar(width, hPadding int) {
	m.sb.WriteString(strings.Repeat(" ", hPadding))
	m.sb.WriteString(style.TopPattern.Render(strings.Repeat("/", width)))
	m.sb.WriteString("\n")
}

// renderHeader
func (m *Model) renderHeader(hPadding int) {
	m.sb.WriteString(style.Title.Render(m.headerText(hPadding)))
	m.sb.WriteString("\n")
}

// headerText builds the header string based on game state. It always ends with a newline and may span multiple lines.
func (m *Model) headerText(horizontalPadding int) string {
	var b strings.Builder
	padString := strings.Repeat(" ", horizontalPadding)
	if m.paused {
		b.WriteString("\n\n")
		b.WriteString(padString)
		b.WriteString("PAUSED")
	} else {
		if m.state.GameMode == state.ModeCrazy {
			// First line: geo/time info (restored)
			b.WriteString(padString)
			b.WriteString(fmt.Sprintf("Latitude: %.4f, Longitude: %.4f, Timezone: %s\n", m.state.LocationInfo.Lat, m.state.LocationInfo.Lon, m.state.LocationInfo.Timezone))
			// Second line: mode/night/floor
			b.WriteString(padString)
			b.WriteString(fmt.Sprintf("Mode: %s, Night: %s  Floor: %d  Lives: %d\n", m.state.GameMode, m.state.NightOption, m.floor.Index, m.haunteed.Lives()))
		} else {
			// One line: mode/floor
			b.WriteString("\n")
			b.WriteString(padString)
			b.WriteString(fmt.Sprintf("Mode: %s  Floor: %d  Lives: %d\n", m.state.GameMode, m.floor.Index, m.haunteed.Lives()))
		}
		// Final line: score/lives
		b.WriteString(padString)
		highScore := m.score.GetHigh()
		if highScore > 0 {
			b.WriteString(fmt.Sprintf("Score: %d  High Score: %d by %s", m.score.Get(), m.score.GetHigh(), m.score.GetHighNick()))
		} else {
			b.WriteString(fmt.Sprintf("Score: %d  High Score: —", m.score.Get()))
		}
	}
	return b.String()
}

// headerRows returns the number of terminal rows the header occupies
func (m *Model) headerRows() int {
	return 3
}

// renderMaze renders a specific viewport of the maze.
func (m *Model) renderMaze(startX, startY, width, height, horizontalPadding int) {
	isLarge := m.state.SpriteSize == state.SpriteLarge
	f := m.floor
	h := m.haunteed
	g := m.ghosts
	htPos := h.Pos()

	dwellerSprites := make(map[dweller.Position][]string)
	dwellerSprites[h.Pos()] = h.Render(m.state.SpriteSize)
	for _, gh := range g {
		dwellerSprites[gh.Pos()] = gh.Render(m.state.SpriteSize)
	}

	for y := startY; y < startY+height && y < f.Maze.Height(); y++ {
		var line1, line2 strings.Builder
		if horizontalPadding > 0 {
			line1.WriteString(strings.Repeat(" ", horizontalPadding))
			if isLarge {
				line2.WriteString(strings.Repeat(" ", horizontalPadding))
			}
		}
		for x := startX; x < startX+width && x < f.Maze.Width(); x++ {
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
			if isLarge {
				line2.WriteString(sprite[1])
			}
		}
		m.sb.WriteString(line1.String())
		m.sb.WriteRune('\n')
		if isLarge {
			m.sb.WriteString(line2.String())
			m.sb.WriteRune('\n')
		}
	}
}

// getStyledMOTD renders MOTD only when the gameplay is paused
func (m *Model) getStyledMOTD(width int) string {
	if !m.paused {
		return ""
	}
	m.motd.SetWidth(width)
	return m.motd.View()
}

// renderMOTD
func (m *Model) renderMOTD(width, horizontalPadding int) {
	m.sb.WriteString(strings.Repeat(" ", horizontalPadding))
	m.sb.WriteString(m.getStyledMOTD(width))
}

// renderFooter
func (m *Model) renderFooter(width, hPadding int) {
	m.sb.WriteString("\n")
	m.sb.WriteString(strings.Repeat(" ", hPadding))
	header := ""
	switch {
	case m.paused:
		header = "p — resume, q — quit"
	case m.state.GameMode == state.ModeCrazy && !m.gotCrumbs && m.haunteed.Lives() > 1:
		header = "← ↑ ↓ → — move, p — pause, c — crumbs, q — quit"
	default:
		header = "← ↑ ↓ → — move, p — pause, q — quit"
	}
	m.sb.WriteString(style.Footer.Render(header))
	m.sb.WriteString(style.Footer.Render(strings.Repeat("/", width-len([]rune(header)))))
	m.sb.WriteString("\n")

}

//

// resetViewport completely resets the viewport to initial state
func (m *Model) resetViewport() {
	m.viewport = Viewport{StartX: 0, StartY: 0, Width: 0, Height: 0}
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

// View returns the complete screen output with game entities and stats.
func (m *Model) View() string {
	m.sb.Reset()

	m.render()

	return m.sb.String()
}
