package setup

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vinser/haunteed/internal/state"
	"github.com/vinser/haunteed/internal/style"
)

const (
	width  = 80
	height = 15

	selectedMode = iota
	selectedCrazyNight
	selectedSpriteSize
	selectedMute
	selectedReset
)

type Model struct {
	mode       string // easy, noisy or crazy
	crazyNight string // never, always or real (at location)
	spriteSize string // small, medium or large
	mute       bool
	reset      bool

	selectedSetting int
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

func New(mode, crazyNight, spriteSize string, mute bool) Model {
	return Model{
		mode:       mode,
		crazyNight: crazyNight,
		spriteSize: spriteSize,
		mute:       mute,
		reset:      false,

		selectedSetting: 0,
	}
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
		case "s":
			return m, saveSettingsCmd(m.mode, m.crazyNight, m.spriteSize, m.mute, m.reset)
		case "esc":
			return m, discardSettingsCmd()
		case "up":
			if m.selectedSetting > 0 {
				m.selectedSetting--
			}
		case "down":
			if m.selectedSetting < numSettings-1 {
				m.selectedSetting++
			}
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
					if m.mode == state.ModeCrazy {
						m.reset = !m.reset
					}
				}

			}
		}
		return m, nil

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

func (m Model) View() string {
	type option struct {
		label string
		value string
	}

	options := []option{{"Game mode", m.mode}}
	if m.mode == state.ModeCrazy {
		options = append(options, option{"Night mode", m.crazyNight})
	}
	options = append(options, option{"Sprite size", m.spriteSize})
	options = append(options, option{"Mute all sounds", fmt.Sprintf("%v", m.mute)})
	options = append(options, option{"Reset progress", fmt.Sprintf("%v", m.reset)})

	var b strings.Builder
	title := style.SetupTitle.Render("Settings")
	b.WriteString("\n" + centerText(title) + "\n\n")

	for i, opt := range options {
		prefix := "  "
		if i == m.selectedSetting {
			prefix = "➤ "
		}
		line := fmt.Sprintf("%s%s: %s", prefix, opt.label, opt.value)
		if i == m.selectedSetting {
			b.WriteString(centerText(style.SetupItemSelected.Render(line)))
		} else {
			b.WriteString(centerText(style.SetupItem.Render(line)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n\n\n\n\n\n" + centerText("↑ ↓ — select, space — change, s — save, esc — cancel") + "\n")
	return b.String()
}

func centerText(text string) string {
	padding := (width - lipgloss.Width(text)) / 2
	return spaces(padding) + text
}

func spaces(n int) string {
	return fmt.Sprintf("%*s", n, "")
}
