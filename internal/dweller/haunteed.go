package dweller

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vinser/haunteed/internal/floor"
	"github.com/vinser/haunteed/internal/state"
	"github.com/vinser/haunteed/internal/style"
)

// Default lives count for Haunteed.
const defaultLives = 3

// Haunteed represents the player character.
type Haunteed struct {
	home         Position
	position     Position
	direction    Direction
	lives        int
	brightSprite []string
	dimSprite    []string
}

// NewHaunteed returns a new Haunteed instance with default values.
func NewHaunteed(home Position) *Haunteed {

	return &Haunteed{
		home:      home,
		position:  home, // starting position
		direction: Right,
		lives:     defaultLives,
	}
}

// PlaceHaunteed places new Haunteed in the maze
func PlaceHaunteed(size string, p Position) *Haunteed {
	haunteed := NewHaunteed(p)
	haunteed.SetHaunteedSprites(size)
	return haunteed
}

// Home returns Haunteed's home position.
func (p *Haunteed) Home() Position {
	return p.home
}

// SetHome sets Haunteed's home (spawn) position.
func (p *Haunteed) SetHome(pos Position) {
	p.home = pos
}

// Pos returns Haunteed's current position.
func (p *Haunteed) Pos() Position {
	return p.position
}

// Dir returns Haunteed's current direction.
func (p *Haunteed) Dir() Direction {
	return p.direction
}

// SetPos sets Haunteed's position explicitly.
func (p *Haunteed) SetPos(pos Position) {
	p.position = pos
}

// NextPos returns the position Haunteed would move to based on direction.
func (p *Haunteed) NextPos() Position {
	pos := p.position
	switch p.direction {
	case Up:
		pos.Y--
	case Down:
		pos.Y++
	case Left:
		pos.X--
	case Right:
		pos.X++
	}
	return pos
}

// Move attempts to move Haunteed if the next tile is not a wall.
func (p *Haunteed) Move(l *floor.Floor) {
	next := p.NextPos()

	tile, err := l.ItemAt(next.X, next.Y)
	if err == nil && tile != floor.Wall {
		p.position = next
	}
}

// HandleInput updates Haunteed's direction based on user input.
func (p *Haunteed) HandleInput(msg tea.KeyMsg) {
	switch msg.String() {
	case "up", "w":
		p.direction = Up
	case "down", "s":
		p.direction = Down
	case "left", "a":
		p.direction = Left
	case "right", "d":
		p.direction = Right
	default:
		p.direction = No
	}
}

// Lives returns Haunteed's remaining lives.
func (p *Haunteed) Lives() int {
	return p.lives
}

// LoseLife reduces Haunteed's lives by 1.
func (p *Haunteed) LoseLife() {
	if p.lives > 0 {
		p.lives--
	}
}

// IsDead returns true if Haunteed has no lives left.
func (p *Haunteed) IsDead() bool {
	return p.lives <= 0
}

// ResetLives resets Haunteed's lives to the default value.
func (p *Haunteed) ResetLives() {
	p.lives = defaultLives
}

// AddLife adds a life to Haunteed.
func (p *Haunteed) AddLife() {
	p.lives++
}

func (h *Haunteed) Render(size string) []string {
	isBright := (time.Now().UnixNano()/int64(time.Millisecond)/500)%2 == 0
	if isBright {
		return h.brightSprite
	}
	return h.dimSprite
}

func (h *Haunteed) SetHaunteedSprites(spriteSize string) {
	brightStyle, dimStyle := getHaunteedStyle()

	for _, s := range getHaunteedSprite(spriteSize) {
		h.brightSprite = append(h.brightSprite, brightStyle.Render(s))
	}
	for _, s := range getHaunteedSprite(spriteSize) {
		h.dimSprite = append(h.dimSprite, dimStyle.Render(s))
	}
}

// haunteedMaxBrightnessFloor is a value used to ensure the Haunteed character
// is always rendered at maximum brightness, independent of the actual floor number.
const haunteedMaxBrightnessFloor = 1000

func getHaunteedStyle() (brightStyle, dimStyle lipgloss.Style) {
	color := style.RGBColor["yellow"]
	brightR, dimR := style.FloorColorShift(color.R, haunteedMaxBrightnessFloor)
	brightG, dimG := style.FloorColorShift(color.G, haunteedMaxBrightnessFloor)
	brightB, dimB := style.FloorColorShift(color.B, haunteedMaxBrightnessFloor)

	dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(style.GenerateHexColor(dimR, dimG, dimB)))
	brightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(style.GenerateHexColor(brightR, brightG, brightB)))

	return brightStyle, dimStyle
}

func getHaunteedSprite(size string) []string {
	switch size {
	case state.SpriteSmall:
		return []string{"␉"}
	case state.SpriteMedium:
		return []string{"Ht"}
	case state.SpriteLarge:
		return []string{" H  ", "  T "}
	}

	return []string{" "}
}
