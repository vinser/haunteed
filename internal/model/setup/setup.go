package setup

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vinser/haunteed/internal/render"
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

func New(mode, crazyNight, spriteSize string, mute bool, width, height int) Model {
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
			return m, saveSettingsCmd(m.mode, m.crazyNight, m.spriteSize, m.mute, m.reset)
		case "esc":
			return m, discardSettingsCmd()
		case "up":
			if m.selectedSetting > 0 {
				m.selectedSetting--
			}
			return m, nil
		case "down":
			if m.selectedSetting < numSettings-1 {
				m.selectedSetting++
			}
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
	}

	options := []option{{"Game mode", m.mode}}
	if m.mode == state.ModeCrazy {
		options = append(options, option{"Night lighting", m.crazyNight})
	}
	options = append(options, option{"Sprite size", m.spriteSize})
	options = append(options, option{"Mute all sounds", checkBox(m.mute)})
	options = append(options, option{"Reset progress", checkBox(m.reset)})
	// Calculate max widths and format string
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
	return b.String()
}

func checkBox(value bool) string {
	if value {
		return "[▪]"
	}
	return "[ ]"
}
