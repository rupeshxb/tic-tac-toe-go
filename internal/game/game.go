// Package game contains the pure tic-tac-toe game logic.
package game

import (
	"errors"
	"fmt"
)

// Board holds the nine cells in row-major order (positions 0-8).
// An empty cell is the empty string; otherwise it is "X" or "O".
type Board [9]string

// Game holds the board and whose turn it is.
type Game struct {
	Board       Board
	CurrentTurn string
}

// Sentinel errors returned by ApplyMove.
var (
	ErrOutOfBounds   = errors.New("position out of bounds")
	ErrWrongTurn     = errors.New("not this player's turn")
	ErrOccupied      = errors.New("cell already occupied")
	ErrGameOver      = errors.New("game is already over")
	ErrInvalidSymbol = errors.New("invalid symbol")
)

// winLines lists every row, column and diagonal that wins the game.
var winLines = [8][3]int{
	{0, 1, 2}, {3, 4, 5}, {6, 7, 8}, // rows
	{0, 3, 6}, {1, 4, 7}, {2, 5, 8}, // columns
	{0, 4, 8}, {2, 4, 6}, // diagonals
}

// New returns an empty game in which X moves first.
func New() *Game {
	return &Game{CurrentTurn: "X"}
}

// ApplyMove places symbol at position for the current player, validating
// the symbol, bounds, game state, turn and cell occupancy, then switches turns.
func (g *Game) ApplyMove(position int, symbol string) error {
	if symbol != "X" && symbol != "O" {
		return fmt.Errorf("%w: %q", ErrInvalidSymbol, symbol)
	}
	if position < 0 || position > 8 {
		return fmt.Errorf("%w: %d", ErrOutOfBounds, position)
	}
	if g.IsOver() {
		return ErrGameOver
	}
	if symbol != g.CurrentTurn {
		return fmt.Errorf("%w: it is %s's turn", ErrWrongTurn, g.CurrentTurn)
	}
	if g.Board[position] != "" {
		return fmt.Errorf("%w: %d", ErrOccupied, position)
	}

	g.Board[position] = symbol
	if symbol == "X" {
		g.CurrentTurn = "O"
	} else {
		g.CurrentTurn = "X"
	}
	return nil
}

// CheckWinner returns "X" or "O" if that player has three in a line,
// or the empty string if there is no winner.
func (g *Game) CheckWinner() string {
	for _, l := range winLines {
		a := g.Board[l[0]]
		if a != "" && a == g.Board[l[1]] && a == g.Board[l[2]] {
			return a
		}
	}
	return ""
}

// IsDraw reports whether the board is full and nobody has won.
func (g *Game) IsDraw() bool {
	if g.CheckWinner() != "" {
		return false
	}
	for _, c := range g.Board {
		if c == "" {
			return false
		}
	}
	return true
}

// IsOver reports whether the game has ended by a win or a draw.
func (g *Game) IsOver() bool {
	return g.CheckWinner() != "" || g.IsDraw()
}
