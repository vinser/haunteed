package app

import (
	"log"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vinser/haunteed/internal/ambilite"
	"github.com/vinser/haunteed/internal/dweller"
	"github.com/vinser/haunteed/internal/flags"
	"github.com/vinser/haunteed/internal/floor"
	"github.com/vinser/haunteed/internal/geoip"
	"github.com/vinser/haunteed/internal/model/about"
	"github.com/vinser/haunteed/internal/model/bosskey"
	"github.com/vinser/haunteed/internal/model/next"
	"github.com/vinser/haunteed/internal/model/over"
	"github.com/vinser/haunteed/internal/model/play"
	"github.com/vinser/haunteed/internal/model/quit"
	"github.com/vinser/haunteed/internal/model/respawn"
	"github.com/vinser/haunteed/internal/model/setup"
	"github.com/vinser/haunteed/internal/model/splash"
	"github.com/vinser/haunteed/internal/score"
	"github.com/vinser/haunteed/internal/sound"
	"github.com/vinser/haunteed/internal/state"
	"github.com/vinser/maze"
)

type status uint

const (
	statusStartSplash status = iota
	statusDoSettings
	statusAbout
	statusGameplay
	statusFloorIntro
	statusRespawning
	statusGameOver
	statusQuitting
)

type Model struct {
	status          status
	state           *state.State
	soundManager    *sound.Manager
	floorCache      map[int]*floor.Floor
	floorVisibility map[int]bool // Persists visibility state for "Crazy" mode across floors
	haunteed        *dweller.Haunteed
	floor           *floor.Floor
	score           *score.Score
	// models
	splash         splash.Model
	setup          setup.Model
	about          about.Model
	play           play.Model
	next           next.Model
	respawn        respawn.Model
	over           over.Model
	quit           quit.Model
	bosskey        bosskey.Model
	bosskeyVisible bool
	// terminal size cache
	termWidth  int
	termHeight int
}

func New(version string) Model {
	// Configure global settings first to ensure consistent behavior.
	geoip.SetCacheTTL(0) // Ensure fresh location data for new sessions.

	soundMgr, soundInitFailed := sound.Initialize()

	state := getState(version)
	if soundInitFailed {
		state.Mute = true
	}
	if state.Mute {
		soundMgr.Mute()
	} else {
		soundMgr.Unmute()
	}

	splash := setSplash(state)
	floorCache := make(map[int]*floor.Floor)
	initialFloor := getFloor(0, state, floorCache, nil, nil)
	startPos := dweller.Position{X: initialFloor.Maze.Start().X, Y: initialFloor.Maze.Start().Y}
	haunteed := dweller.PlaceHaunteed(state.SpriteSize, state.GameMode, startPos)
	score := score.NewScore()
	if highScores := state.GetHighScores(); len(highScores) > 0 {
		score.SetHigh(highScores[0].Score)
		score.SetNick(highScores[0].Nick)
	}
	return Model{
		status:          statusStartSplash,
		state:           state,
		soundManager:    soundMgr,
		floorCache:      floorCache,
		floorVisibility: make(map[int]bool),
		haunteed:        haunteed,
		floor:           initialFloor,
		score:           score,
		splash:          splash,
		bosskey:         bosskey.New(soundMgr),
	}
}

