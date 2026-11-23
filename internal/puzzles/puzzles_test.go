package puzzles

import (
	"strings"
	"testing"

	"github.com/testsabirweb/chess-app/internal/chess"
)

// Every shipped line must be legal, must be the tier it claims, and the stated
// solution must be the move the rules describe.
func TestBankValid(t *testing.T) {
	all := Bank()
	if len(all) == 0 {
		t.Fatal("the bank is empty")
	}
	for i, p := range all {
		if err := p.Valid(); err != nil {
			t.Fatalf("puzzle %d (%s): %v", i, Encode(p), err)
		}
	}
}

func TestBankHasEnoughOfEachTier(t *testing.T) {
	byTier := ByTier()
	want := map[Tier]int{
		Check:     20,
		Mate1:     40,
		Mate1Trap: 30,
		Mate2:     20,
	}
	for tier, n := range want {
		if len(byTier[tier]) < n {
			t.Fatalf("%s: %d puzzles, want at least %d", tier, len(byTier[tier]), n)
		}
	}
}

func TestBankPositionsAreUnique(t *testing.T) {
	seen := map[string]Tier{}
	for _, p := range Bank() {
		key := PositionKey(p.Board)
		if other, dup := seen[key]; dup {
			t.Fatalf("position repeats: %s and %s", other, p.Tier)
		}
		seen[key] = p.Tier
	}
}

// A black pawn on rank 1 could promote during black's reply, and the game has
// no promotion yet, so the bank never places one there.
func TestBankHasNoPawnAboutToPromote(t *testing.T) {
	for _, p := range Bank() {
		for r := 0; r <= 1; r++ {
			for f := 0; f < 5; f++ {
				got := p.Board.At(chess.Sq(f, r))
				if got.Type == chess.Pawn && got.Color == chess.Black {
					t.Fatalf("black pawn on rank %d in %s", r, Encode(p))
				}
			}
		}
	}
}

// In every mate-in-two, whatever black replies, white has a mate.
func TestMateTwoIsForced(t *testing.T) {
	for _, p := range ByTier()[Mate2] {
		after := p.Board.After(p.Solution)
		if !after.InCheck(chess.Black) {
			t.Fatalf("%s: solution gives no check", Encode(p))
		}
		replies := after.LegalMovesFor(chess.Black)
		if len(replies) == 0 {
			t.Fatalf("%s: black has no reply", Encode(p))
		}
		for _, r := range replies {
			if len(MateMoves(after.After(r), chess.White)) == 0 {
				t.Fatalf("%s: reply %v escapes", Encode(p), r)
			}
		}
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	for _, p := range Bank() {
		got, err := Decode(Encode(p))
		if err != nil {
			t.Fatalf("decode %q: %v", Encode(p), err)
		}
		if got.Tier != p.Tier || got.Solution != p.Solution {
			t.Fatalf("round trip changed %q", Encode(p))
		}
		if PositionKey(got.Board) != PositionKey(p.Board) {
			t.Fatalf("round trip changed the board of %q", Encode(p))
		}
	}
}

func TestLoadRejectsBadLines(t *testing.T) {
	bad := []string{
		"m9 | ..... / ..... / ..... / ..... / ..... | a0-b1",
		"m1 | ..... / ..... | a0-b1",
		"m1 | ..... / ..... / ..... / ..... / ....x | a0-b1",
		"m1 | ..... / ..... / ..... / ..... / ..... | z0-b1",
		"m1 | ..... / ..... / ..... / ..... / ..... | a0",
		"nonsense",
	}
	for _, line := range bad {
		if _, err := Load(line); err == nil {
			t.Fatalf("accepted %q", line)
		}
	}
	if _, err := Load("# only comments\n"); err == nil {
		t.Fatal("accepted an empty bank")
	}
}

// Mirror images count as the same position, so the bank does not ship both.
func TestPositionKeyIgnoresMirroring(t *testing.T) {
	a := chess.NewBoard(5, 5)
	a.Set(chess.Sq(0, 0), chess.Piece{Type: chess.King, Color: chess.White})
	a.Set(chess.Sq(4, 4), chess.Piece{Type: chess.King, Color: chess.Black})
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(4, 0), chess.Piece{Type: chess.King, Color: chess.White})
	b.Set(chess.Sq(0, 4), chess.Piece{Type: chess.King, Color: chess.Black})
	if PositionKey(a) != PositionKey(b) {
		t.Fatal("mirrored positions should share a key")
	}
}

