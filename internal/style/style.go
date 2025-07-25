package style

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	// General UI
	SplashDot      = lipgloss.NewStyle().Foreground(lipgloss.Color("220")) // Yellowish
	SplashHaunteed = lipgloss.NewStyle().Foreground(lipgloss.Color("226")) // Bright yellow

	SetupTitle        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("228"))
	SetupItem         = lipgloss.NewStyle()
	SetupItemSelected = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)

	PlayHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("82")) // Green

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
