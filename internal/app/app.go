package app

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vinser/haunteed/internal/dweller"

	"github.com/vinser/haunteed/internal/flags"
	floor "github.com/vinser/haunteed/internal/floor"
	"github.com/vinser/haunteed/internal/model/intro"
	"github.com/vinser/haunteed/internal/model/over"
	"github.com/vinser/haunteed/internal/model/play"
	"github.com/vinser/haunteed/internal/model/quit"
	"github.com/vinser/haunteed/internal/model/respawn"
	"github.com/vinser/haunteed/internal/model/setup"
	"github.com/vinser/haunteed/internal/model/splash"
	"github.com/vinser/haunteed/internal/score"
	"github.com/vinser/haunteed/internal/state"
	"github.com/vinser/maze"
)

type status uint

const (
	statusStartSplash status = iota
	statusDoSettings
	statusGameplay
	statusFloorIntro
	statusRespawning
	statusGameOver
	statusQuitting
)

type Model struct {
	status          status
	state           *state.State
	floorCache      map[int]*floor.Floor
	floorVisibility map[int]bool // Persists visibility state for "Crazy" mode across floors
	haunteed        *dweller.Haunteed
	floor           *floor.Floor
	score           *score.Score
	// models
	splash  splash.Model
	setup   setup.Model
	play    play.Model
	intro   intro.Model
	respawn respawn.Model
	over    over.Model
	quit    quit.Model
}

func New() Model {
	state := getState()
	splash := setSplash(state)

	floorCache := make(map[int]*floor.Floor)
	initialFloor := getFloor(0, state, floorCache, nil, nil)
	startPos := dweller.Position{X: initialFloor.Maze.Start().X, Y: initialFloor.Maze.Start().Y}
	haunteed := dweller.PlaceHaunteed(state.SpriteSize, startPos)
	score := score.NewScore()
	score.SetHigh(state.GetHighScore())
	return Model{
		status:          statusStartSplash,
		state:           state,
		floorCache:      floorCache,
		floorVisibility: make(map[int]bool),
		haunteed:        haunteed,
		floor:           initialFloor,
		score:           score,
		splash:          splash,
	}
}

func getState() *state.State {
	st := state.Load()
	if fl, ok := flags.Parse(); ok {
		if fl.Reset {
			return state.New()
		}

		if fl.Mute {
			st.Mute = true
		}
		if fl.Mode != "" && fl.Mode != st.GameMode {
			st.GameMode = fl.Mode
		}
		if fl.CrazyNight != "" {
			st.CrazyNight = fl.CrazyNight
		}
		if fl.SpriteSize != "" {
			st.SpriteSize = fl.SpriteSize
		}
	}
	return st
}

func setSplash(st *state.State) splash.Model {
	switch st.SpriteSize {
	case "small":
		return splash.New(21, 15)
	case "medium":
		return splash.New(43, 15)
	case "large":
		return splash.New(85, 31)
	default:
		return splash.New(43, 15)
	}
}

