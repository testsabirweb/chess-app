package chess_test

import (
	"reflect"
	"testing"

	"github.com/testsabirweb/chess-app/internal/chess"
)

func TestWhitePawnAttacksDiagonalsNotAhead(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(2, 1), white(chess.Pawn))
	got := b.Attacks(chess.Sq(2, 1))
	want := []chess.Square{chess.Sq(1, 2), chess.Sq(3, 2)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v (empty diagonals are still attacked, the square ahead is not)", got, want)
	}
}

func TestBlackPawnAttacksDownTheBoard(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(2, 3), black(chess.Pawn))
	got := b.Attacks(chess.Sq(2, 3))
	want := []chess.Square{chess.Sq(1, 2), chess.Sq(3, 2)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestPawnOnTheEdgeAttacksOneSquare(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(0, 1), white(chess.Pawn))
	if got := b.Attacks(chess.Sq(0, 1)); !reflect.DeepEqual(got, []chess.Square{chess.Sq(1, 2)}) {
		t.Fatalf("got %v", got)
	}
	b.Set(chess.Sq(4, 4), white(chess.Pawn))
	if got := b.Attacks(chess.Sq(4, 4)); len(got) != 0 {
		t.Fatalf("pawn on the last rank attacks %v", got)
	}
}

// A slider's line includes the first piece it meets, even a friendly one, and
// stops there.
func TestRookAttacksStopAtAndIncludeBlockers(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(0, 0), black(chess.Rook))
	b.Set(chess.Sq(0, 2), black(chess.Pawn)) // friendly
	b.Set(chess.Sq(3, 0), white(chess.Pawn)) // enemy
	got := b.Attacks(chess.Sq(0, 0))
	// Sorted by rank, then file.
	want := []chess.Square{chess.Sq(1, 0), chess.Sq(2, 0), chess.Sq(3, 0), chess.Sq(0, 1), chess.Sq(0, 2)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestKnightAndKingAttacks(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(0, 0), black(chess.Knight))
	if got := b.Attacks(chess.Sq(0, 0)); !reflect.DeepEqual(got, []chess.Square{chess.Sq(2, 1), chess.Sq(1, 2)}) {
		t.Fatalf("knight: %v", got)
	}
	b = chess.NewBoard(5, 5)
	b.Set(chess.Sq(4, 4), black(chess.King))
	if got := b.Attacks(chess.Sq(4, 4)); !reflect.DeepEqual(got, []chess.Square{chess.Sq(3, 3), chess.Sq(4, 3), chess.Sq(3, 4)}) {
		t.Fatalf("king: %v", got)
	}
}

func TestAttacksOfAnEmptySquareOrOffBoard(t *testing.T) {
	b := chess.NewBoard(5, 5)
	if got := b.Attacks(chess.Sq(2, 2)); len(got) != 0 {
		t.Fatalf("empty square attacks %v", got)
	}
	if got := b.Attacks(chess.Sq(9, 9)); got != nil {
		t.Fatalf("off-board attacks %v", got)
	}
}

// For every piece but the pawn, a move onto an empty square or a capture of an
// enemy is always an attacked square.
func TestMovesAreAlwaysAttacked(t *testing.T) {
	for _, pt := range []chess.PieceType{chess.Knight, chess.Bishop, chess.Rook, chess.Queen, chess.King} {
		b := chess.NewBoard(5, 5)
		b.Set(chess.Sq(2, 2), white(pt))
		b.Set(chess.Sq(4, 4), black(chess.Pawn))
		attacked := map[chess.Square]bool{}
		for _, s := range b.Attacks(chess.Sq(2, 2)) {
			attacked[s] = true
		}
		for _, s := range b.MoveTargets(chess.Sq(2, 2)) {
			if !attacked[s] {
				t.Fatalf("%v can move to %v but does not attack it", pt, s)
			}
		}
	}
}
