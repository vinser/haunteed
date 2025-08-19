package app

import (
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vinser/haunteed/internal/ambilite"
	"github.com/vinser/haunteed/internal/dweller"
	"github.com/vinser/haunteed/internal/geoip"
	"github.com/vinser/haunteed/internal/sound"

	"github.com/vinser/haunteed/internal/flags"
	"github.com/vinser/haunteed/internal/floor"
	"github.com/vinser/haunteed/internal/model/next"
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
	next    next.Model
	respawn respawn.Model
	over    over.Model
	quit    quit.Model
	// terminal size cache
	termWidth  int
	termHeight int
}

func New() Model {
	// Configure global settings first to ensure consistent behavior.
	geoip.SetCacheTTL(0) // Ensure fresh location data for new sessions.

	state := getState()
	splash := setSplash(state)
	state.SoundManager.PlayLoop(sound.INTRO)
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
			st.SetMute(true)
		}
		if fl.Mode != "" {
			st.GameMode = fl.Mode
		}
		if fl.Night != "" {
			st.NightOption = fl.Night
		}
		if fl.Sprite != "" {
			st.SpriteSize = fl.Sprite
		}
	}
	return st
}

func setSplash(st *state.State) splash.Model {
	var s splash.Model
	switch st.SpriteSize {
	case "small":
		s = splash.New(st, 21, 15)
	case "medium":
		s = splash.New(st, 47, 15)
	case "large":
		s = splash.New(st, 85, 31)
	default:
		s = splash.New(st, 47, 15)
	}
	return s
}

const minFloorVisibilityRadius = 4

