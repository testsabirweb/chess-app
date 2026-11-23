package game

import (
	"testing"

	"github.com/testsabirweb/chess-app/internal/chess"
	"github.com/testsabirweb/chess-app/internal/layout"
)

func TestPawnWarsStartIsNotOver(t *testing.T) {
	if _, _, over := pawnWarsResult(pawnWarsSetup(), chess.White); over {
		t.Fatal("the starting position is already over")
	}
}

func TestPawnWarsPromotionWins(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(2, 4), chess.Piece{Type: chess.Pawn, Color: chess.White})
	b.Set(chess.Sq(0, 3), chess.Piece{Type: chess.Pawn, Color: chess.Black})
	if w, draw, over := pawnWarsResult(b, chess.White); !over || draw || w != chess.White {
		t.Fatalf("white pawn on the far rank: winner=%v draw=%v over=%v", w, draw, over)
	}

	b = chess.NewBoard(5, 5)
	b.Set(chess.Sq(2, 0), chess.Piece{Type: chess.Pawn, Color: chess.Black})
	b.Set(chess.Sq(0, 1), chess.Piece{Type: chess.Pawn, Color: chess.White})
	if w, draw, over := pawnWarsResult(b, chess.Black); !over || draw || w != chess.Black {
		t.Fatalf("black pawn on the far rank: winner=%v draw=%v over=%v", w, draw, over)
	}
}

func TestPawnWarsLosingTheLastPawnLoses(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(2, 2), chess.Piece{Type: chess.Pawn, Color: chess.White})
	if w, draw, over := pawnWarsResult(b, chess.White); !over || draw || w != chess.White {
		t.Fatalf("black has no pawns: winner=%v draw=%v over=%v", w, draw, over)
	}
	b = chess.NewBoard(5, 5)
	b.Set(chess.Sq(2, 2), chess.Piece{Type: chess.Pawn, Color: chess.Black})
	if w, draw, over := pawnWarsResult(b, chess.Black); !over || draw || w != chess.Black {
		t.Fatalf("white has no pawns: winner=%v draw=%v over=%v", w, draw, over)
	}
}

// Two pawns facing each other with nothing to take: nobody can move, so
// everybody wins.
func TestPawnWarsBlockedIsADraw(t *testing.T) {
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(2, 2), chess.Piece{Type: chess.Pawn, Color: chess.White})
	b.Set(chess.Sq(2, 3), chess.Piece{Type: chess.Pawn, Color: chess.Black})
	w, draw, over := pawnWarsResult(b, chess.White)
	if !over || !draw || w != noWinner {
		t.Fatalf("blocked position: winner=%v draw=%v over=%v", w, draw, over)
	}
}

func newPawnWars(g *Game) *PawnWarsScene {
	g.ctx.M = layout.Compute(1080, 2400, 2.75, layout.Insets{}, 5, 5)
	g.ctx.DT = 1.0 / 60
	return NewPawnWarsScene(g)
}

func pwPress(t *testing.T, s *PawnWarsScene, g *Game, sq chess.Square) {
	t.Helper()
	tapCell(g, g.ctx.M, int(sq.File), int(sq.Rank))
	if err := s.Update(&g.ctx); err != nil {
		t.Fatal(err)
	}
	g.pointer.JustPressed = nil
}

func pwSettle(t *testing.T, s *PawnWarsScene, g *Game) {
	t.Helper()
	for i := 0; i < 1500; i++ {
		if s.state == pwIdle && !s.kit.busy() {
			return
		}
		if err := s.Update(&g.ctx); err != nil {
			t.Fatal(err)
		}
	}
	t.Fatalf("pawn wars never settled (state=%v busy=%v)", s.state, s.kit.busy())
}

func pwPlay(t *testing.T, s *PawnWarsScene, g *Game, from, to chess.Square) {
	t.Helper()
	pwPress(t, s, g, from)
	pwPress(t, s, g, to)
	pwSettle(t, s, g)
}

// A move hands the turn over.
func TestPawnWarsTurnsAlternate(t *testing.T) {
	g := testGame()
	s := newPawnWars(g)
	pwPlay(t, s, g, chess.Sq(1, 1), chess.Sq(1, 2))
	if s.turn != chess.Black {
		t.Fatalf("turn = %v after white moved, want black", s.turn)
	}
	// Black takes the pawn diagonally.
	pwPlay(t, s, g, chess.Sq(2, 3), chess.Sq(1, 2))
	if s.turn != chess.White {
		t.Fatalf("turn = %v after black moved, want white", s.turn)
	}
}

