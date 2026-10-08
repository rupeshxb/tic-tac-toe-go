package game

import (
	"errors"
	"testing"
)

// play applies moves alternately starting with X, failing the test on error.
func play(t *testing.T, g *Game, positions ...int) {
	t.Helper()
	for _, p := range positions {
		if err := g.ApplyMove(p, g.CurrentTurn); err != nil {
			t.Fatalf("ApplyMove(%d) unexpected error: %v", p, err)
		}
	}
}

// TestNew checks the initial state: empty board, X to move, not over.
func TestNew(t *testing.T) {
	g := New()
	if g.CurrentTurn != "X" {
		t.Errorf("CurrentTurn = %q, want X", g.CurrentTurn)
	}
	for i, c := range g.Board {
		if c != "" {
			t.Errorf("Board[%d] = %q, want empty", i, c)
		}
	}
	if g.CheckWinner() != "" || g.IsDraw() || g.IsOver() {
		t.Error("new game should have no winner, draw, or end state")
	}
}

// TestApplyMoveValid checks a legal move places the symbol and swaps turns.
func TestApplyMoveValid(t *testing.T) {
	g := New()
	if err := g.ApplyMove(4, "X"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.Board[4] != "X" {
		t.Errorf("Board[4] = %q, want X", g.Board[4])
	}
	if g.CurrentTurn != "O" {
		t.Errorf("CurrentTurn = %q, want O", g.CurrentTurn)
	}
}

// TestApplyMoveOccupied checks moving onto a taken cell is rejected.
func TestApplyMoveOccupied(t *testing.T) {
	g := New()
	play(t, g, 0)
	err := g.ApplyMove(0, "O")
	if !errors.Is(err, ErrOccupied) {
		t.Errorf("err = %v, want ErrOccupied", err)
	}
	if g.CurrentTurn != "O" {
		t.Error("turn should not change after a rejected move")
	}
}

// TestApplyMoveOutOfBounds checks positions outside 0-8 are rejected.
func TestApplyMoveOutOfBounds(t *testing.T) {
	for _, p := range []int{-1, 9, 100} {
		g := New()
		if err := g.ApplyMove(p, "X"); !errors.Is(err, ErrOutOfBounds) {
			t.Errorf("position %d: err = %v, want ErrOutOfBounds", p, err)
		}
	}
}

// TestApplyMoveWrongTurn checks a player cannot move out of turn.
func TestApplyMoveWrongTurn(t *testing.T) {
	g := New()
	if err := g.ApplyMove(0, "O"); !errors.Is(err, ErrWrongTurn) {
		t.Errorf("err = %v, want ErrWrongTurn", err)
	}
	if g.Board[0] != "" {
		t.Error("board should be unchanged after a rejected move")
	}
}

// TestApplyMoveInvalidSymbol checks symbols other than X and O are rejected.
func TestApplyMoveInvalidSymbol(t *testing.T) {
	g := New()
	if err := g.ApplyMove(0, "Z"); !errors.Is(err, ErrInvalidSymbol) {
		t.Errorf("err = %v, want ErrInvalidSymbol", err)
	}
}

// TestApplyMoveAfterGameOver checks no moves are accepted once someone has won.
func TestApplyMoveAfterGameOver(t *testing.T) {
	g := New()
	play(t, g, 0, 3, 1, 4, 2) // X wins top row
	if err := g.ApplyMove(8, "O"); !errors.Is(err, ErrGameOver) {
		t.Errorf("err = %v, want ErrGameOver", err)
	}
}

// TestRowWin checks a winning row is detected.
func TestRowWin(t *testing.T) {
	g := New()
	play(t, g, 0, 3, 1, 4, 2) // X: 0,1,2  O: 3,4
	if w := g.CheckWinner(); w != "X" {
		t.Errorf("winner = %q, want X", w)
	}
	if !g.IsOver() || g.IsDraw() {
		t.Error("game should be over and not a draw")
	}
}

// TestColumnWin checks a winning column is detected.
func TestColumnWin(t *testing.T) {
	g := New()
	play(t, g, 1, 0, 2, 3, 8, 6) // O: 0,3,6
	if w := g.CheckWinner(); w != "O" {
		t.Errorf("winner = %q, want O", w)
	}
	if !g.IsOver() {
		t.Error("game should be over")
	}
}

// TestDiagonalWin checks both diagonals are detected.
func TestDiagonalWin(t *testing.T) {
	g := New()
	play(t, g, 0, 1, 4, 2, 8) // X: 0,4,8
	if w := g.CheckWinner(); w != "X" {
		t.Errorf("main diagonal winner = %q, want X", w)
	}

	g = New()
	play(t, g, 2, 0, 4, 1, 6) // X: 2,4,6
	if w := g.CheckWinner(); w != "X" {
		t.Errorf("anti diagonal winner = %q, want X", w)
	}
}

// TestDraw checks a full board with no winner is a draw.
func TestDraw(t *testing.T) {
	g := New()
	// X O X
	// X O O
	// O X X
	play(t, g, 0, 1, 2, 4, 3, 5, 7, 6, 8)
	if g.CheckWinner() != "" {
		t.Fatalf("unexpected winner %q", g.CheckWinner())
	}
	if !g.IsDraw() {
		t.Error("IsDraw = false, want true")
	}
	if !g.IsOver() {
		t.Error("IsOver = false, want true")
	}
}
