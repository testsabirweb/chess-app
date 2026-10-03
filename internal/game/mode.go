package game

import "github.com/testsabirweb/chess-app/internal/chess"

// Mode is one of the games on offer. Which one is played is a grown-up's
// choice, made on the home screen; the game never moves on by itself.
type Mode uint8

const (
	ModeStar     Mode = iota // the original game: find the star
	ModeWhich                // which piece can reach the star?
	ModeTreasure             // collect several stars
	ModeCatch                // capture the black pawns
	ModeSafe                 // reach the star without stopping where the guard attacks
	ModePawnWars             // two players, one phone
)

type modeInfo struct {
	name string
	icon string // Twemoji code, as in render.EmojiName
	// needsPiece is true when the player picks which piece to move. Modes that
	// deal their own pieces start the moment their button is tapped.
	needsPiece bool
}

var modeTable = [...]modeInfo{
	ModeStar:     {"Find the star", "2b50", true},
	ModeWhich:    {"Which piece?", "2753", false},
	ModeTreasure: {"Collect the stars", "1f48e", true},
	ModeCatch:    {"Catch the pawns", "265f", true},
	ModeSafe:     {"Stay safe", "1f6e1", true},
	ModePawnWars: {"Pawn Wars", "1f91d", false},
}

func (m Mode) info() modeInfo { return modeTable[m] }

// activeModes are the modes that can be played, in order. A mode is added here
// in the same change that makes it playable, so there is never a button that
// does nothing.
var activeModes = []Mode{ModeStar, ModeWhich, ModeTreasure, ModeCatch}

// rowModes are the modes that get a button on the home screen. The star game
// has none: it is what the piece cards already play, so a button for it would
// only be a second way to do the same thing. When no other game exists the row
// is not drawn at all.
func rowModes() []Mode {
	out := make([]Mode, 0, len(activeModes))
	for _, m := range activeModes {
		if m != ModeStar {
			out = append(out, m)
		}
	}
	return out
}

// beginnerPieces are the pieces the modes that deal their own pieces use. The
// knight is left out on purpose: its L is the one move he has not got yet, and
// a mode that keeps springing it on him would turn a game he can win into one
// he guesses at. Picking the knight card on the home screen still works.
var beginnerPieces = []chess.PieceType{chess.Pawn, chess.Bishop, chess.Rook, chess.Queen, chess.King}

// newModeScene builds the scene for a mode. pt is the piece he picked, and is
// ignored by the modes that deal their own.
func newModeScene(g *Game, mode Mode, pt chess.PieceType) Scene {
	switch mode {
	case ModeWhich:
		return NewWhichScene(g)
	case ModeTreasure:
		return NewTreasureScene(g, pt)
	case ModeCatch:
		return NewCatchScene(g, pt)
	default:
		return NewPlayScene(g, pt)
	}
}
