package chess_test

import (
	"reflect"
	"testing"

	"github.com/testsabirweb/chess-app/internal/chess"
)

func white(t chess.PieceType) chess.Piece {
	return chess.Piece{Type: t, Color: chess.White}
}

func black(t chess.PieceType) chess.Piece {
	return chess.Piece{Type: t, Color: chess.Black}
}

func targets(b *chess.Board, from chess.Square) []chess.Square {
	return b.MoveTargets(from)
}

func TestRookFromCenter5x5(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(2, 2), white(chess.Rook))
	got := targets(b, chess.Sq(2, 2))
	want := []chess.Square{
		chess.Sq(2, 0), chess.Sq(2, 1),
		chess.Sq(0, 2), chess.Sq(1, 2), chess.Sq(3, 2), chess.Sq(4, 2),
		chess.Sq(2, 3), chess.Sq(2, 4),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rook c3 targets = %v, want %v", got, want)
	}
}

func TestBishopFromCenter5x5(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(2, 2), white(chess.Bishop))
	got := targets(b, chess.Sq(2, 2))
	want := []chess.Square{
		chess.Sq(0, 0), chess.Sq(4, 0),
		chess.Sq(1, 1), chess.Sq(3, 1),
		chess.Sq(1, 3), chess.Sq(3, 3),
		chess.Sq(0, 4), chess.Sq(4, 4),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("bishop c3 targets = %v, want %v", got, want)
	}
}

func TestQueenFromCenter5x5(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(2, 2), white(chess.Queen))
	got := targets(b, chess.Sq(2, 2))
	if len(got) != 16 {
		t.Fatalf("queen c3 targets = %d, want 16", len(got))
	}
}

func TestKnightFromCorner5x5(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(0, 0), white(chess.Knight))
	got := targets(b, chess.Sq(0, 0))
	want := []chess.Square{chess.Sq(2, 1), chess.Sq(1, 2)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("knight a1 targets = %v, want %v", got, want)
	}
}

func TestKingFromCenter5x5(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(2, 2), white(chess.King))
	got := targets(b, chess.Sq(2, 2))
	if len(got) != 8 {
		t.Fatalf("king c3 targets = %d, want 8", len(got))
	}
}

func TestKingFromCorner5x5(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(0, 0), white(chess.King))
	got := targets(b, chess.Sq(0, 0))
	if len(got) != 3 {
		t.Fatalf("king a1 targets = %d, want 3", len(got))
	}
}

func TestWhitePawnFromHome5x5(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(2, 1), white(chess.Pawn))
	got := targets(b, chess.Sq(2, 1))
	want := []chess.Square{chess.Sq(2, 2), chess.Sq(2, 3)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pawn c2 targets = %v, want %v", got, want)
	}
}

func TestBlockedSlider(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(2, 2), white(chess.Rook))
	b.Set(chess.Sq(2, 4), white(chess.Pawn))
	got := targets(b, chess.Sq(2, 2))
	for _, s := range got {
		if s.Rank > 3 {
			t.Fatalf("rook should not pass friendly pawn, got %v", got)
		}
	}
}

func TestPawnCaptureOnlyOnOccupied(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(2, 2), white(chess.Pawn))
	b.Set(chess.Sq(3, 3), black(chess.Pawn))
	got := targets(b, chess.Sq(2, 2))
	found := false
	for _, s := range got {
		if s == chess.Sq(3, 3) {
			found = true
		}
		if s == chess.Sq(1, 3) {
			t.Fatal("pawn should not capture empty diagonal square")
		}
	}
	if !found {
		t.Fatal("pawn should capture enemy on diagonal")
	}
}

func TestMovesDeterministic(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(2, 2), white(chess.Queen))
	a := b.Moves(nil, chess.Sq(2, 2))
	b2 := b.Clone()
	c := b2.Moves(nil, chess.Sq(2, 2))
	if !reflect.DeepEqual(a, c) {
		t.Fatal("Moves should be deterministic across calls")
	}
}

func TestOccupiedSorted(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(3, 3), white(chess.Rook))
	b.Set(chess.Sq(1, 1), white(chess.Pawn))
	got := b.Occupied()
	want := []chess.Square{chess.Sq(1, 1), chess.Sq(3, 3)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Occupied = %v, want %v", got, want)
	}
}

func TestSetOffBoardPanics(t *testing.T) {
	b := chess.NewBoard(5, 5)
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on off-board set")
		}
	}()
	b.Set(chess.Sq(5, 5), white(chess.King))
}

func TestCloneDeep(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(2, 2), white(chess.King))
	c := b.Clone()
	c.Set(chess.Sq(2, 2), chess.Piece{})
	if b.At(chess.Sq(2, 2)).IsEmpty() {
		t.Fatal("clone should be independent")
	}
}

func TestAllTargetsValid(t *testing.T) {
	pieces := []chess.PieceType{chess.Pawn, chess.Knight, chess.Bishop, chess.Rook, chess.Queen, chess.King}
	for _, w := range []int{5, 8} {
		b := chess.NewBoard(w, w)
		for r := 0; r < w; r++ {
			for f := 0; f < w; f++ {
				from := chess.Sq(f, r)
				for _, pt := range pieces {
					b.Set(from, white(pt))
					ts := targets(b, from)
					seen := map[chess.Square]bool{}
					for _, to := range ts {
						if to == from {
							t.Fatalf("%v from %v has self target", pt, from)
						}
						if !b.Contains(to) {
							t.Fatalf("%v target %v off board", pt, to)
						}
						if seen[to] {
							t.Fatalf("%v duplicate target %v", pt, to)
						}
						seen[to] = true
					}
					b.Set(from, chess.Piece{})
				}
			}
		}
	}
}

func TestKnightKingRotationSymmetry(t *testing.T) {
	for _, pt := range []chess.PieceType{chess.Knight, chess.King} {
		b := chess.NewBoard(8, 8)
		from := chess.Sq(3, 3)
		b.Set(from, white(pt))
		n := len(targets(b, from))

		b2 := chess.NewBoard(8, 8)
		rot := chess.Sq(7-3, 7-3)
		b2.Set(rot, white(pt))
		n2 := len(targets(b2, rot))
		if n != n2 {
			t.Fatalf("%v symmetry: center=%d rotated=%d", pt, n, n2)
		}
	}
}
