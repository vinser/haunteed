package dweller

// Position represents coordinates on the map.
type Position struct {
	X, Y int
}

// Direction represents movement direction.
type Direction int

const (
	No Direction = iota
	Up
	Down
	Left
	Right
)
