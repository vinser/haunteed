package floor

import (
	"errors"
	"log"
	"math"
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
	CrumblingWall
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

func (f *Floor) FullVisibilityRadius() int {
	return int(math.Sqrt(float64(f.Maze.Width()*f.Maze.Width() + f.Maze.Height()*f.Maze.Height())))
}

const (
	// Ghosts' den size
	DenWidth  = 5
	DenHeight = 3

	// Maze dimensions
	ModeEasyWidth   = 21
	ModeEasyHeight  = 15
	ModeNoisyWidth  = 31
	ModeNoisyHeight = 21
	ModeCrazyWidth  = 41
	ModeCrazyHeight = 25
)

// New initializes a new floor with its configuration and dot count.
func New(index int, seed int64, startPoint, endPoint *maze.Point, width, height int, spriteSize, gameMode, crazyNight string) *Floor {
	// Determine maze dimensions based on game mode
	switch gameMode {
	case state.ModeNoisy:
		width, height = ModeNoisyWidth, ModeNoisyHeight
	case state.ModeCrazy:
		width, height = ModeCrazyWidth, ModeCrazyHeight
	default: // state.ModeEasy
		width, height = ModeEasyWidth, ModeEasyHeight
	}

	// If no seed is provided, the behavior will be deterministic for a given index.
	if seed == 0 {
		seed = int64(index)
	}
	rng := rand.New(rand.NewSource(seed))
	m, err := maze.New(width, height, DenWidth, DenHeight)
	if err != nil {
		log.Fatal(err)
	}
	m.Generate(seed, startPoint, endPoint, nil, "top", getBias(gameMode, index))

	items := newItems(m)

	solution, ok := m.Solve()
	if !ok {
		log.Fatalf("no solution for width=%d, height=%d, denWidth=%d, denHeight=%d, seed=%d", width, height, DenWidth, DenHeight, seed)
	}
	solution = solution[1 : len(solution)-1]
	items = placeDots(items, solution)

	// Scale item counts based on maze area
	baseArea := 21.0 * 15.0
	currentArea := float64(width * height)
	scaleFactor := currentArea / baseArea

	pelletCount := int(math.Max(4, float64(rng.Intn(2)+4)*scaleFactor))
	items = placePowerPellets(items, m, pelletCount)

	// In "Crazy" mode, a fuse is placed on every floor.
	if gameMode == state.ModeCrazy {
		items = placeFuse(items, m, rng)
	}

	crumblingWallCount := int(math.Max(5, float64(5)*scaleFactor))
	items = placeCrumblingWalls(items, m, rng, crumblingWallCount)

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

// getBias calculates bias that controls the straightness of paths
// The lower bias makes paths more curly
func getBias(gameMode string, floorIndex int) float64 {
	const (
		easyBias  = 0.6
		noisyBias = 0.5
		crazyBias = 0.2
		minBias   = 0.1
	)
	switch gameMode {
	case state.ModeEasy:
		return min((easyBias - float64(floorIndex)*0.02), noisyBias)
	case state.ModeNoisy:
		return min((noisyBias - float64(floorIndex)*0.05), crazyBias)
	case state.ModeCrazy:
		return min((crazyBias - float64(floorIndex)*0.02), minBias)
	default: // state.ModeNoisy
		return noisyBias
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

// BreakWall changes a crumbling wall into an empty space.
func (f *Floor) BreakWall(x, y int) {
	if x < 0 || x >= f.Maze.Width() || y < 0 || y >= f.Maze.Height() {
		return
	}
	f.Items[y][x] = Empty
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

// ShowCrumbs make crumbs look like in easy mode
func (f *Floor) ShowCrumbs(floorNum int, spriteSize string) {
	brightStyle, _ := getFloorItemStyle(floorNum, Dot)
	var sprite []string
	for _, s := range getFloorSprite(spriteSize, state.ModeEasy, Dot) {
		sprite = append(sprite, brightStyle.Render(s))
	}
	f.Sprites[Dot] = sprite
}

func setFloorSprites(floorNum int, spriteSize, gameMode string) (map[ItemType][]string, []string) {
	var sprites = map[ItemType][]string{
		Wall:          nil,
		CrumblingWall: nil,
		Dot:           nil,
		PowerPellet:   nil,
		Start:         nil,
		End:           nil,
		Fuse:          nil,
		Empty:         nil,
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
	case CrumblingWall:
		color = style.RGBColor["grey"]
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
			return []string{"▒"}
		case CrumblingWall:
			return []string{"░"}
		case Dot:
			if mode != state.ModeEasy { // Bread crumbs are only for easy game mode
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
		case CrumblingWall:
			return []string{"░░"}
		case Dot:
			if mode != state.ModeEasy {
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
			return []string{"▒▒▒▒", "▒▒▒▒"}
		case CrumblingWall:
			return []string{"░░░░", "░░░░"}
		case Dot:
			if mode != state.ModeEasy {
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