func getFloor(index int, st *state.State, cache map[int]*floor.Floor, startPoint, endPoint *maze.Point) *floor.Floor {
	if f, ok := cache[index]; ok {
		// A floor is regenerated if the required connection points do not match the cached version.
		// This ensures that returning to a floor from a different direction connects correctly.
		startMismatch := startPoint != nil && f.Maze.Start() != *startPoint
		endMismatch := endPoint != nil && f.Maze.End() != *endPoint

		if !startMismatch && !endMismatch {
			return f // Cached version is compatible, return it.
		}
	}
	// Not in cache or incompatible, (re)generate the floor.
	if _, ok := st.FloorSeeds[index]; !ok {
		st.FloorSeeds[index] = time.Now().UnixNano()
	}
	f := floor.New(index, st.FloorSeeds[index], startPoint, endPoint, st.SpriteSize, st.GameMode, st.CrazyNight)

	cache[index] = f
	return f
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.splash.Init())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q": // quit all app models
			m.status = statusQuitting // Set status to show quit message
			return m, tea.Quit
		case "m":
			m.state.Mute = !m.state.Mute
			return m, nil
		}
	}

	switch m.status {
	case statusStartSplash:
		switch msg := msg.(type) {
		case splash.MakeSettingsMsg:
			m.status = statusDoSettings
			m.setup = setup.New(m.state.GameMode, m.state.CrazyNight, m.state.SpriteSize, m.state.Mute)
		case splash.TimedoutMsg:
			m.status = statusGameplay
			m.resetPlayModel()
		default:
			m.splash, cmd = m.splash.Update(msg)
		}
		cmds = append(cmds, cmd)
	case statusDoSettings:
		switch msg := msg.(type) {
		case setup.SaveSettingsMsg:
			m.status = statusGameplay
			if msg.Reset {
				m.state = state.New()
			} else {
				m.state.GameMode = msg.Mode
				m.state.Mute = msg.Mute
				m.state.CrazyNight = msg.CrazyNight
			}
			// Reset for a new game
			m.floorCache = make(map[int]*floor.Floor)
			m.floorVisibility = make(map[int]bool)
			m.floor = getFloor(0, m.state, m.floorCache, nil, nil)
			startPos := dweller.Position{X: m.floor.Maze.Start().X, Y: m.floor.Maze.Start().Y}
			m.haunteed = dweller.PlaceHaunteed(m.state.SpriteSize, startPos)
			m.score.Reset()
			m.score.SetHigh(m.state.GetHighScore())
			m.resetPlayModel()
		case setup.DiscardSettingsMsg:
			m.status = statusGameplay
			m.resetPlayModel()
		default:
			m.setup, cmd = m.setup.Update(msg)
		}
		cmds = append(cmds, cmd)
	case statusGameplay:
		switch msg := msg.(type) {
		case play.NextFloorMsg:
			m.status = statusFloorIntro
			nextFloorIndex := m.floor.Index + 1
			prevFloorEndPoint := m.floor.Maze.End()
			m.floor = getFloor(nextFloorIndex, m.state, m.floorCache, &prevFloorEndPoint, nil)
			startPoint := m.floor.Maze.Start()
			m.haunteed.SetPos(dweller.Position{X: startPoint.X, Y: startPoint.Y})
			m.haunteed.SetHome(dweller.Position{X: startPoint.X, Y: startPoint.Y})
			m.haunteed.SetHaunteedSprites(m.state.SpriteSize)
			m.intro = intro.New(nextFloorIndex)
		case play.PrevFloorMsg:
			m.status = statusFloorIntro
			prevFloorIndex := m.floor.Index - 1
			currentFloorStartPoint := m.floor.Maze.Start()
			// The new floor's end must connect to the current floor's start.
			m.floor = getFloor(prevFloorIndex, m.state, m.floorCache, nil, &currentFloorStartPoint)
			endPoint := m.floor.Maze.End()
			m.haunteed.SetPos(dweller.Position{X: endPoint.X, Y: endPoint.Y})
			startPoint := m.floor.Maze.Start()
			m.haunteed.SetHome(dweller.Position{X: startPoint.X, Y: startPoint.Y})
			m.haunteed.SetHaunteedSprites(m.state.SpriteSize)
			m.intro = intro.New(prevFloorIndex)
		case play.RespawnMsg:
			m.status = statusRespawning
			m.respawn = respawn.New(msg.Lives)
		case play.GameOverMsg:
			m.status = statusGameOver
			m.over = over.New(msg.Mode, msg.Score, msg.HighScore)
		case play.VisibilityToggledMsg:
			m.floorVisibility[msg.FloorIndex] = msg.IsVisible
			return m, nil // State updated, no further action needed
		default:
			m.play, cmd = m.play.Update(msg)
		}
		cmds = append(cmds, cmd)
	case statusFloorIntro:
		switch msg := msg.(type) {
		case intro.TimedoutMsg:
			m.resetPlayModel()
			m.status = statusGameplay
		default:
			m.intro, cmd = m.intro.Update(msg)
		}
		cmds = append(cmds, cmd)
	case statusRespawning:
		switch msg := msg.(type) {
		case respawn.TimedoutMsg:
			m.resetPlayModelForRespawn()
			m.status = statusGameplay
		default:
			m.respawn, cmd = m.respawn.Update(msg)
		}
		cmds = append(cmds, cmd)
	case statusGameOver:
		m.over, cmd = m.over.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) resetPlayModel() {
	m.play = play.New(m.state, m.floor, m.score, m.haunteed, m.floorVisibility[m.floor.Index])
}

func (m *Model) resetPlayModelForRespawn() {
	// The play model holds the haunteed and ghosts. We need a new one to reset their positions.
	// The score and floor state are preserved.
	// We keep the current haunteed instance because it tracks lives.
	m.haunteed.SetPos(m.haunteed.Home())
	// Create a new play model, which will re-place ghosts.
	m.play = play.New(m.state, m.floor, m.score, m.haunteed, m.floorVisibility[m.floor.Index])
}

func (m Model) View() string {
	switch m.status {
	case statusStartSplash:
		return m.splash.View()
	case statusDoSettings:
		return m.setup.View()
	case statusGameplay:
		return m.play.View()
	case statusFloorIntro:
		return m.intro.View()
	case statusRespawning:
		return m.respawn.View()
	case statusGameOver:
		return m.over.View()
	case statusQuitting:
		return m.quit.View()
	}
	return ""
}
