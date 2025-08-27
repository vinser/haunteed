package dweller

import (
	"math/rand"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/vinser/haunteed/internal/floor"
	"github.com/vinser/haunteed/internal/state"
	"github.com/vinser/haunteed/internal/style"
)

// GhostState defines the current behavior mode of a ghost.
type GhostState int

const (
	Chase GhostState = iota
	Scatter
	Frightened
	Eaten
	Exiting
)

// GhostType defines the ghost's identity.
type GhostType int

const (
	Curly GhostType = iota
	Lofty
	Fluffy
	Virty
)

// Ghost represents a ghost entity.
type Ghost struct {
	position      Position
	direction     Direction
	state         GhostState
	stateSprites  map[GhostState][]string
	ghostType     GhostType
	typeSprite    []string
	home          Position
	scatterTarget Position
	rng           *rand.Rand
	exitTarget    Position // where to move during exiting
	releaseTime   time.Time
}

// NewGhost creates a ghost with specified type and home position.
func NewGhost(t GhostType, home Position, mazeWidth, mazeHeight int, rng *rand.Rand) *Ghost {
	var scatter Position
	switch t {
	case Curly:
		scatter = Position{X: mazeWidth - 1, Y: 0}
	case Fluffy:
		scatter = Position{X: 0, Y: 0}
	case Lofty:
		scatter = Position{X: mazeWidth - 1, Y: mazeHeight - 1}
	case Virty:
		scatter = Position{X: 0, Y: mazeHeight - 1}
	}

	return &Ghost{
		home:          home,
		position:      home,
		direction:     Left,
		state:         Chase,
		ghostType:     t,
		scatterTarget: scatter,
		rng:           rng,
	}
}

// GhostController manages ghost behavior state transitions over time.
type GhostController struct {
	modeIndex   int
	modeTimer   time.Time
	modePattern []ghostModePhase
}

// ghostModePhase defines a chase/scatter phase and its duration.
type ghostModePhase struct {
	state    GhostState
	duration time.Duration
}

// NewGhostController initializes chase/scatter phase logic.
func NewGhostController() *GhostController {
	return &GhostController{
		modeIndex: 0,
		modeTimer: time.Now(),
		modePattern: []ghostModePhase{
			// {state: Scatter, duration: 7 * time.Second},
			// {state: Chase, duration: 20 * time.Second},
			// {state: Scatter, duration: 7 * time.Second},
			// {state: Chase, duration: 20 * time.Second},
			{state: Scatter, duration: 5 * time.Second},
			{state: Chase, duration: 9999 * time.Hour}, // effectively infinite
		},
	}
}

// Update updates ghost states based on the time and current phase.
func (gc *GhostController) Update(ghosts []*Ghost) {
	if time.Since(gc.modeTimer) >= gc.modePattern[gc.modeIndex].duration {
		gc.modeIndex++
		if gc.modeIndex >= len(gc.modePattern) {
			gc.modeIndex = len(gc.modePattern) - 1
		}
		gc.modeTimer = time.Now()
	}

	currentState := gc.modePattern[gc.modeIndex].state

	for _, g := range ghosts {
		if g.state == Chase || g.state == Scatter {
			g.state = currentState
		}
	}
}

// targetPos returns the current target position depending on ghost type and state.
func (g *Ghost) targetPos(ht Position, htDir Direction, curlyPos Position) Position {
	switch g.state {
	case Chase:
		switch g.ghostType {
		case Curly:
			return ht
		case Fluffy:
			return ht.moveInN(htDir, 4)
		case Lofty:
			v := ht.moveInN(htDir, 2)
			return curlyPos.vectorTarget(v)
		case Virty:
			if manhattan(g.position, ht) > 8 {
				return ht
			}
			return g.scatterTarget
		}
	case Frightened:
		// Not used here, handled separately
	case Eaten:
		// Not used here, handled separately
	}
	// Default fallback
	return g.scatterTarget
}

