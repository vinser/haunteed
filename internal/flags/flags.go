package flags

import (
	"fmt"
	"os"
	"strings"
)

// Flags stores the parsed command-line options
type Flags struct {
	Mode   string
	Night  string
	Sprite string
	Mute   bool
	Reset  bool
}

// Parse parses command-line flags and returns the resulting config
func Parse() (*Flags, bool) {
	// Custom flag variables
	var mode string
	var night string
	var sprite string
	var mute bool
	var reset bool

	// Create custom FlagSet to allow custom usage output
	fs := NewFlagSetWithVisit()

	// Define flags with both short and long forms
	fs.StringVar(&mode, "game-mode", "g", "", "Game mode: easy (default), noisy, or crazy")
	fs.StringVar(&night, "night-option", "n", "", "Night option for crazy mode: never, always or real (default)")
	fs.StringVar(&sprite, "sprite-size", "s", "", "Sprite size: small, medium (default), or large")
	fs.BoolVar(&mute, "mute", "m", false, "Mute all sounds")
	fs.BoolVar(&reset, "reset", "r", false, "Reset saved progress and settings")

	// Parse command-line flags
	fs.Parse(os.Args[1:])

	if !fs.HasCustom() {
		return nil, false
	}

	// Normalize mode value
	if fs.IsCustom("game-mode") {
		mode = strings.ToLower(mode)
		if mode != "easy" && mode != "noisy" && mode != "crazy" && mode != "test" {
			fmt.Fprintf(os.Stderr, "Invalid game mode: %s. Use 'easy', 'noisy', or 'crazy'.\n", mode)
			fs.Usage()
			os.Exit(1)
		}
	}

	// Normalize crazy-night value
	if fs.IsCustom("night-option") {
		night = strings.ToLower(night)
		if night != "never" && night != "always" && night != "real" {
			fmt.Fprintf(os.Stderr, "Invalid night option: %s. Use 'never', 'always', or 'real'.\n", night)
			fs.Usage()
			os.Exit(1)
		}
	}

	// Normalize sprite size value
	if fs.IsCustom("sprite-size") {
		sprite = strings.ToLower(sprite)
		if sprite != "" && sprite != "small" && sprite != "medium" && sprite != "large" {
			fmt.Fprintf(os.Stderr, "Invalid sprite size: %s. Use 'small', 'medium' or 'large'.\n", sprite)
			fs.Usage()
			os.Exit(1)
		}
	}

	return &Flags{
		Mode:   mode,
		Night:  night,
		Sprite: sprite,
		Mute:   mute,
		Reset:  reset,
	}, true
}
