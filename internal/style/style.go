package style

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	// General UI
	SplashDot      = lipgloss.NewStyle().Foreground(lipgloss.Color("255")) // Bright white
	SplashHaunteed = lipgloss.NewStyle().Foreground(lipgloss.Color("226")) // Bright yellow
	SplashGhosts   = []lipgloss.Style{
		lipgloss.NewStyle().Foreground(lipgloss.Color("9")),  // Bright red
		lipgloss.NewStyle().Foreground(lipgloss.Color("13")), // Bright magenta
		lipgloss.NewStyle().Foreground(lipgloss.Color("14")), // Bright cyan
		lipgloss.NewStyle().Foreground(lipgloss.Color("10")), // Bright green
	}

	SetupTitle        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("228")) // Bright yellow
	SetupItem         = lipgloss.NewStyle()
	SetupItemSelected = lipgloss.NewStyle().Foreground(lipgloss.Color("204")).Bold(true) // Pinkish-reddish purple
	PlayHeader        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("82"))  // Green

	HighScore = lipgloss.NewStyle().Foreground(lipgloss.Color("9")) // Bright red
	// Page styles
	TopPattern = lipgloss.NewStyle().Foreground(lipgloss.Color("204"))            // Pinkish-reddish purple
	Title      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("228")) // Bright yellow
	Content    = lipgloss.NewStyle()
	Footer     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

type RGB struct {
	R int
	G int
	B int
}

var RGBColor = map[string]RGB{
	"black":   {0, 0, 0},
	"red":     {255, 0, 0},
	"green":   {0, 255, 0},
	"blue":    {0, 0, 255},
	"yellow":  {255, 255, 0},
	"magenta": {255, 0, 255},
	"cyan":    {0, 255, 255},
	"white":   {255, 255, 255},
	"grey":    {128, 128, 128},
	"brown":   {165, 42, 42},
}

// GenerateHexColor generates hexadcimal string for a given RGB values. r, g, b sould be in the range 0-255
// Format: #RRGGBB
func GenerateHexColor(r, g, b int) string {
	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

const (
	brightMin    = 128
	brightGround = 64
	dimShift     = 64
	dimStep      = 16
)

func FloorColorShift(colorNum, floorNum int) (int, int) {
	if colorNum == 0 {
		return 0, 0
	}
	bright := colorNum - brightGround + floorNum*dimStep
	if bright > 255 {
		bright = 255
	}
	if bright < brightMin {
		bright = brightMin
	}
	dim := bright - dimShift
	return bright, dim
}
