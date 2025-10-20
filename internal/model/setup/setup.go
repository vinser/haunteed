package setup

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vinser/haunteed/internal/geoip"
	"github.com/vinser/haunteed/internal/render"
	"github.com/vinser/haunteed/internal/sound"
	"github.com/vinser/haunteed/internal/state"
	"github.com/vinser/haunteed/internal/style"
)

const (
	selectedMode = iota
	selectedCrazyNight
	selectedSpriteSize
	selectedMute
	selectedReset
)

type Model struct {
	width      int
	height     int
	termWidth  int
	termHeight int

	mode       string // easy, noisy or crazy
	crazyNight string // never, always or real (at location)
	spriteSize string // small, medium or large
	mute       bool
	reset      bool

	selectedSetting int
	soundManager    *sound.Manager
}

type ViewAboutMsg struct{}

func viewAboutCmd() tea.Cmd {
	return func() tea.Msg {
		return ViewAboutMsg{}
	}
}

type SaveSettingsMsg struct {
	Mode       string
	CrazyNight string
	SpriteSize string
	Mute       bool
	Reset      bool
}

func saveSettingsCmd(mode, crazyNight, spriteSize string, mute, reset bool) tea.Cmd {
	return func() tea.Msg {
		return SaveSettingsMsg{
			Mode:       mode,
			CrazyNight: crazyNight,
			SpriteSize: spriteSize,
			Mute:       mute,
			Reset:      reset,
		}
	}
}

type DiscardSettingsMsg struct{}

func discardSettingsCmd() tea.Cmd {
	return func() tea.Msg {
		return DiscardSettingsMsg{}
	}
}

func New(mode, crazyNight, spriteSize string, mute bool, width, height int, sm *sound.Manager) Model {
	if width < lipgloss.Width(footer) {
		width = lipgloss.Width(footer)
	}
	return Model{
		width:  width,
		height: height,

		mode:       mode,
		crazyNight: crazyNight,
		spriteSize: spriteSize,
		mute:       mute,
		reset:      false,

		selectedSetting: 0,
		soundManager:    sm,
	}
}

func (m *Model) SetSize(width, height int) {
	m.termWidth = width
	m.termHeight = height
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		numSettings := 4
		if m.mode == state.ModeCrazy {
			numSettings = 5
		}

		switch msg.String() {
		case "a":
			return m, viewAboutCmd()
		case "s":
			m.soundManager.Play(sound.UI_SAVE)
			return m, saveSettingsCmd(m.mode, m.crazyNight, m.spriteSize, m.mute, m.reset)
		case "esc":
			m.soundManager.Play(sound.UI_CANCEL)
			return m, discardSettingsCmd()
		case "up":
			if m.selectedSetting > 0 {
				m.selectedSetting--
			}
			m.soundManager.Play(sound.UI_CLICK)
			return m, nil
		case "down":
			if m.selectedSetting < numSettings-1 {
				m.selectedSetting++
			}
			m.soundManager.Play(sound.UI_CLICK)
			return m, nil
		case "enter", " ":
			if numSettings == 5 {
				switch m.selectedSetting {
				case 0:
					m.mode = nextMode(m.mode)
					// If mode changes away from crazy, reset night mode and selection
					if m.mode != state.ModeCrazy {
						m.crazyNight = "never"
					}
				case 1:
					m.crazyNight = nextCrazyNight(m.crazyNight)
				case 2:
					m.spriteSize = nextSpriteSize(m.spriteSize)
				case 3:
					// Toggle mute
					m.mute = !m.mute
				case 4:
					m.reset = !m.reset
				}
			} else {
				switch m.selectedSetting {
				case 0:
					m.mode = nextMode(m.mode)
					// If mode changes away from crazy, reset night mode and selection
					if m.mode != state.ModeCrazy {
						m.crazyNight = "never"
					}
				case 1:
					m.spriteSize = nextSpriteSize(m.spriteSize)
				case 2:
					// Toggle mute
					m.mute = !m.mute
				case 3:
					m.reset = !m.reset
				}

			}
			m.soundManager.Play(sound.UI_CLICK)
			return m, nil
		}

	}
	return m, nil
}

func nextMode(current string) string {
	switch current {
	case "easy":
		return "noisy"
	case "noisy":
		return "crazy"
	case "crazy":
		return "easy"
	default:
		return "easy"
	}
}

func nextCrazyNight(current string) string {
	switch current {
	case "never":
		return "always"
	case "always":
		return "real"
	case "real":
		return "never"
	default:
		return "never"
	}
}

