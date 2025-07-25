package floor

import (
	"math"
	"math/rand"
	"time"

	"github.com/vinser/maze"
)

// newItems creates a new items grid and fills it with walls from the maze.
func newItems(m *maze.Maze) [][]ItemType {
	items := make([][]ItemType, Height)
	for y := 0; y < Height; y++ {
		items[y] = make([]ItemType, Width)
		for x := 0; x < Width; x++ {
			cell, ok := m.Cell(x, y)
			if !ok {
				continue
			}
			switch cell {
			case maze.Wall:
				items[y][x] = Wall
			case maze.Path:
				items[y][x] = Empty
			case maze.Start:
				items[y][x] = Start
			case maze.End:
				items[y][x] = End
			default:
				items[y][x] = Wall
			}
		}
	}
	return items
}

// placeDots fills items grid with solution Dots
func placeDots(items [][]ItemType, solution []maze.Point) [][]ItemType {
	// Create a map of solution points for easy lookup when rendering.
	solutionPoints := make(map[maze.Point]bool)
	for _, p := range solution {
		solutionPoints[p] = true
	}
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			if items[y][x] == Empty && solutionPoints[maze.Point{X: x, Y: y}] {
				items[y][x] = Dot
			}
		}
	}
	return items
}

// placePowerPellets places requested number of power pellets in the items grid
// at maximum distance from the maze center and between them.
func placePowerPellets(items [][]ItemType, m *maze.Maze, requested int) [][]ItemType {
	var candidates []maze.Point
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			// Candidates for power pellets are empty points outside the den.
			if items[y][x] == Empty && !m.IsInsideDen(maze.Point{X: x, Y: y}) {
				candidates = append(candidates, maze.Point{X: x, Y: y})
			}
		}
	}

	if requested <= 0 || len(candidates) == 0 {
		return items
	}

	if requested > len(candidates) {
		requested = len(candidates)
	}

	mazeCenter := maze.Point{X: Width / 2, Y: Height / 2}
	var pelletLocations []maze.Point

	// 1. Place the first pellet: farthest from the maze center.
	bestCandidateIndex := -1
	maxDistFromCenter := -1
	for i, p := range candidates {
		dist := manhattan(p.X, p.Y, mazeCenter.X, mazeCenter.Y)
		if dist > maxDistFromCenter {
			maxDistFromCenter = dist
			bestCandidateIndex = i
		}
	}

	if bestCandidateIndex != -1 {
		pelletLocations = append(pelletLocations, candidates[bestCandidateIndex])
		// Remove the chosen candidate by swapping with the last element and slicing.
		candidates[bestCandidateIndex] = candidates[len(candidates)-1]
		candidates = candidates[:len(candidates)-1]
	}

	// 2. Place subsequent pellets: use a greedy approach to maximize distance from other pellets.
	for len(pelletLocations) < requested && len(candidates) > 0 {
		bestCandidateIndex = -1
		maxMinDist := -1

		for i, cand := range candidates {
			minDistToPellets := math.MaxInt32
			for _, pellet := range pelletLocations {
				dist := manhattan(cand.X, cand.Y, pellet.X, pellet.Y)
				if dist < minDistToPellets {
					minDistToPellets = dist
				}
			}

			if minDistToPellets > maxMinDist {
				maxMinDist = minDistToPellets
				bestCandidateIndex = i
			}
		}

		if bestCandidateIndex != -1 {
			pelletLocations = append(pelletLocations, candidates[bestCandidateIndex])
			candidates[bestCandidateIndex] = candidates[len(candidates)-1]
			candidates = candidates[:len(candidates)-1]
		} else {
			// Should not happen if there are candidates left, but as a safeguard.
			break
		}
	}

	// 3. Update the items grid with the chosen power pellet locations.
	for _, p := range pelletLocations {
		items[p.Y][p.X] = PowerPellet
	}

	return items
}

// placeFuse places a single fuse at a random empty location.
func placeFuse(items [][]ItemType, m *maze.Maze, rng *rand.Rand) [][]ItemType {
	var candidates []maze.Point
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			// Candidates for fuse are empty points outside the den.
			if items[y][x] == Empty && !m.IsInsideDen(maze.Point{X: x, Y: y}) {
				candidates = append(candidates, maze.Point{X: x, Y: y})
			}
		}
	}

	if len(candidates) > 0 {
		fuseLocation := candidates[rng.Intn(len(candidates))]
		items[fuseLocation.Y][fuseLocation.X] = Fuse
	}

	return items
}

// Manhattan distance between two points.
func manhattan(x1, y1, x2, y2 int) int {
	return abs(x1-x2) + abs(y1-y2)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

const (
	// Default ghost tick interval for the first floor.
	defaultGhostTickInterval = 500 * time.Millisecond
	// Minimum ghost tick interval.
	minGhostTickInterval = 400 * time.Millisecond
	// Ghost tick interval decrease step.
	stepGhostTickInterval = 10 * time.Millisecond
)

// ghostInterval returns the ghost movement interval based on floor.
func ghostInterval(floor int) time.Duration {
	calculated := defaultGhostTickInterval - time.Duration(abs(floor))*stepGhostTickInterval
	if calculated < minGhostTickInterval {
		return minGhostTickInterval
	}
	return calculated
}