func getState(appVersion string) *state.State {
	st := state.Load(appVersion)
	if fl, ok := flags.Parse(); ok {
		if fl.Version {
			log.Printf("Haunteed version: %s\n", appVersion)
			os.Exit(0)
		}
		if fl.Reset {
			state.Reset()
			return state.New(appVersion)
		}

		if fl.Mute {
			st.Mute = true
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
	width, height := getDefaultWidthHeight()
	model := splash.New(st, width, height)
	return model
}

func setSetup(st *state.State, sm *sound.Manager) setup.Model {
	width, height := getDefaultWidthHeight()
	model := setup.New(st.GameMode, st.NightOption, st.SpriteSize, st.Mute, width, height, sm)
	return model
}

func setAbout(st *state.State) about.Model {
	width, height := getDefaultWidthHeight()
	model := about.New(st, width, height)
	return model
}

func setRespawn(st *state.State, lives int) respawn.Model {
	width, height := getDefaultWidthHeight()
	model := respawn.New(lives, width, height)
	return model
}

func setNext(st *state.State, index int) next.Model {
	width, height := getDefaultWidthHeight()
	model := next.New(index, width, height)
	return model
}

func (m *Model) setGameOver(score int) over.Model {
	highScores := m.state.GetHighScores()

	highScore := 0
	if len(highScores) > 0 {
		highScore = highScores[0].Score
	}

	if score > highScore {
		m.soundManager.PlayWithCallback(sound.HIGH_SCORE, func() {
			m.soundManager.PlayLoop(sound.INTRO)
		})
	} else {
		m.soundManager.PlayWithCallback(sound.GAME_OVER, func() {
			m.soundManager.PlayLoop(sound.INTRO)
		})
	}
	width, height := getDefaultWidthHeight()
	model := over.New(m.state, score, highScores, width, height)
	return model
}

func (m *Model) setQuit() quit.Model {
	m.soundManager.StopAll()
	m.soundManager.Play(sound.QUIT)
	width, height := getDefaultWidthHeight()
	model := quit.New(width, height)
	return model
}

func getDefaultWidthHeight() (int, int) {
	return getWidthHeight(&state.State{})
}

func getWidthHeight(st *state.State) (int, int) {
	mazeWidth, mazeHeight := getMazeDimensions(st.GameMode)
	var spriteWidth, spriteHeight int
	switch st.SpriteSize {
	case state.SpriteSmall:
		spriteWidth, spriteHeight = 1, 1
	case state.SpriteMedium:
		spriteWidth, spriteHeight = 2, 1
	case state.SpriteLarge:
		spriteWidth, spriteHeight = 4, 2
	default:
		spriteWidth, spriteHeight = 2, 1 // Default to medium
	}

	width := mazeWidth * spriteWidth
	height := mazeHeight * spriteHeight
	return width, height
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
	width, height := getMazeDimensions(st.GameMode)
	f := floor.New(index, st.FloorSeeds[index], startPoint, endPoint, width, height, st.SpriteSize, st.GameMode, st.NightOption)

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

func getMazeDimensions(gameMode string) (width, height int) {
	switch gameMode {
	case state.ModeEasy:
		return floor.ModeEasyWidth, floor.ModeEasyHeight
	case state.ModeNoisy:
		return floor.ModeNoisyWidth, floor.ModeNoisyHeight
	case state.ModeCrazy:
		return floor.ModeCrazyWidth, floor.ModeCrazyHeight
	default: // state.ModeNoisy
		return floor.ModeNoisyWidth, floor.ModeNoisyHeight
	}
}

func (m Model) Init() tea.Cmd {
	m.soundManager.PlayLoop(sound.INTRO)
	return tea.Batch(m.splash.Init(), m.over.Init(), tea.DisableMouse)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.bosskeyVisible {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "b" || msg.String() == "B" {
				m.bosskeyVisible = false
				switch m.status {
				case statusStartSplash:
					m.soundManager.PlayLoop(sound.INTRO)
					return m, m.splash.Init()
				case statusDoSettings:
					return m, m.setup.Init()
				case statusAbout:
					return m, m.about.Init()
				case statusGameplay:
					return m, m.play.Init()
				case statusFloorIntro:
					return m, m.next.Init()
				case statusRespawning:
					return m, m.respawn.Init()
				case statusGameOver:
					m.soundManager.PlayLoop(sound.INTRO)
					return m, m.over.Init()
				case statusQuitting:
					return m, m.quit.Init()
				default:
					return m, nil
				}
			}
		case bosskey.TickMsg:
			newBosskey, cmd := m.bosskey.Update(msg)
			m.bosskey = newBosskey
			return m, cmd
		}
		return m, nil
	}

	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "b", "B":
			m.bosskeyVisible = true
			m.bosskey.SetSize(m.termWidth, m.termHeight)
			m.soundManager.StopAll()
			return m, m.bosskey.Init()
		case "ctrl+c", "q", "Q": // quit all app models
			m.status = statusQuitting // Set status to show quit message
			m.quit = m.setQuit()
			m.quit.SetSize(m.termWidth, m.termHeight)
			return m, m.quit.Init()
		case "m", "M": // mute/unmute
			m.state.Mute = !m.state.Mute
			if m.state.Mute {
				m.soundManager.Mute()
			} else {
				m.soundManager.Unmute()
			}
			return m, nil
		}
	case tea.WindowSizeMsg:
		// Always remember the latest terminal size
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		m.bosskey.SetSize(msg.Width, msg.Height)
		// Handle terminal resize by passing dimensions to the play model
		switch m.status {
		case statusStartSplash:
			m.splash.SetSize(msg.Width, msg.Height)
		case statusDoSettings:
			m.setup.SetSize(msg.Width, msg.Height)
		case statusAbout:
			m.about.SetSize(msg.Width, msg.Height)
		case statusGameplay:
			// Create a custom window size message for the play model
			playWindowSizeMsg := play.WindowSizeMsg{
				Width:  msg.Width,
				Height: msg.Height,
			}
			m.play, cmd = m.play.Update(playWindowSizeMsg)
			cmds = append(cmds, cmd)
		case statusFloorIntro:
			m.next.SetSize(msg.Width, msg.Height)
		case statusRespawning:
			m.respawn.SetSize(msg.Width, msg.Height)
		case statusGameOver:
			m.over.SetSize(msg.Width, msg.Height)
		case statusQuitting:
			m.quit.SetSize(msg.Width, msg.Height)
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
			m.setup = setSetup(m.state, m.soundManager)
			m.setup.SetSize(m.termWidth, m.termHeight)
			m.soundManager.StopListed(sound.INTRO)
		case splash.TimedoutMsg:
			m.status = statusGameplay
			m.resetPlayModel()
			m.soundManager.StopListed(sound.INTRO)
			cmd = m.play.Init()
		default:
			m.splash, cmd = m.splash.Update(msg)
		}
		cmds = append(cmds, cmd)
	case statusDoSettings:
		switch msg := msg.(type) {
		case setup.ViewAboutMsg:
			m.status = statusAbout
			m.about = setAbout(m.state)
			m.about.SetSize(m.termWidth, m.termHeight)
		case setup.SaveSettingsMsg:
			m.status = statusGameplay
			if msg.Reset {
				state.Reset()
				m.state = state.New(m.state.Version)
			} else {
				m.state.GameMode = msg.Mode
				m.state.NightOption = msg.CrazyNight
				m.state.SpriteSize = msg.SpriteSize
				m.state.Mute = msg.Mute
			}
			if m.state.Mute {
				m.soundManager.Mute()
			} else {
				m.soundManager.Unmute()
			}
			m.resetForNewGame()
			cmd = m.play.Init()
		case setup.DiscardSettingsMsg:
			m.status = statusGameplay
			m.resetPlayModel()
			cmd = m.play.Init()
		default:
			m.setup, cmd = m.setup.Update(msg)
		}
		cmds = append(cmds, cmd)
	case statusAbout:
		switch msg := msg.(type) {
		case about.CloseAboutMsg:
			m.status = statusDoSettings
			m.setup = setSetup(m.state, m.soundManager)
			m.setup.SetSize(m.termWidth, m.termHeight)
		default:
			m.about, cmd = m.about.Update(msg)
		}
		cmds = append(cmds, cmd)
	case statusGameplay:
		switch msg := msg.(type) {
		case play.NextFloorMsg:
			m.soundManager.Play(sound.TRANSITION_UP)
			m.status = statusFloorIntro
			nextFloorIndex := m.floor.Index + 1
			prevFloorEndPoint := m.floor.Maze.End()
			m.floor = getFloor(nextFloorIndex, m.state, m.floorCache, &prevFloorEndPoint, nil)
			startPoint := m.floor.Maze.Start()
			m.haunteed.SetPos(dweller.Position{X: startPoint.X, Y: startPoint.Y})
			m.haunteed.SetHome(dweller.Position{X: startPoint.X, Y: startPoint.Y})
			m.haunteed.SetHaunteedSprites(m.state.SpriteSize)
			m.next = setNext(m.state, nextFloorIndex)
			m.next.SetSize(m.termWidth, m.termHeight)
		case play.PrevFloorMsg:
			m.soundManager.Play(sound.TRANSITION_DOWN)
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
			m.next = setNext(m.state, prevFloorIndex)
			m.next.SetSize(m.termWidth, m.termHeight)
		case play.RespawnMsg:
			setFloorVisibility(m.floor, m.state)
			m.status = statusRespawning
			m.respawn = setRespawn(m.state, msg.Lives)
			m.respawn.SetSize(m.termWidth, m.termHeight)
			cmd = m.respawn.Init()
		case play.GameOverMsg:
			m.status = statusGameOver
			score := msg.Score
			m.over = m.setGameOver(score)
			m.over.SetSize(m.termWidth, m.termHeight)
			cmd = m.over.Init()
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
			cmd = m.play.Init()
		default:
			m.next, cmd = m.next.Update(msg)
		}
		cmds = append(cmds, cmd)
	case statusRespawning:
		switch msg := msg.(type) {
		case respawn.TimedoutMsg:
			m.resetPlayModel()
			m.haunteed.SetPos(m.haunteed.Home())
			m.status = statusGameplay
			cmd = m.play.Init()
		default:
			m.respawn, cmd = m.respawn.Update(msg)
		}
		cmds = append(cmds, cmd)
	case statusGameOver:
		switch msg := msg.(type) {
		case over.SaveHighScoreMsg:
			if err := m.state.UpdateAndSave(m.floor.Index, m.score.Get(), m.floor.Seed, msg.Nick); err != nil {
				log.Fatal(err)
			}
			m.over.SetHighScores(m.state.GetHighScores())
		case over.PlayAgainMsg:
			m.soundManager.StopListed(sound.INTRO)
			m.resetForNewGame()
			m.status = statusGameplay
			cmd = m.play.Init()
		case over.QuitGameMsg:
			m.status = statusQuitting
			m.soundManager.StopListed(sound.INTRO)
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
	m.haunteed = dweller.PlaceHaunteed(m.state.SpriteSize, m.state.GameMode, startPos)
	m.score.Reset()
	if highScores := m.state.GetHighScores(); len(highScores) > 0 {
		m.score.SetHigh(highScores[0].Score)
		m.score.SetNick(highScores[0].Nick)
	}
	m.resetPlayModel()
}

func (m *Model) resetPlayModel() {
	m.play = play.New(m.state, m.soundManager, m.floor, m.score, m.haunteed, m.floorVisibility[m.floor.Index])
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
	m.play = play.New(m.state, m.soundManager, m.floor, m.score, m.haunteed, m.floorVisibility[m.floor.Index])
	// Seed size immediately
	if m.termWidth > 0 && m.termHeight > 0 {
		m.play, _ = m.play.Update(play.WindowSizeMsg{Width: m.termWidth, Height: m.termHeight})
	}
}

func (m Model) View() string {
	if m.bosskeyVisible {
		return m.bosskey.View()
	}
	switch m.status {
	case statusStartSplash:
		return m.splash.View()
	case statusDoSettings:
		return m.setup.View()
	case statusAbout:
		return m.about.View()
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
