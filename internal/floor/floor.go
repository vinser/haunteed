package floor

import (
	"errors"
	"log"
	"math/rand"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/vinser/haunteed/internal/state"
	"github.com/vinser/haunteed/internal/style"
	"github.com/vinser/maze"
)

// ItemType represents a type of cell in the maze.
type ItemType int

const (
	Wall ItemType = iota
	Dot
	Empty
	PowerPellet
	Start
	End
	Fuse
)

type Floor struct {
	Index             int
	Seed              int64
	Maze              *maze.Maze
	Items             [][]ItemType // items placed in the maze grid
	GhostTickInterval time.Duration
	Sprites           map[ItemType][]string
	DimFuseSprite     []string
	VisibilityRadius  int
}

const (
	// Maze settings
	Width  = 21
	Height = 15
	// Ghosts' den size
	DenWidth  = 5
	DenHeight = 3
	// Maze generation Bias difines maze complexity
	Bias = 0.2
)

// New initializes a new floor with its configuration and dot count.
func New(index int, seed int64, startPoint, endPoint *maze.Point, spriteSize, gameMode, crazyNight string) *Floor {
	// If no seed is provided, the behavior will be deterministic for a given index.
	if seed == 0 {
		seed = int64(index)
	}
	rng := rand.New(rand.NewSource(seed))
	m, err := maze.New(Width, Height, DenWidth, DenHeight)
	if err != nil {
		log.Fatal(err)
	}
	m.Generate(seed, startPoint, endPoint, nil, "top", Bias)

	items := newItems(m)

	solution, ok := m.Solve()
	if !ok {
		log.Fatalf("no solution for width=%d, height=%d, denWidth=%d, denHeight=%d, seed=%d", Width, Height, DenWidth, DenHeight, seed)
	}
	solution = solution[1 : len(solution)-1]
	items = placeDots(items, solution)
	pelNum := rng.Intn(2) + 4 // 4-5 power pellets per floor
	items = placePowerPellets(items, m, pelNum)
	// In "Crazy" mode, a fuse is placed on every floor.
	if gameMode == state.ModeCrazy {
		items = placeFuse(items, m, rng)
	}

	sprites, dimFuseSprite := setFloorSprites(index, spriteSize, gameMode)
	return &Floor{
		Index:             index,
		Seed:              seed,
		Maze:              m,
		Items:             items,
		GhostTickInterval: ghostInterval(index),
		Sprites:           sprites,
		DimFuseSprite:     dimFuseSprite,
	}
}

// ItemAt returns the tile at the specified coordinates.
func (f *Floor) ItemAt(x, y int) (ItemType, error) {
	if x < 0 || x >= f.Maze.Width() || y < 0 || y >= f.Maze.Height() {
		return Empty, errors.New("out of bounds")
	}
	return f.Items[y][x], nil
}

// EatItem replaces a dot or power pellet with empty space and returns the eaten tile type.
func (f *Floor) EatItem(x, y int) ItemType {
	if x < 0 || x >= f.Maze.Width() || y < 0 || y >= f.Maze.Height() {
		return Empty
	}
	originalTile := f.Items[y][x]
	if originalTile == Dot || originalTile == PowerPellet {
		f.Items[y][x] = Empty
	}
	return originalTile
}

// RenderAt renders the tile at the specified coordinates using the given sprite size.
func (f *Floor) RenderAt(x, y int) []string {
	item, _ := f.ItemAt(x, y)
	sprite, ok := f.Sprites[item]
	if !ok {
		// Fallback for any unmapped item types.
		sprite = f.Sprites[Empty]
	}
	return sprite
}

func setFloorSprites(floorNum int, spriteSize, gameMode string) (map[ItemType][]string, []string) {
	var sprites = map[ItemType][]string{
		Wall:        nil,
		Dot:         nil,
		PowerPellet: nil,
		Start:       nil,
		End:         nil,
		Fuse:        nil,
		Empty:       nil,
	}
	var dimFuseSprite []string

	for item := range sprites {
		brightStyle, dimStyle := getFloorItemStyle(floorNum, item)
		var sprite []string
		for _, s := range getFloorSprite(spriteSize, gameMode, item) {
			if item == Fuse {
				sprite = append(sprite, brightStyle.Bold(true).Render(s))
				continue
			}
			sprite = append(sprite, brightStyle.Render(s))
		}
		sprites[item] = sprite

		if item == Fuse {
			for _, s := range getFloorSprite(spriteSize, gameMode, item) {
				dimFuseSprite = append(dimFuseSprite, dimStyle.Render(s))
			}
		}
	}
	return sprites, dimFuseSprite
}

func getFloorItemStyle(floorNum int, item ItemType) (brightStyle, dimStyle lipgloss.Style) {
	var color style.RGB
	switch item {
	case Wall:
		color = style.RGBColor["white"]
	case Dot:
		color = style.RGBColor["white"]
	case PowerPellet:
		color = style.RGBColor["white"]
	case Start:
		color = style.RGBColor["red"]
	case End:
		color = style.RGBColor["green"]
	case Fuse:
		color = style.RGBColor["yellow"]
	default:
		color = style.RGBColor["white"]
	}

	brightR, dimR := style.FloorColorShift(color.R, floorNum)
	brightG, dimG := style.FloorColorShift(color.G, floorNum)
	brightB, dimB := style.FloorColorShift(color.B, floorNum)

	dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(style.GenerateHexColor(dimR, dimG, dimB)))
	brightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(style.GenerateHexColor(brightR, brightG, brightB)))

	return brightStyle, dimStyle
}

func getFloorSprite(size string, mode string, item ItemType) []string {
	switch size {
	case state.SpriteSmall:
		switch item {
		case Wall:
			return []string{"░"}
		case Dot:
			if mode == state.ModeNoisy {
				return []string{" "}
			}
			return []string{"⋅"}
		case PowerPellet:
			return []string{"∘"}
		case Start:
			return []string{"▾"}
		case End:
			return []string{"▴"}
		case Fuse:
			return []string{"↯"}
		default:
			return []string{" "}
		}
	case state.SpriteMedium:
		switch item {
		case Wall:
			return []string{"▒▒"}
		case Dot:
			if mode == state.ModeNoisy {
				return []string{"  "}
			}
			return []string{"╺╸"}
		case PowerPellet:
			return []string{"◀▶"}
		case Start:
			return []string{"◥◤"}
		case End:
			return []string{"◢◣"}
		case Fuse:
			return []string{"↯↯"}
		default:
			return []string{"  "}
		}
	case state.SpriteLarge:
		switch item {
		case Wall:
			return []string{"░░░░", "░░░░"}
		case Dot:
			if mode == state.ModeNoisy {
				return []string{"    ", "    "}
			}
			return []string{" ▗▖ ", " ▝▘ "}
		case PowerPellet:
			return []string{" ▛▜ ", " ▙▟ "}
		case Start:
			return []string{" ◥◤ ", " ◥◤ "}
		case End:
			return []string{" ◢◣ ", " ◢◣ "}
		case Fuse:
			return []string{" ↯↯ ", " ↯↯ "}
		default:
			return []string{"    ", "    "}
		}
	}

	return []string{" "}
}
