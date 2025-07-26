package flags

import (
	"flag"
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
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Define flags with both short and long forms
	fs.StringVar(&mode, "mode", "easy", "Game mode: easy, noisy, or crazy")
	fs.StringVar(&night, "crazy-night", "", "Select night option for crazy mode: never, always or real")
	fs.StringVar(&sprite, "sprite-size", "medium", "Sprite size: small, medium, or large")
	fs.BoolVar(&mute, "mute", false, "Mute all sounds")
	fs.BoolVar(&reset, "reset", false, "Reset saved progress and settings")

	// Override the usage output
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fs.PrintDefaults()
	}
	// Parse command-line flags
	fs.Parse(os.Args[1:])

	// Check if any flags have been set non-default
	hasCustom := false
	fs.Visit(func(f *flag.Flag) {
		if f.Value.String() != f.DefValue {
			hasCustom = true
		}
	})
	if !hasCustom {
		return nil, false
	}

	// Normalize mode value
	mode = strings.ToLower(mode)
	if mode != "easy" && mode != "noisy" && mode != "crazy" && mode != "test" {
		fmt.Fprintf(os.Stderr, "Invalid game mode: %s. Use 'easy', 'noisy', or 'crazy'.\n", mode)
		fs.Usage()
		os.Exit(1)
	}

	// Normalize crazy-night value
	if mode != "crazy" {
		if night != "" {
			fmt.Fprintf(os.Stderr, "Night option is only valid for 'crazy' mode.\n")
			fs.Usage()
			os.Exit(1)
		}
	} else {
		night = strings.ToLower(night)
		if night != "never" && night != "always" && night != "real" {
			fmt.Fprintf(os.Stderr, "Invalid night option: %s. Use 'never', 'always', or 'real'.\n", night)
			fs.Usage()
			os.Exit(1)
		}
	}

	// Normalize sprite size value
	sprite = strings.ToLower(sprite)
	if sprite != "" && sprite != "small" && sprite != "medium" && sprite != "large" {
		fmt.Fprintf(os.Stderr, "Invalid sprite size: %s. Use 'small', 'medium' or 'large'.\n", sprite)
		fs.Usage()
		os.Exit(1)
	}

	return &Flags{
		Mode:   mode,
		Night:  night,
		Sprite: sprite,
		Mute:   mute,
		Reset:  reset,
	}, true
}
