package flags

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// Flags stores the parsed command-line options
type Flags struct {
	Mode       string
	CrazyNight string
	SpriteSize string
	Mute       bool
	Reset      bool
}

// Parse parses command-line flags and returns the resulting config
func Parse() (*Flags, bool) {
	// Custom flag variables
	var mode string
	var crazyNight string
	var spriteSize string
	var mute bool
	var reset bool

	// Create custom FlagSet to allow custom usage output
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Define flags with both short and long forms
	fs.StringVar(&mode, "mode", "", "Game mode: easy, noisy, or crazy")
	fs.StringVar(&crazyNight, "crazy-night", "", "Select night option for crazy mode: never, always or real")
	fs.StringVar(&spriteSize, "sprite-size", "", "Sprite size: small, medium, or large")
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
		fmt.Fprintf(os.Stderr, "Invalid mode: %s. Use 'easy', 'noisy', or 'crazy'.\n", mode)
		fs.Usage()
		os.Exit(1)
	}

	// Normalize crazy-night value
	if mode != "crazy" {
		if crazyNight != "" {
			fmt.Fprintf(os.Stderr, "Crazy night option is only valid for 'crazy' mode.\n")
			fs.Usage()
			os.Exit(1)
		}
	} else {
		crazyNight = strings.ToLower(crazyNight)
		if crazyNight != "never" && crazyNight != "always" && crazyNight != "real" {
			fmt.Fprintf(os.Stderr, "Invalid crazyNight: %s. Use 'never', 'always', or 'real'.\n", crazyNight)
			fs.Usage()
			os.Exit(1)
		}
	}

	// Normalize sprite size value
	spriteSize = strings.ToLower(spriteSize)
	if spriteSize != "" && spriteSize != "small" && spriteSize != "medium" && spriteSize != "large" {
		fmt.Fprintf(os.Stderr, "Invalid sprite-size: %s. Use 'small', 'medium' or 'large'.\n", spriteSize)
		fs.Usage()
		os.Exit(1)
	}

	return &Flags{
		Mode:       mode,
		CrazyNight: crazyNight,
		SpriteSize: spriteSize,
		Mute:       mute,
		Reset:      reset,
	}, true
}