// The pawns cannot be touched out of turn.
func TestPawnWarsCannotMoveTheOtherSide(t *testing.T) {
	g := testGame()
	s := newPawnWars(g)
	pwPress(t, s, g, chess.Sq(1, 3)) // a black pawn on white's turn
	if s.hasSel {
		t.Fatal("black pawn was picked up on white's turn")
	}
	if s.wobbleT <= 0 {
		t.Fatal("no wobble for touching the other side")
	}
}

// Taking the last pawn wins, and wins the sticker.
func TestPawnWarsWhiteWinsByCapture(t *testing.T) {
	g := testGame()
	s := newPawnWars(g)
	s.board = chess.NewBoard(5, 5)
	s.board.Set(chess.Sq(1, 1), chess.Piece{Type: chess.Pawn, Color: chess.White})
	s.board.Set(chess.Sq(2, 2), chess.Piece{Type: chess.Pawn, Color: chess.Black})
	s.turn = chess.White

	pwPress(t, s, g, chess.Sq(1, 1))
	pwPress(t, s, g, chess.Sq(2, 2))
	pwSettle(t, s, g)
	if got := len(g.Stickers()); got != 1 {
		t.Fatalf("got %d stickers for white's win, want 1", got)
	}
}

// Black winning is gentle: a banner, confetti on black's side, no sticker, and
// a fresh game.
func TestPawnWarsBlackWinsWithoutASticker(t *testing.T) {
	g := testGame()
	s := newPawnWars(g)
	s.board = chess.NewBoard(5, 5)
	s.board.Set(chess.Sq(0, 1), chess.Piece{Type: chess.Pawn, Color: chess.White})
	s.board.Set(chess.Sq(2, 1), chess.Piece{Type: chess.Pawn, Color: chess.Black})
	s.turn = chess.Black

	for i := 0; i < 60; i++ { // one tap, then the hop
		if i == 0 {
			pwPress(t, s, g, chess.Sq(2, 1))
		} else if i == 1 {
			tapCell(g, g.ctx.M, 2, 0)
		}
		if err := s.Update(&g.ctx); err != nil {
			t.Fatal(err)
		}
		g.pointer.JustPressed = nil
	}
	if s.state != pwBlackWon {
		t.Fatalf("state = %v, want black-won", s.state)
	}
	if s.banner != "Black wins!" {
		t.Fatalf("banner = %q", s.banner)
	}
	for i := 0; i < 300; i++ {
		if err := s.Update(&g.ctx); err != nil {
			t.Fatal(err)
		}
	}
	if s.state != pwIdle || s.turn != chess.White {
		t.Fatalf("after the banner: state=%v turn=%v, want a fresh game", s.state, s.turn)
	}
	if got := len(s.board.Occupied()); got != 6 {
		t.Fatalf("%d pawns after the reset, want 6", got)
	}
	if len(g.Stickers()) != 0 {
		t.Fatal("black's win gave out a sticker")
	}
}

// A blocked game is a draw, and everyone wins.
func TestPawnWarsBlockedIsADrawWithASticker(t *testing.T) {
	g := testGame()
	s := newPawnWars(g)
	s.board = chess.NewBoard(5, 5)
	s.board.Set(chess.Sq(2, 2), chess.Piece{Type: chess.Pawn, Color: chess.White})
	s.board.Set(chess.Sq(2, 4), chess.Piece{Type: chess.Pawn, Color: chess.Black})
	s.turn = chess.Black

	pwPress(t, s, g, chess.Sq(2, 4))
	pwPress(t, s, g, chess.Sq(2, 3)) // blocks white completely
	for i := 0; i < 60; i++ {
		if err := s.Update(&g.ctx); err != nil {
			t.Fatal(err)
		}
	}
	if s.banner != "Everyone wins!" {
		t.Fatalf("banner = %q, want a draw", s.banner)
	}
	pwSettle(t, s, g)
	if got := len(g.Stickers()); got != 1 {
		t.Fatalf("got %d stickers for the draw, want 1", got)
	}
}

// The starting board is the three middle files, three pawns each.
func TestPawnWarsSetup(t *testing.T) {
	b := pawnWarsSetup()
	white, black := 0, 0
	for _, sq := range b.Occupied() {
		p := b.At(sq)
		if p.Type != chess.Pawn {
			t.Fatalf("non-pawn on the board at %v", sq)
		}
		switch p.Color {
		case chess.White:
			white++
			if sq.Rank != 1 {
				t.Fatalf("white pawn off its rank at %v", sq)
			}
		case chess.Black:
			black++
			if sq.Rank != 3 {
				t.Fatalf("black pawn off its rank at %v", sq)
			}
		}
	}
	if white != 3 || black != 3 {
		t.Fatalf("white %d black %d, want 3 each", white, black)
	}
}
