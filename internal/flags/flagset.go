package flags

import (
	"flag"
	"fmt"
	"os"
	"sort"
)

type FlagSetWithVisit struct {
	fs       *flag.FlagSet
	visited  map[string]bool
	aliases  map[string]string // short name → long name
	usageMap map[string]string // long name → usage string
}

func NewFlagSetWithVisit() *FlagSetWithVisit {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	fsv := &FlagSetWithVisit{
		fs:       fs,
		visited:  make(map[string]bool),
		aliases:  make(map[string]string),
		usageMap: make(map[string]string),
	}

	// Override default usage
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fsv.printUsage()
	}

	return fsv
}

// Register a bool flag with optional short alias
func (fsv *FlagSetWithVisit) BoolVar(p *bool, name, short string, value bool, usage string) {
	fsv.fs.BoolVar(p, name, value, usage)
	if short != "" {
		fsv.aliases[short] = name
	}
	fsv.usageMap[name] = usage
}

// Register a string flag with optional short alias
func (fsv *FlagSetWithVisit) StringVar(p *string, name, short, value, usage string) {
	fsv.fs.StringVar(p, name, value, usage)
	if short != "" {
		fsv.aliases[short] = name
	}
	fsv.usageMap[name] = usage
}

// Expand short aliases and parse args
func (fsv *FlagSetWithVisit) Parse(args []string) error {
	args = fsv.expandAliases(args)
	err := fsv.fs.Parse(args)
	if err != nil {
		return err
	}
	fsv.fs.Visit(func(f *flag.Flag) {
		fsv.visited[f.Name] = true
	})
	return nil
}

// Replace short flags (e.g. -r) with full names (e.g. -reset)
func (fsv *FlagSetWithVisit) expandAliases(args []string) []string {
	var expanded []string
	for _, arg := range args {
		// Match: -r or -r=value
		if len(arg) >= 2 && arg[0] == '-' && arg[1] != '-' {
			eqIdx := -1
			for i := 1; i < len(arg); i++ {
				if arg[i] == '=' {
					eqIdx = i
					break
				}
			}
			name := arg[1:]
			value := ""
			if eqIdx != -1 {
				name = arg[1:eqIdx]
				value = arg[eqIdx:]
			}
			if full, ok := fsv.aliases[name]; ok {
				expanded = append(expanded, "-"+full+value)
			} else {
				expanded = append(expanded, arg)
			}
		} else {
			expanded = append(expanded, arg)
		}
	}
	return expanded
}

// Check if a specific flag was explicitly set
func (fsv *FlagSetWithVisit) IsCustom(name string) bool {
	return fsv.visited[name]
}

// Check if any non-default flags were set
func (fsv *FlagSetWithVisit) HasCustom() bool {
	hasCustom := false
	fsv.fs.Visit(func(f *flag.Flag) {
		if f.Value.String() != f.DefValue {
			hasCustom = true
		}
	})
	return hasCustom
}

func (fsv *FlagSetWithVisit) Usage() {
	fsv.fs.Usage()
}

func (fsv *FlagSetWithVisit) PrintDefaults() {
	fsv.fs.PrintDefaults()
}

// Print formatted usage with short aliases
func (fsv *FlagSetWithVisit) printUsage() {
	var names []string
	var nameLen int
	for name := range fsv.usageMap {
		names = append(names, name)
		if len(name) > nameLen {
			nameLen = len(name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		usage := fsv.usageMap[name]
		short := ""
		for s, full := range fsv.aliases {
			if full == name {
				short = s
				break
			}
		}
		if short != "" {
			fmt.Fprintf(os.Stderr, "  -%s, -%-*s\t%s\n", short, nameLen, name, usage)
		} else {
			fmt.Fprintf(os.Stderr, "      -%-*s\t%s\n", nameLen, name, usage)
		}
	}
}