// A puzzle that is wrong must be rejected, or bad lines could slip in.
func TestValidRejectsWrongTier(t *testing.T) {
	byTier := ByTier()
	m1 := byTier[Mate1][0]
	m1.Tier = Mate2 // it has a mate in one
	if err := m1.Valid(); err == nil {
		t.Fatal("a mate in one was accepted as a mate in two")
	}
	check := byTier[Check][0]
	check.Tier = Mate1
	if err := check.Valid(); err == nil && len(MateMoves(check.Board, chess.White)) != 1 {
		t.Fatal("a non-mate position was accepted as mate in one")
	}
}

// The bank should read like a human wrote it: readable rows and moves.
func TestBankIsReadable(t *testing.T) {
	data := strings.Split(strings.TrimSpace(Encode(Bank()[0])), " | ")
	if len(data) != 3 || len(strings.Split(data[1], " / ")) != 5 || !strings.Contains(data[2], "-") {
		t.Fatalf("line looks wrong: %q", Encode(Bank()[0]))
	}
}

// The mate-in-three solver must agree with hand-made positions.
func TestForcedMateDepths(t *testing.T) {
	// K+Q v K: queen b4 -> b1 is mate in one; before it, white has mate in one.
	b := chess.NewBoard(5, 5)
	b.Set(chess.Sq(0, 0), chess.Piece{Type: chess.King, Color: chess.Black})
	b.Set(chess.Sq(1, 4), chess.Piece{Type: chess.Queen, Color: chess.White})
	b.Set(chess.Sq(2, 2), chess.Piece{Type: chess.King, Color: chess.White})
	if !ForcedMate(b, chess.White, 1) {
		t.Fatal("queen b4 should mate in one by b1")
	}
	if !ForcedMate(b, chess.White, 3) {
		t.Fatal("a mate in one is also reachable within three moves")
	}
	if ForcedMate(b, chess.Black, 1) {
		t.Fatal("the black king has no mate of its own")
	}
}

// Every mate-in-three in the bank is really forced in three, and not sooner.
func TestBankMateThree(t *testing.T) {
	for _, p := range ByTier()[Mate3] {
		if err := p.Valid(); err != nil {
			t.Fatalf("%s: %v", Encode(p), err)
		}
		if len(MateMoves(p.Board, chess.White)) != 0 {
			t.Fatalf("%s: has a mate in one", Encode(p))
		}
		if ForcedMate(p.Board, chess.White, 2) {
			t.Fatalf("%s: has a mate in two", Encode(p))
		}
		if !ForcedMate(p.Board, chess.White, 3) {
			t.Fatalf("%s: not a forced mate in three", Encode(p))
		}
	}
}

// The bank has a real quota of puzzles where the knight delivers the last
// check, and they must be in the mating tiers only.
func TestBankHasKnightFinishes(t *testing.T) {
	want := map[Tier]int{Mate2: 30, Mate3: 20}
	for tier, n := range want {
		depth := 2
		if tier == Mate3 {
			depth = 3
		}
		got := 0
		for _, p := range ByTier()[tier] {
			if KnightMatesIn(p.Board, chess.White, depth) {
				got++
			}
		}
		if got < n {
			t.Fatalf("%s: %d knight-finish puzzles, want at least %d", tier, got, n)
		}
	}
	// No knight puzzles in the earliest tiers: they have no knights at all.
	for _, p := range append(ByTier()[Check], append(ByTier()[Mate1], ByTier()[Mate1Trap]...)...) {
		for _, sq := range p.Board.Occupied() {
			if p.Board.At(sq).Type == chess.Knight {
				t.Fatalf("%s has a knight", Encode(p))
			}
		}
	}
}

// The generator now prefers the back rank and a pawn wall in front of the king,
// the patterns a beginner is taught first.
func TestBankPrefersBackRankPatterns(t *testing.T) {
	check := func(tier Tier) (back, wall int) {
		for _, p := range ByTier()[tier] {
			k, _ := p.Board.KingSquare(chess.Black)
			if int(k.Rank) != 4 {
				continue
			}
			back++
			for df := -1; df <= 1; df++ {
				f := int(k.File) + df
				if f < 0 || f >= 5 {
					continue
				}
				pc := p.Board.At(chess.Sq(f, 3))
				if pc.Type == chess.Pawn && pc.Color == chess.Black {
					wall++
					break
				}
			}
		}
		return back, wall
	}
	for _, tier := range []Tier{Mate1, Mate2, Mate3} {
		list := ByTier()[tier]
		back, wall := check(tier)
		if back < len(list)*40/100 {
			t.Fatalf("%s: only %d/%d kings on the back rank", tier, back, len(list))
		}
		if wall < len(list)*10/100 {
			t.Fatalf("%s: only %d/%d kings have a pawn wall", tier, wall, len(list))
		}
	}
}
