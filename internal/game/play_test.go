package game

import (
	"math/rand/v2"
	"testing"

	"github.com/testsabirweb/chess-app/internal/challenge"
	"github.com/testsabirweb/chess-app/internal/chess"
	"github.com/testsabirweb/chess-app/internal/layout"
	"github.com/testsabirweb/chess-app/internal/render"
	"github.com/testsabirweb/chess-app/internal/reward"
)

func testGame() *Game {
	g := &Game{
		reward: reward.NewPicker(render.RewardEmojiIndices(), rand.New(rand.NewPCG(1, 2))),
		ctx:    Context{Rand: rand.New(rand.NewPCG(3, 4))},
	}
	g.ctx.Pointer = &g.pointer
	return g
}

// Landing on the star must collect it exactly once. A regression left the scene
// in stateMoving after the win, so every following frame landed again and the
// reward popped over and over.
func TestStarIsCollectedOnce(t *testing.T) {
	g := testGame()
	p := NewPlayScene(g, chess.Rook)
	m := layout.Compute(1080, 2400, 2.75, layout.Insets{}, 5, 5)
	ctx := &g.ctx
	ctx.M = m
	ctx.DT = 1.0 / 60

	// Put the rook one step from the star and hop it there.
	p.board.Set(p.at, chess.Piece{})
	p.at = chess.Sq(0, 0)
	p.targets = []chess.Square{chess.Sq(0, 3)}
	p.board.Set(p.at, p.cur.Piece)
	p.solutions = p.board.MoveTargets(p.at)
	p.steps = 1
	p.optimal = 1
	p.laidOut = false
	p.pieceSelected = true
	p.syncPiecePos(m)
	p.startMove(chess.Sq(0, 3), m)

	for i := 0; i < 600; i++ {
		if err := p.Update(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(g.Stickers()); got != 1 {
		t.Fatalf("collected %d stickers from one star, want 1", got)
	}
}

// fixedPuzzle is a rook with two stars, each one move apart in turn:
// (0,0) -> (0,3) -> (4,3).
func fixedPuzzle() challenge.Puzzle {
	b := chess.NewBoard(5, 5)
	piece := chess.Piece{Type: chess.Rook, Color: chess.White}
	b.Set(chess.Sq(0, 0), piece)
	return challenge.Puzzle{
		Board: b, From: chess.Sq(0, 0), Piece: piece,
		Targets: []chess.Square{chess.Sq(0, 3), chess.Sq(4, 3)}, Optimal: 2,
	}
}

func hop(t *testing.T, g *Game, p *PlayScene, to chess.Square, frames int) {
	t.Helper()
	p.pieceSelected = true
	p.startMove(to, g.ctx.M)
	for i := 0; i < frames; i++ {
		if err := p.Update(&g.ctx); err != nil {
			t.Fatal(err)
		}
	}
}

// Several stars: each one is collected as the piece lands on it, and only the
// last one wins the round.
func TestTreasureNeedsEveryStar(t *testing.T) {
	g := testGame()
	g.ctx.M = layout.Compute(1080, 2400, 2.75, layout.Insets{}, 5, 5)
	g.ctx.DT = 1.0 / 60
	served := false
	p := newPlayScene(g, chess.Rook, func() challenge.Puzzle {
		if served {
			return challenge.NewTreasure(g.ctx.Rand, chess.Rook, chess.White, 2, nil)
		}
		served = true
		return fixedPuzzle()
	})

	hop(t, g, p, chess.Sq(0, 3), 60)
	if len(p.targets) != 1 || len(g.Stickers()) != 0 || p.kit.busy() {
		t.Fatalf("after one star: %d left, %d stickers, busy=%v", len(p.targets), len(g.Stickers()), p.kit.busy())
	}

	hop(t, g, p, chess.Sq(4, 3), 600)
	if got := len(g.Stickers()); got != 1 {
		t.Fatalf("got %d stickers after the last star, want exactly 1", got)
	}
}

func newFixedTreasure(g *Game) *PlayScene {
	g.ctx.M = layout.Compute(1080, 2400, 2.75, layout.Insets{}, 5, 5)
	g.ctx.DT = 1.0 / 60
	return newPlayScene(g, chess.Rook, func() challenge.Puzzle { return fixedPuzzle() })
}

// Clearing every star in the fewest moves earns the bigger celebration.
func TestTreasureShortestRouteIsPerfect(t *testing.T) {
	g := testGame()
	p := newFixedTreasure(g)
	hop(t, g, p, chess.Sq(0, 3), 40)
	hop(t, g, p, chess.Sq(4, 3), 30) // lands at ~25 frames, before the next puzzle is dealt
	if !p.kit.celebrating() || !p.kit.perfect {
		t.Fatalf("celebrating=%v perfect=%v, want a perfect win in 2 moves", p.kit.celebrating(), p.kit.perfect)
	}
}

// Wandering still wins, it just is not "perfect".
func TestTreasureLongRouteIsNotPerfect(t *testing.T) {
	g := testGame()
	p := newFixedTreasure(g)
	hop(t, g, p, chess.Sq(4, 0), 40)
	hop(t, g, p, chess.Sq(4, 3), 40)
	if len(p.targets) != 0 && len(p.targets) != 1 {
		t.Fatalf("%d stars left", len(p.targets))
	}
	hop(t, g, p, chess.Sq(0, 3), 30)
	if !p.kit.celebrating() || p.kit.perfect {
		t.Fatalf("celebrating=%v perfect=%v, want a plain win after 3 moves", p.kit.celebrating(), p.kit.perfect)
	}
}

// A star the piece can no longer reach hops somewhere it can, and the round
// stays winnable.
func TestStarsStayReachable(t *testing.T) {
	g := testGame()
	g.ctx.M = layout.Compute(1080, 2400, 2.75, layout.Insets{}, 5, 5)
	g.ctx.DT = 1.0 / 60
	b := chess.NewBoard(5, 5)
	piece := chess.Piece{Type: chess.Pawn, Color: chess.White}
	b.Set(chess.Sq(2, 1), piece)
	p := newPlayScene(g, chess.Pawn, func() challenge.Puzzle {
		return challenge.Puzzle{
			Board: b.Clone(), From: chess.Sq(2, 1), Piece: piece,
			Targets: []chess.Square{chess.Sq(2, 3), chess.Sq(2, 4)}, Optimal: 3,
		}
	})
	// The pawn steps up and past nothing, but a star behind it would be lost; put
	// one there directly and let the safety net notice.
	p.targets = []chess.Square{chess.Sq(2, 0), chess.Sq(2, 4)}
	p.at = chess.Sq(2, 2)
	p.board.Set(chess.Sq(2, 1), chess.Piece{})
	p.board.Set(p.at, piece)
	p.keepStarsReachable(&g.ctx)
	for i, tgt := range p.targets {
		if !challenge.CanReach(p.board, p.at, tgt, maxJourney+2) {
			t.Fatalf("star %d at %v is still out of reach", i, tgt)
		}
	}
}

// fixedCatch is a rook with two black pawns to take: (0,0) x (0,2) x (4,2).
func fixedCatch() challenge.Puzzle {
	b := chess.NewBoard(5, 5)
	piece := chess.Piece{Type: chess.Rook, Color: chess.White}
	enemy := chess.Piece{Type: chess.Pawn, Color: chess.Black}
	b.Set(chess.Sq(0, 0), piece)
	b.Set(chess.Sq(0, 2), enemy)
	b.Set(chess.Sq(4, 2), enemy)
	return challenge.Puzzle{
		Board: b, From: chess.Sq(0, 0), Piece: piece,
		Targets: []chess.Square{chess.Sq(0, 2), chess.Sq(4, 2)}, Optimal: 2,
	}
}

func newFixedCatch(g *Game) *PlayScene {
	g.ctx.M = layout.Compute(1080, 2400, 2.75, layout.Insets{}, 5, 5)
	g.ctx.DT = 1.0 / 60
	ps := newPlayScene(g, chess.Rook, func() challenge.Puzzle { return fixedCatch() })
	return ps
}

// Landing on a pawn takes it off the board; the round is won when the last one
// is gone, and only then.
func TestCatchRemovesEachPawn(t *testing.T) {
	g := testGame()
	p := newFixedCatch(g)

	hop(t, g, p, chess.Sq(0, 2), 40)
	if !p.board.At(chess.Sq(0, 2)).IsEmpty() && p.board.At(chess.Sq(0, 2)).Color == chess.Black {
		t.Fatal("the captured pawn is still on the board")
	}
	if len(p.targets) != 1 || len(g.Stickers()) != 0 {
		t.Fatalf("after one pawn: %d left, %d stickers", len(p.targets), len(g.Stickers()))
	}

	hop(t, g, p, chess.Sq(4, 2), 30)
	if !p.kit.celebrating() || !p.kit.perfect {
		t.Fatalf("celebrating=%v perfect=%v, want a perfect win in 2 moves", p.kit.celebrating(), p.kit.perfect)
	}
	for i := 0; i < 600; i++ {
		if err := p.Update(&g.ctx); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(g.Stickers()); got != 1 {
		t.Fatalf("got %d stickers, want exactly 1", got)
	}
}

// A pawn the piece can no longer reach is picked up and set down somewhere it
// can: the target list and the board must agree.
func TestCaughtPawnsStayReachable(t *testing.T) {
	g := testGame()
	g.ctx.M = layout.Compute(1080, 2400, 2.75, layout.Insets{}, 5, 5)
	piece := chess.Piece{Type: chess.Pawn, Color: chess.White}
	enemy := chess.Piece{Type: chess.Pawn, Color: chess.Black}
	p := newPlayScene(g, chess.Pawn, func() challenge.Puzzle {
		b := chess.NewBoard(5, 5)
		b.Set(chess.Sq(2, 1), piece)
		b.Set(chess.Sq(3, 2), enemy)
		return challenge.Puzzle{Board: b, From: chess.Sq(2, 1), Piece: piece, Targets: []chess.Square{chess.Sq(3, 2)}, Optimal: 1}
	})
	// The white pawn has climbed past the black one, which it can never take now.
	p.board.Set(chess.Sq(2, 1), chess.Piece{})
	p.at = chess.Sq(2, 3)
	p.board.Set(p.at, piece)
	p.keepStarsReachable(&g.ctx)

	if len(p.targets) != 1 {
		t.Fatalf("%d targets, want 1", len(p.targets))
	}
	to := p.targets[0]
	if got := p.board.At(to); got.Type != chess.Pawn || got.Color != chess.Black {
		t.Fatalf("no black pawn on the new target %v", to)
	}
	if !p.board.At(chess.Sq(3, 2)).IsEmpty() {
		t.Fatal("the old pawn square was not cleared")
	}
	if !challenge.CanReach(p.board, p.at, to, maxJourney+2) {
		t.Fatalf("pawn at %v is still out of reach", to)
	}
	if got := len(p.board.Occupied()); got != 2 {
		t.Fatalf("%d pieces on the board, want 2", got)
	}
}

// settle runs the scene until nothing is moving and no celebration is on.
func settle(t *testing.T, g *Game, p *PlayScene) {
	t.Helper()
	for i := 0; i < 1500 && (p.state != stateIdle || p.kit.busy()); i++ {
		if err := p.Update(&g.ctx); err != nil {
			t.Fatal(err)
		}
	}
	if p.state != stateIdle || p.kit.busy() {
		t.Fatal("scene never settled")
	}
}

// Random play in the several-target games, with every piece: whatever the
// child does, the board and the target list stay consistent and every target
// can still be reached.
func TestMultiTargetRandomPlayStaysConsistent(t *testing.T) {
	types := []chess.PieceType{chess.Pawn, chess.Knight, chess.Bishop, chess.Rook, chess.Queen, chess.King}
	for _, mode := range []string{"treasure", "catch", "safe"} {
		for _, pt := range types {
			g := testGame()
			g.ctx.M = layout.Compute(1080, 2400, 2.75, layout.Insets{}, 5, 5)
			g.ctx.DT = 1.0 / 60
			var p *PlayScene
			switch mode {
			case "catch":
				// Rounds of nothing but pawns, which the alternating game only
				// reaches every other round.
				p = newPlayScene(g, pt, multiSource(g, func(_, k int, mem *challenge.Memory) challenge.Puzzle {
					return challenge.NewCatch(g.ctx.Rand, pt, chess.White, k, mem)
				}))
			case "safe":
				p = NewSafeScene(g, pt)
			default:
				p = NewTreasureScene(g, pt)
			}
			rng := rand.New(rand.NewPCG(uint64(pt), 99))
			for move := 0; move < 120; move++ {
				settle(t, g, p)
				if len(p.solutions) == 0 {
					continue
				}
				// Invariants.
				pieces := len(p.board.Occupied())
				want := 1 // the player's piece
				if p.capture {
					want += len(p.targets)
				}
				if p.safe {
					want++ // the guard
					if p.avoid(p.at) {
						t.Fatalf("%v move %d: the piece is standing on a guarded square %v", pt, move, p.at)
					}
					if g := p.board.At(p.guard); g.IsEmpty() || g.Color != chess.Black {
						t.Fatalf("%v move %d: the guard is gone from %v", pt, move, p.guard)
					}
				}
				if pieces != want {
					t.Fatalf("mode=%v %v move %d: %d pieces on the board, want %d", mode, pt, move, pieces, want)
				}
				for _, tg := range p.targets {
					if p.capture {
						if got := p.board.At(tg); got.Type != chess.Pawn || got.Color != chess.Black {
							t.Fatalf("mode=%v %v: target %v holds no black pawn", mode, pt, tg)
						}
					} else if !p.board.At(tg).IsEmpty() {
						t.Fatalf("mode=%v %v: star %v is under a piece", mode, pt, tg)
					}
					if !challenge.CanReachAvoiding(p.board, p.at, tg, maxJourney+2, p.avoid) {
						t.Fatalf("mode=%v %v move %d: target %v unreachable from %v", mode, pt, move, tg, p.at)
					}
				}
				to := p.solutions[rng.IntN(len(p.solutions))]
				if p.avoid != nil && p.avoid(to) {
					// The scene turns this tap down, so the piece stays where it is.
					p.refuse(&g.ctx, to, g.ctx.M)
					continue
				}
				p.pieceSelected = true
				p.startMove(to, g.ctx.M)
			}
		}
	}
}

// fixedSafe is a rook with a guard rook at (4,2). With the player lifted, the
// guard attacks all of rank 2 and all of file 4, so (0,2) is hot while (0,3) is
// safe; the star at (1,3) is two safe moves away.
func fixedSafe() challenge.Puzzle {
	b := chess.NewBoard(5, 5)
	piece := chess.Piece{Type: chess.Rook, Color: chess.White}
	guard := chess.Sq(4, 2)
	b.Set(chess.Sq(0, 0), piece)
	b.Set(guard, chess.Piece{Type: chess.Rook, Color: chess.Black})
	lifted := b.Clone()
	lifted.Set(chess.Sq(0, 0), chess.Piece{})
	return challenge.Puzzle{
		Board: b, From: chess.Sq(0, 0), Piece: piece,
		Targets: []chess.Square{chess.Sq(1, 3)}, Optimal: 2,
		Guard: guard, Hot: lifted.Attacks(guard),
	}
}

func newFixedSafe(g *Game) *PlayScene {
	g.ctx.M = layout.Compute(1080, 2400, 2.75, layout.Insets{}, 5, 5)
	g.ctx.DT = 1.0 / 60
	return newPlayScene(g, chess.Rook, func() challenge.Puzzle { return fixedSafe() }, func(p *PlayScene) { p.safe = true })
}

// Tapping a square the guard attacks is refused: the piece stays put, nothing is
// collected, and the warning flashes. A safe square works as normal.
func TestSafeRefusesGuardedSquares(t *testing.T) {
	g := testGame()
	p := newFixedSafe(g)

	p.pieceSelected = true
	tapCell(g, g.ctx.M, 0, 2) // hot
	if err := p.Update(&g.ctx); err != nil {
		t.Fatal(err)
	}
	g.pointer.JustPressed = nil
	if p.state != stateIdle || p.at != chess.Sq(0, 0) {
		t.Fatalf("the piece moved onto a guarded square: state=%v at=%v", p.state, p.at)
	}
	if p.flashT <= 0 || p.lungeT <= 0 {
		t.Fatalf("no warning after a refused move: flash=%v lunge=%v", p.flashT, p.lungeT)
	}

	hop(t, g, p, chess.Sq(0, 3), 40) // safe
	if p.at != chess.Sq(0, 3) {
		t.Fatalf("a safe move was refused: at=%v", p.at)
	}
	hop(t, g, p, chess.Sq(1, 3), 30)
	if !p.kit.celebrating() || !p.kit.perfect {
		t.Fatalf("celebrating=%v perfect=%v, want a perfect win in 2 safe moves", p.kit.celebrating(), p.kit.perfect)
	}
}

// For the first rounds the red squares are always on show; afterwards they come
// with the move dots.
func TestSafeWarningFadesWithTheTeachingRounds(t *testing.T) {
	g := testGame()
	p := newFixedSafe(g)
	if p.safeRound > safeTeachRounds || p.dangerStrength() != 1 {
		t.Fatalf("round %d: strength %v, want the warning fully on show", p.safeRound, p.dangerStrength())
	}
	p.safeRound = safeTeachRounds + 1
	if got := p.dangerStrength(); got != 0 {
		t.Fatalf("after the teaching rounds, with nothing picked up, strength = %v, want 0", got)
	}
	p.pieceSelected = true
	p.hintT = 0
	if got := p.dangerStrength(); got != 1 {
		t.Fatalf("with the dots showing, strength = %v, want 1", got)
	}
}

// The collect game alternates stars and pawns: a round is stars when the target
// squares are empty and pawns when pieces stand on them.
func TestCollectAlternatesStarsAndPawns(t *testing.T) {
	g := testGame()
	g.ctx.M = layout.Compute(1080, 2400, 2.75, layout.Insets{}, 5, 5)
	p := NewTreasureScene(g, chess.Rook)
	var kinds []bool
	for round := 0; round < 6; round++ {
		kinds = append(kinds, p.capture)
		for _, tg := range p.targets {
			if p.board.At(tg).IsEmpty() == p.capture {
				t.Fatalf("round %d: capture=%v but target %v holds %v", round, p.capture, tg, p.board.At(tg))
			}
		}
		p.newChallenge()
	}
	for i, c := range kinds {
		if want := i%2 == 1; c != want {
			t.Fatalf("round %d: capture=%v, want %v (stars first, then pawns, alternating): %v", i, c, want, kinds)
		}
	}
}