// Place new ghosts in the ghosts den randomly.
func PlaceGhosts(floorNum int, spriteSize string, gameMode string, mazeWidth, mazeHeight, denWidth, denHeight int, rng *rand.Rand) []*Ghost {
	if denWidth%2 == 0 {
		denWidth++
	}
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	// Compute top-left position of the den inner area (room - walls)
	startCol := (mazeWidth-denWidth)/2 + 1
	startRow := (mazeHeight-denHeight)/2 + 1

	ghosts := make([]*Ghost, 4)
	for i := Curly; i <= Virty; i++ {
		pos := Position{
			X: startCol + rng.Intn(denWidth-2),
			Y: startRow + rng.Intn(denHeight-2),
		}
		ghosts[i] = NewGhost(GhostType(i), pos, mazeWidth, mazeHeight, rng)
		ghosts[i].SetExit(mazeWidth, mazeHeight, denWidth, denHeight)
		ghosts[i].SetState(Exiting)
		delay := time.Duration(i) * 3 * time.Second
		ghosts[i].SetRelease(delay)
		ghosts[i].typeSprite = setGhostTypeSprite(floorNum, spriteSize, i, gameMode)
		ghosts[i].stateSprites = setGhostStateSprites(floorNum, spriteSize, gameMode)
	}

	return ghosts
}

// State returns the type of the ghost.
func (g *Ghost) Type() GhostType {
	return g.ghostType
}

// State returns the current state of the ghost.
func (g *Ghost) State() GhostState {
	return g.state
}

// SetState updates the ghost's state.
func (g *Ghost) SetState(state GhostState) {
	g.state = state
}

// Pos returns the current position of the ghost.
func (g *Ghost) Pos() Position {
	return g.position
}

// SetPos sets the ghost's position directly.
func (g *Ghost) SetPos(pos Position) {
	g.position = pos
}

// Home returns the ghost's home position.
func (g *Ghost) Home() Position {
	return g.home
}

func (g *Ghost) SetExit(mazeWidth, mazeHeight, denWidth, denHeight int) {
	g.exitTarget = Position{X: mazeWidth / 2, Y: (mazeHeight-denHeight)/2 - 1}

}

// SetRelease sets the time after which the ghost is allowed to exit the den.
func (g *Ghost) SetRelease(delay time.Duration) {
	g.releaseTime = time.Now().Add(delay)
}