func nextSpriteSize(current string) string {
	switch current {
	case "small":
		return "medium"
	case "medium":
		return "large"
	case "large":
		return "small"
	default:
		return "medium"
	}
}

const footer = "↑ ↓ — select, space — change, a — about, s — save, esc — cancel"

func (m Model) View() string {
	return render.Page("Settings", m.renderOptions(), footer, m.width, m.height, m.termWidth, m.termHeight)
}

func (m Model) renderOptions() string {
	type option struct {
		label string
		value string
		key   int
	}

	// Descriptions for each logical setting (by key)
	descriptions := map[int]string{
		selectedMode: `Choose your level of despair:
- easy: peaceful night, maybe too peaceful
- noisy: servers groan and fans whisper
- crazy: reality melts with uptime and caffeine.`,

		selectedCrazyNight: `Controls how much the upper floors fear the dark:
- never: eternal daylight — ignorance is bliss
- always: permanent night — the basement won
- real: follows your location — day, dusk, night, regret, repeat.`,

		selectedSpriteSize: `How big the horrors appear:
- small: plausible deniability
- medium: comfortably terrifying
- large: face-to-face with your mistakes.`,

		selectedMute: `Silence the datacenter… or at least pretend to.
Ghosts don’t need speakers anyway.`,

		selectedReset: `Erase your sins and start another night shift.
Heads up — ghosts never forget.`,
	}

	// Build option list based on current mode
	options := []option{{"Game mode", m.mode, selectedMode}}
	if m.mode == state.ModeCrazy {
		options = append(options, option{"Night shadows", m.crazyNight, selectedCrazyNight})
	}
	options = append(options,
		option{"Sprite size", m.spriteSize, selectedSpriteSize},
		option{"Mute all sounds", checkBox(m.mute), selectedMute},
		option{"Reset progress", checkBox(m.reset), selectedReset},
	)

	// Calculate maximum label/value widths for aligned layout
	maxLabel, maxValue := 10, 10
	for _, row := range options {
		if len(row.label) > maxLabel {
			maxLabel = len(row.label)
		}
		if len(row.value) > maxValue {
			maxValue = len(row.value)
		}
	}
	format := fmt.Sprintf("%%-2s%%-%ds:%%%ds", maxLabel, maxValue+2)

	var b strings.Builder

	// Render options
	for i, opt := range options {
		prefix := "  "
		if i == m.selectedSetting {
			prefix = "▶ "
		}
		line := fmt.Sprintf(format, prefix, opt.label, opt.value)
		if i == m.selectedSetting {
			b.WriteString(style.SetupItemSelected.Render(line))
		} else {
			b.WriteString(style.SetupItem.Render(line))
		}
		b.WriteString("\n")
	}

	// Gap between options and description
	gapLines := 3
	if m.mode == state.ModeCrazy {
		gapLines = 2
	}
	b.WriteString(strings.Repeat("\n", gapLines))

	// Determine which option is currently selected and show its description
	selectedKey := options[m.selectedSetting].key
	desc := descriptions[selectedKey]
	// If "Night shadows" option is selected and set to "real", check network-based location
	if selectedKey == selectedCrazyNight && m.crazyNight == "real" {
		loc, err := geoip.GetLocationInfo()
		if err != nil {
			// Network or lookup failed — fallback to Kansas City
			desc = "Alert: No network detected.\nYou've been placed in the endless corn maze — Kansas City, MO (CST).\nFind your way out before your DNS expires."
		} else if loc != nil {
			// Successful lookup — replace description with a ghostly message
			desc = fmt.Sprintf(
				"The ghosts have found your datacenter in %s, %s.\nThey’ve synced their shifts with your sunrise — good luck escaping daylight savings.",
				loc.City, loc.Country,
			)
		}
	}
	descLines := 0
	if desc != "" {
		// Simple ghostly style (gray italic text)
		descText := style.SetupDescription.Render(desc)
		b.WriteString(descText)
		descLines = len(strings.Split(desc, "\n"))
	}

	// Compute the maximum number of description lines
	maxDescLines := 0
	for _, d := range descriptions {
		lines := len(strings.Split(d, "\n"))
		if lines > maxDescLines {
			maxDescLines = lines
		}
	}

	// Normalize total height (consistent view regardless of mode)
	maxTotalLines := len(options) + gapLines + maxDescLines
	currentLines := len(options) + gapLines + descLines
	if padding := maxTotalLines - currentLines; padding > 0 {
		b.WriteString(strings.Repeat("\n", padding))
	}

	return b.String()
}

func checkBox(value bool) string {
	if value {
		return "[▪]"
	}
	return "[ ]"
}