func getFloor(index int, st *state.State, cache map[int]*floor.Floor, startPoint, endPoint *maze.Point) *floor.Floor {
	if f, ok := cache[index]; ok {
		// A floor is regenerated if the required connection points (upstairs or downstairs) do not match the cached version.
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
	f := floor.New(index, st.FloorSeeds[index], startPoint, endPoint, st.SpriteSize, st.GameMode, st.NightOption)

	// Set floor visibility radius
	setFloorVisibility(f, st)
	cache[index] = f

	return f
}

// Set floor visibility radius
func setFloorVisibility(f *floor.Floor, st *state.State) {
	litIntensity := ambilite.Intensity(time.Now(), st.LocationInfo.Lat, st.LocationInfo.Lon, st.LocationInfo.Timezone)
	switch st.GameMode {
	case state.ModeEasy, state.ModeNoisy:
		f.VisibilityRadius = f.FullVisibilityRadius()
	case state.ModeCrazy:
		switch st.NightOption {
		case state.NightNever:
			if f.Index < 0 {
				f.VisibilityRadius = minFloorVisibilityRadius
			} else {
				f.VisibilityRadius = f.FullVisibilityRadius()
			}
		case state.NightAlways:
			f.VisibilityRadius = minFloorVisibilityRadius
		case state.NightReal:
			if f.Index < 0 {
				f.VisibilityRadius = minFloorVisibilityRadius
			} else {
				f.VisibilityRadius = minFloorVisibilityRadius + int(float64(f.FullVisibilityRadius()-minFloorVisibilityRadius)*litIntensity)
			}
		}
	}
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
			m.quit = quit.New()
			return m, nil
		case "m": // mute/unmute
			m.state.SetMute(!m.state.Mute)
			return m, nil
		}
	case tea.WindowSizeMsg:
		// Always remember the latest terminal size
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		// Handle terminal resize by passing dimensions to the play model
		if m.status == statusGameplay {
			// Create a custom window size message for the play model
			playWindowSizeMsg := play.WindowSizeMsg{
				Width:  msg.Width,
				Height: msg.Height,
			}
			m.play, cmd = m.play.Update(playWindowSizeMsg)
			cmds = append(cmds, cmd)
		}
		// Force a full repaint by returning no cached content and clearing the screen
		cmds = append(cmds, tea.ClearScreen)
		return m, tea.Batch(cmds...)
	}

	switch m.status {
	case statusStartSplash:
		switch msg := msg.(type) {
		case splash.MakeSettingsMsg:
			m.status = statusDoSettings
			m.setup = setup.New(m.state.GameMode, m.state.NightOption, m.state.SpriteSize, m.state.Mute)
			m.state.SoundManager.StopListed(sound.INTRO)
		case splash.TimedoutMsg:
			m.status = statusGameplay
			m.resetPlayModel()
			m.state.SoundManager.StopListed(sound.INTRO)
		default:
			m.splash, cmd = m.splash.Update(msg)
		}
		cmds = append(cmds, cmd)
	case statusDoSettings:
		switch msg := msg.(type) {
		case setup.SaveSettingsMsg:
			m.status = statusGameplay
			// Preserve the sound manager as it's a global resource that persists across state resets.
			soundManager := m.state.SoundManager
			if msg.Reset {
				m.state = state.New()
			} else {
				m.state.GameMode = msg.Mode
				m.state.NightOption = msg.CrazyNight
				m.state.SpriteSize = msg.SpriteSize
				m.state.SetMute(msg.Mute)
			}
			m.state.SoundManager = soundManager
			m.resetForNewGame()
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
			m.state.SoundManager.Play(sound.TRANSITION_UP)
			m.status = statusFloorIntro
			nextFloorIndex := m.floor.Index + 1
			prevFloorEndPoint := m.floor.Maze.End()
			m.floor = getFloor(nextFloorIndex, m.state, m.floorCache, &prevFloorEndPoint, nil)
			startPoint := m.floor.Maze.Start()
			m.haunteed.SetPos(dweller.Position{X: startPoint.X, Y: startPoint.Y})
			m.haunteed.SetHome(dweller.Position{X: startPoint.X, Y: startPoint.Y})
			m.haunteed.SetHaunteedSprites(m.state.SpriteSize)
			m.next = next.New(nextFloorIndex)
		case play.PrevFloorMsg:
			m.state.SoundManager.Play(sound.TRANSITION_DOWN)
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
			m.next = next.New(prevFloorIndex)
		case play.RespawnMsg:
			setFloorVisibility(m.floor, m.state)
			m.status = statusRespawning
			m.respawn = respawn.New(msg.Lives)
		case play.GameOverMsg:
			m.status = statusGameOver
			score := msg.Score
			highScore := m.state.GetHighScore() // Get high score before update
			if err := m.state.UpdateAndSave(m.floor.Index, score, m.floor.Seed); err != nil {
				log.Fatal(err)
			}
			if score > highScore {
				m.state.SoundManager.PlayWithCallback(sound.HIGH_SCORE, func() {
					m.state.SoundManager.PlayLoop(sound.INTRO)
				})
			} else {
				m.state.SoundManager.PlayWithCallback(sound.GAME_OVER, func() {
					m.state.SoundManager.PlayLoop(sound.INTRO)
				})
			}
			// Pass the *old* high score to the 'over' model for correct message display
			m.over = over.New(m.state.GameMode, score, highScore)
		case play.VisibilityToggledMsg:
			m.floorVisibility[msg.FloorIndex] = msg.IsVisible
			return m, nil // State updated, no further action needed
		default:
			m.play, cmd = m.play.Update(msg)
		}
		cmds = append(cmds, cmd)
	case statusFloorIntro:
		switch msg := msg.(type) {
		case next.TimedoutMsg:
			m.resetPlayModel()
			m.status = statusGameplay
		default:
			m.next, cmd = m.next.Update(msg)
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
		switch msg := msg.(type) {
		case over.PlayAgainMsg:
			m.state.SoundManager.StopListed(sound.INTRO)
			m.resetForNewGame()
		case over.QuitGameMsg:
			m.status = statusQuitting
			m.state.SoundManager.StopListed(sound.INTRO)
			return m, tea.Quit
		default:
			m.over, cmd = m.over.Update(msg)
		}
		cmds = append(cmds, cmd)
	case statusQuitting:
		switch msg := msg.(type) {
		case quit.TimedoutMsg:
			return m, tea.Quit
		default:
			m.quit, cmd = m.quit.Update(msg)
		}
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) resetForNewGame() {
	// Reset for a new game
	m.floorCache = make(map[int]*floor.Floor)
	m.floorVisibility = make(map[int]bool)
	m.floor = getFloor(0, m.state, m.floorCache, nil, nil)
	startPos := dweller.Position{X: m.floor.Maze.Start().X, Y: m.floor.Maze.Start().Y}
	m.haunteed = dweller.PlaceHaunteed(m.state.SpriteSize, startPos)
	m.score.Reset()
	m.score.SetHigh(m.state.GetHighScore())
	m.resetPlayModel()
	m.status = statusGameplay
}

func (m *Model) resetPlayModel() {
	m.play = play.New(m.state, m.floor, m.score, m.haunteed, m.floorVisibility[m.floor.Index])
	// Seed the play model with the latest terminal size so it renders correctly before any manual resize
	if m.termWidth > 0 && m.termHeight > 0 {
		m.play, _ = m.play.Update(play.WindowSizeMsg{Width: m.termWidth, Height: m.termHeight})
	}
}

func (m *Model) resetPlayModelForRespawn() {
	// The play model holds the haunteed and ghosts. We need a new one to reset their positions.
	// The score and floor state are preserved.
	// We keep the current haunteed instance because it tracks lives.
	m.haunteed.SetPos(m.haunteed.Home())
	// Create a new play model, which will re-place ghosts.
	m.play = play.New(m.state, m.floor, m.score, m.haunteed, m.floorVisibility[m.floor.Index])
	// Seed size immediately
	if m.termWidth > 0 && m.termHeight > 0 {
		m.play, _ = m.play.Update(play.WindowSizeMsg{Width: m.termWidth, Height: m.termHeight})
	}
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
		return m.next.View()
	case statusRespawning:
		return m.respawn.View()
	case statusGameOver:
		return m.over.View()
	case statusQuitting:
		return m.quit.View()
	}
	return ""
}