// Move moves the ghost in its current direction.
// NextPos returns the position the ghost would move to.
func (g *Ghost) NextPos() Position {
	pos := g.position
	switch g.direction {
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

// MoveGhosts moves each ghost according to its state.
func MoveGhosts(ghosts []*Ghost, f *floor.Floor, powerMode bool, htPos Position, htDir Direction) {
	var curlyPos Position
	for _, g := range ghosts {
		if g.ghostType == Curly {
			curlyPos = g.Pos()
			break
		}
	}

	for _, g := range ghosts {
		switch g.State() {
		case Frightened:
			g.MoveRandom(f, ghosts)
		case Eaten:
			if g.Pos() == g.Home() {
				if !powerMode {
					g.SetState(Chase)
				}
			} else {
				g.MoveToHome(f, ghosts)
			}
		case Chase,
			Scatter:
			target := g.targetPos(htPos, htDir, curlyPos)
			g.moveToTarget(f, target, ghosts)
		case Exiting:
			if powerMode {
				// Do nothing — ghost waits in den
				continue
			}
			if time.Now().Before(g.releaseTime) {
				continue // Delay exit
			}
			if g.Pos().X == g.exitTarget.X && abs(g.Pos().Y-g.exitTarget.Y) <= 1 { // Fix exitTarget inaccuracy
				g.SetState(Chase)
			} else {
				g.moveToTarget(f, g.exitTarget, ghosts)
			}
		}
	}
}

// Move moves the ghost in its current direction.
func (g *Ghost) Move() {
	g.position = g.NextPos()
}

// MoveToHome moves the ghost one step closer to its home position.
func (g *Ghost) MoveToHome(f *floor.Floor, allGhosts []*Ghost) {
	g.moveToTarget(f, g.home, allGhosts)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// SetDirection sets the ghost's movement direction.
func (g *Ghost) SetDirection(dir Direction) {
	g.direction = dir
}

func (g *Ghost) MoveRandom(f *floor.Floor, allGhosts []*Ghost) {
	// Directions excluding opposite
	possible := g.validDirectionsExcludingOpposite(f, allGhosts)

	// If there are no valid directions excluding opposite try all
	if len(possible) == 0 {
		possible = g.validAllDirections(f, allGhosts)
	}

	if len(possible) == 0 {
		return // No valid directions
	}

	g.direction = possible[g.rng.Intn(len(possible))]
	g.Move()
}

func (g *Ghost) validDirectionsExcludingOpposite(f *floor.Floor, allGhosts []*Ghost) []Direction {
	var dirs []Direction
	opp := oppositeDirection(g.direction)

	for _, d := range []Direction{Up, Down, Left, Right} {
		if d == opp {
			continue
		}
		if g.canMoveTo(g.position, d, f, allGhosts) {
			dirs = append(dirs, d)
		}
	}
	return dirs
}

func (g *Ghost) validAllDirections(f *floor.Floor, allGhosts []*Ghost) []Direction {
	var dirs []Direction
	for _, d := range []Direction{Up, Down, Left, Right} {
		if g.canMoveTo(g.position, d, f, allGhosts) {
			dirs = append(dirs, d)
		}
	}
	return dirs
}

// canMoveTo checks if a ghost can move to a new position.
// It checks for walls and other ghosts.
func (g *Ghost) canMoveTo(p Position, d Direction, f *floor.Floor, allGhosts []*Ghost) bool {
	newPos := p.moveIn(d)

	// Check for walls
	tile, err := f.ItemAt(newPos.X, newPos.Y)
	if err != nil || tile == floor.Wall || tile == floor.CrumblingWall {
		return false
	}

	// Check for other ghosts
	for _, otherGhost := range allGhosts {
		if g == otherGhost {
			continue // Don't check against self
		}
		if otherGhost.Pos() == newPos {
			return false // Another ghost is there
		}
	}

	return true
}

func (p Position) moveIn(d Direction) Position {
	switch d {
	case Up:
		p.Y--
	case Down:
		p.Y++
	case Left:
		p.X--
	case Right:
		p.X++
	}
	return p
}

// moveInN returns the position moved N steps in a given direction.
func (p Position) moveInN(d Direction, n int) Position {
	for i := 0; i < n; i++ {
		p = p.moveIn(d)
	}
	return p
}

// vectorTarget computes the point two times the vector from self to target.
func (from Position) vectorTarget(to Position) Position {
	dx := to.X - from.X
	dy := to.Y - from.Y
	return Position{X: from.X + 2*dx, Y: from.Y + 2*dy}
}

// manhattan returns the Manhattan distance between two points.
func manhattan(a, b Position) int {
	return abs(a.X-b.X) + abs(a.Y-b.Y)
}

func oppositeDirection(d Direction) Direction {
	switch d {
	case Up:
		return Down
	case Down:
		return Up
	case Left:
		return Right
	case Right:
		return Left
	default:
		return d
	}
}

// findBestDirections finds the optimal direction(s) from a list of valid moves to get closer to a target.
func (g *Ghost) findBestDirections(target Position, validDirections []Direction) []Direction {
	shortest := 1 << 30
	var candidates []Direction

	for _, d := range validDirections {
		next := g.position.moveIn(d)
		dist := manhattan(next, target)
		if dist < shortest {
			shortest = dist
			candidates = []Direction{d}
		} else if dist == shortest {
			candidates = append(candidates, d)
		}
	}
	return candidates
}

// moveToTarget moves the ghost one step toward the target position.
func (g *Ghost) moveToTarget(l *floor.Floor, target Position, allGhosts []*Ghost) {
	// Find best directions, excluding reversing.
	candidates := g.findBestDirections(target, g.validDirectionsExcludingOpposite(l, allGhosts))

	// Fallback if no valid direction excluding reverse
	if len(candidates) == 0 {
		candidates = g.findBestDirections(target, g.validAllDirections(l, allGhosts))
	}

	if len(candidates) > 0 {
		g.direction = candidates[g.rng.Intn(len(candidates))]
		g.Move()
	}
}

func (g *Ghost) Render(size string) []string {
	// States like Frightened or Eaten override the type-specific sprite.
	sprite, ok := g.stateSprites[g.State()]
	if !ok {
		// For other states, use the type-specific sprite.
		sprite = g.typeSprite
	}
	return sprite
}

func setGhostTypeSprite(floorNum int, spriteSize string, ghostType GhostType, gameMode string) []string {
	brightStyle, _ := getGostTypeStyle(floorNum, ghostType)
	var sprite []string
	for _, s := range getGhostTypeSprite(spriteSize, ghostType) {
		sprite = append(sprite, brightStyle.Render(s))
	}
	return sprite
}

func getGostTypeStyle(floorNum int, ghostType GhostType) (brightStyle, dimStyle lipgloss.Style) {
	var color style.RGB
	switch ghostType {
	case Curly:
		color = style.RGBColor["red"]
	case Lofty:
		color = style.RGBColor["magenta"]
	case Fluffy:
		color = style.RGBColor["cyan"]
	case Virty:
		color = style.RGBColor["green"]
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

func getGhostTypeSprite(size string, ghostType GhostType) []string {
	switch size {
	case state.SpriteSmall:
		switch ghostType {
		case Curly:
			return []string{"␍"}
		case Lofty:
			return []string{"␊"}
		case Fluffy:
			return []string{"␌"}
		case Virty:
			return []string{"␋"}
		default:
			return []string{" "}
		}
	case state.SpriteMedium:
		switch ghostType {
		case Curly:
			return []string{"Cr"}
		case Lofty:
			return []string{"Lf"}
		case Fluffy:
			return []string{"Ff"}
		case Virty:
			return []string{"Vt"}
		default:
			return []string{" "}
		}
	case state.SpriteLarge:
		switch ghostType {
		case Curly:
			return []string{" C  ", "  R "}
		case Lofty:
			return []string{" L  ", "  F "}
		case Fluffy:
			return []string{" F  ", "  F "}
		case Virty:
			return []string{" V  ", "  T "}
		default:
			return []string{"    ", "    "}
		}
	}
	return []string{" "}
}

func setGhostStateSprites(floorNum int, spriteSize, gameMode string) map[GhostState][]string {
	var sprites = map[GhostState][]string{
		Frightened: nil,
		Eaten:      nil,
	}

	for s := range sprites {
		brightStyle, _ := getGostStateStyle(floorNum, s)
		var sprite []string
		for _, s := range getGhostStateSprite(spriteSize, s) {
			sprite = append(sprite, brightStyle.Render(s))
		}
		sprites[s] = sprite
	}
	return sprites
}

func getGostStateStyle(floorNum int, ghostState GhostState) (brightStyle, dimStyle lipgloss.Style) {
	var color style.RGB
	switch ghostState {
	case Frightened:
		color = style.RGBColor["blue"]
	case Eaten:
		color = style.RGBColor["grey"]
	default:
		color = style.RGBColor["grey"]
	}

	brightR, dimR := style.FloorColorShift(color.R, floorNum)
	brightG, dimG := style.FloorColorShift(color.G, floorNum)
	brightB, dimB := style.FloorColorShift(color.B, floorNum)

	dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(style.GenerateHexColor(dimR, dimG, dimB)))
	brightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(style.GenerateHexColor(brightR, brightG, brightB)))

	return brightStyle, dimStyle
}

func getGhostStateSprite(size string, ghostState GhostState) []string {
	switch size {
	case state.SpriteSmall:
		switch ghostState {
		case Frightened:
			return []string{"␛"}
		case Eaten:
			return []string{"␡"}
		default:
			return []string{" "}
		}
	case state.SpriteMedium:
		switch ghostState {
		case Frightened:
			return []string{"Es"}
		case Eaten:
			return []string{"Dl"}
		default:
			return []string{" "}
		}
	case state.SpriteLarge:
		switch ghostState {
		case Frightened:
			return []string{" E  ", " SC "}
		case Eaten:
			return []string{" D  ", " EL "}
		default:
			return []string{"    ", "    "}
		}
	}
	return []string{" "}
}
