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
