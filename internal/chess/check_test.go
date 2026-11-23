package chess_test

import (
	"testing"

	"github.com/testsabirweb/chess-app/internal/chess"
)

// mateOne is K+Q against a king in the corner, already mate if it were black's
// turn to face it: black k a0, white Q b1 supported by K c2.
func mateOne() *chess.Board {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(0, 0), black(chess.King))
	b.Set(chess.Sq(1, 1), white(chess.Queen))
	b.Set(chess.Sq(2, 2), white(chess.King))
	return b
}

func TestKingSquare(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(3, 2), black(chess.King))
	if s, ok := b.KingSquare(chess.Black); !ok || s != chess.Sq(3, 2) {
		t.Fatalf("KingSquare = %v, %v", s, ok)
	}
	if _, ok := b.KingSquare(chess.White); ok {
		t.Fatal("found a white king on a board without one")
	}
}

func TestAttackedByEachPiece(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(0, 1), white(chess.Pawn))
	attacked := []chess.Square{chess.Sq(1, 2)}
	free := []chess.Square{chess.Sq(0, 2), chess.Sq(0, 0), chess.Sq(1, 1)}
	for _, s := range attacked {
		if !b.Attacked(s, chess.White) {
			t.Fatalf("pawn should attack %v", s)
		}
	}
	for _, s := range free {
		if b.Attacked(s, chess.White) {
			t.Fatalf("pawn should not attack %v", s)
		}
	}
	if b.Attacked(chess.Sq(1, 2), chess.Black) {
		t.Fatal("black attacks with no black pieces")
	}
}

func TestAttackedSeesThroughNothing(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(0, 0), black(chess.Rook))
	b.Set(chess.Sq(0, 2), white(chess.Pawn)) // blocks at (0,2)
	if !b.Attacked(chess.Sq(0, 2), chess.Black) {
		t.Fatal("a rook attacks the piece that blocks it")
	}
	if b.Attacked(chess.Sq(0, 3), chess.Black) {
		t.Fatal("a rook does not attack past a blocker")
	}
}

func TestInCheck(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(0, 0), black(chess.King))
	b.Set(chess.Sq(0, 3), white(chess.Rook))
	if !b.InCheck(chess.Black) {
		t.Fatal("rook down the file should give check")
	}
	b.Set(chess.Sq(0, 2), white(chess.Pawn)) // now blocked
	if b.InCheck(chess.Black) {
		t.Fatal("a blocked rook gives no check")
	}
	if b.InCheck(chess.White) {
		t.Fatal("a board with no white king is never in check")
	}
}

func TestCheckmate(t *testing.T) {
	b := mateOne()
	if !b.InCheck(chess.Black) || !b.Checkmate(chess.Black) {
		t.Fatal("cornered king with a supported queen should be mate")
	}
	if b.Checkmate(chess.White) {
		t.Fatal("white is not even in check")
	}
}

// The mating move itself: with the queen on b4 the king is not yet in check;
// Qb1 is mate.
func TestAfterPlaysTheMoveOnACopy(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(0, 0), black(chess.King))
	b.Set(chess.Sq(1, 4), white(chess.Queen))
	b.Set(chess.Sq(2, 2), white(chess.King))
	if b.InCheck(chess.Black) || b.Checkmate(chess.Black) {
		t.Fatal("precondition: black should not be in check before the move")
	}
	start := b.At(chess.Sq(1, 4))
	m := chess.Move{From: chess.Sq(1, 4), To: chess.Sq(1, 1)}
	after := b.After(m)
	if after.At(chess.Sq(1, 1)) != start || !after.At(chess.Sq(1, 4)).IsEmpty() {
		t.Fatal("After did not move the piece")
	}
	if b.At(chess.Sq(1, 4)) != start {
		t.Fatal("After changed the original board")
	}
	if !after.Checkmate(chess.Black) {
		t.Fatal("Qb1 should mate the cornered king")
	}
}

func TestKingMayNotStepIntoCheck(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(0, 0), black(chess.King))
	b.Set(chess.Sq(4, 0), white(chess.Rook))
	b.Set(chess.Sq(4, 4), white(chess.King)) // only so white has a king
	var to []chess.Square
	for _, m := range b.LegalMoves(chess.Sq(0, 0)) {
		to = append(to, m.To)
	}
	for _, s := range to {
		if s == chess.Sq(1, 0) {
			t.Fatal("the king may not stay on the rook's rank")
		}
	}
	want := map[chess.Square]bool{chess.Sq(0, 1): true, chess.Sq(1, 1): true}
	if len(to) != len(want) {
		t.Fatalf("legal king moves %v, want %v", to, want)
	}
	for _, s := range to {
		if !want[s] {
			t.Fatalf("unexpected king move to %v", s)
		}
	}
}

func TestKingMayNotStandNextToTheOtherKing(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(0, 0), black(chess.King))
	b.Set(chess.Sq(2, 1), white(chess.King))
	// The white king on c1 takes b0 and b1 away; a1 is still free.
	for _, m := range b.LegalMoves(chess.Sq(0, 0)) {
		if m.To == chess.Sq(1, 1) || m.To == chess.Sq(1, 0) {
			t.Fatalf("king moved to %v, next to the other king", m.To)
		}
	}
	if len(b.LegalMoves(chess.Sq(0, 0))) != 1 {
		t.Fatalf("king should have one move (a1), got %v", b.LegalMoves(chess.Sq(0, 0)))
	}
}

func TestPinnedPieceKeepsToItsLine(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(0, 0), white(chess.King))
	b.Set(chess.Sq(0, 2), white(chess.Rook))
	b.Set(chess.Sq(0, 4), black(chess.Rook))
	b.Set(chess.Sq(4, 4), black(chess.King))
	for _, m := range b.LegalMoves(chess.Sq(0, 2)) {
		if m.From.File != m.To.File {
			t.Fatalf("pinned rook may not leave the file (move to %v)", m.To)
		}
	}
	if len(b.LegalMoves(chess.Sq(0, 2))) != 3 {
		t.Fatalf("pinned rook should have 3 legal moves up and down its file, got %v", b.LegalMoves(chess.Sq(0, 2)))
	}
}

func TestStalemate(t *testing.T) {
	// Black k a0; white Q b2 covers a1, b1, b0 but not a0.
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(0, 0), black(chess.King))
	b.Set(chess.Sq(1, 2), white(chess.Queen))
	b.Set(chess.Sq(4, 4), white(chess.King))
	if b.InCheck(chess.Black) {
		t.Fatal("precondition: black should not be in check")
	}
	if !b.Stalemate(chess.Black) {
		t.Fatal("all three escape squares are covered: stalemate")
	}
	if b.Checkmate(chess.Black) {
		t.Fatal("not in check, so not mate")
	}
}

// Boards without kings (every game before the puzzles) behave exactly as
// before: no check, no mate, and every pseudo-legal move is legal.
func TestNoKingsMeansNoCheckRules(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(0, 0), white(chess.Rook))
	if b.InCheck(chess.White) || b.Checkmate(chess.White) || b.Stalemate(chess.White) {
		t.Fatal("a kingless board has no check rules")
	}
	if len(b.LegalMoves(chess.Sq(0, 0))) != len(b.MoveTargets(chess.Sq(0, 0))) {
		t.Fatal("with no king, legal moves equal pseudo-legal moves")
	}
}
