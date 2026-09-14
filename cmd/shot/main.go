// Command shot drives the game on the desktop and writes PNG screenshots at
// scripted frames. It is a development aid for iterating on the look without
// needing a phone in hand.
//
//	go run ./cmd/shot -out /tmp/shots -scene play -piece rook
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/testsabirweb/chess-app/internal/challenge"
	"github.com/testsabirweb/chess-app/internal/chess"
	"github.com/testsabirweb/chess-app/internal/game"
)

type step struct {
	frame int
	do    func(g *game.Game)
	shot  string
}

type shotGame struct {
	g       *game.Game
	off     *ebiten.Image
	frame   int
	steps   []step
	outDir  string
	w, h    int
	pending []string
}

func (s *shotGame) Update() error {
	for _, st := range s.steps {
		if st.frame == s.frame && st.do != nil {
			st.do(s.g)
		}
	}
	s.frame++
	if s.frame > s.lastFrame()+2 {
		return ebiten.Termination
	}
	return s.g.Update()
}

func (s *shotGame) lastFrame() int {
	last := 0
	for _, st := range s.steps {
		if st.frame > last {
			last = st.frame
		}
	}
	return last
}

func (s *shotGame) Draw(screen *ebiten.Image) {
	b := screen.Bounds()
	if s.off == nil || s.off.Bounds() != b {
		s.off = ebiten.NewImage(b.Dx(), b.Dy())
	}
	s.off.Clear()
	s.g.Draw(s.off)
	screen.DrawImage(s.off, nil)

	for _, st := range s.steps {
		if st.shot == "" || st.frame != s.frame-1 {
			continue
		}
		s.save(st.shot)
	}
}

func (s *shotGame) save(name string) {
	b := s.off.Bounds()
	buf := make([]byte, 4*b.Dx()*b.Dy())
	s.off.ReadPixels(buf)
	img := &image.RGBA{Pix: buf, Stride: 4 * b.Dx(), Rect: image.Rect(0, 0, b.Dx(), b.Dy())}
	p := filepath.Join(s.outDir, name+".png")
	f, err := os.Create(p)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println("wrote", p)
}

func (s *shotGame) Layout(w, h int) (int, int) { return w, h }

func (s *shotGame) LayoutF(w, h float64) (float64, float64) {
	return s.g.LayoutF(w, h)
}

func tapPiece(g *game.Game) {
	if info, ok := g.PlayInfo(); ok {
		g.TapSquare(int(info.Piece.File), int(info.Piece.Rank))
	}
}

// tapTowardStar walks the piece greedily towards the target so the script can
// reach the celebration on multi-move challenges.
func tapTowardStar(g *game.Game) {
	info, ok := g.PlayInfo()
	if !ok || len(info.Hints) == 0 {
		return
	}
	pick := info.Hints[0]
	best := 1 << 30
	for _, s := range info.Hints {
		if s == info.Target {
			pick = s
			break
		}
		// Walk the real move graph rather than guessing by distance.
		b := info.Board.Clone()
		p := b.At(info.Piece)
		b.Set(info.Piece, chess.Piece{})
		b.Set(s, p)
		d := challenge.MovesTo(b, s, info.Target, 4)
		if d > 0 && d < best {
			best, pick = d, s
		}
	}
	if !info.Selected {
		g.TapSquare(int(info.Piece.File), int(info.Piece.Rank))
	}
	g.TapSquare(int(pick.File), int(pick.Rank))
}

func pieceByName(n string) chess.PieceType {
	switch strings.ToLower(n) {
	case "pawn":
		return chess.Pawn
	case "knight":
		return chess.Knight
	case "bishop":
		return chess.Bishop
	case "queen":
		return chess.Queen
	case "king":
		return chess.King
	default:
		return chess.Rook
	}
}

func main() {
	out := flag.String("out", "shots", "directory for PNGs")
	scene := flag.String("scene", "home", "home or play")
	piece := flag.String("piece", "rook", "piece for the play scene")
	skip := flag.Int("skip", 0, "skip this many generated puzzles first")
	seed := flag.Int("stickers", 0, "pre-seed this many collected stickers")
	dpScale := flag.Float64("scale", 0, "override the dp scale (e.g. 2.75 for a Motorola Edge 50 Neo)")
	hintDelay := flag.Float64("hintdelay", 0, "seconds a picked-up piece waits before its moves show (0 = no wait)")
	w := flag.Int("w", 432, "window width")
	h := flag.Int("h", 960, "window height")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		panic(err)
	}

	var g *game.Game
	var steps []step
	if *scene == "play" {
		g = game.NewInPlay(pieceByName(*piece))
		// The pause before the hints is off by default: these shots are for
		// looking at the hinted board, not for sitting through the wait. Pass
		// -hintdelay to watch the dots actually arrive.
		g.SetHintDelay(*hintDelay)
		g.SeedStickers(*seed)
		for i := 0; i < *skip; i++ {
			g.NextChallenge()
		}
		// wait is how long the script must hold after picking the piece up
		// before the dots are on screen. Only the hint shot needs it; the moves
		// themselves go through whether the dots are showing or not.
		wait := int(*hintDelay * 60)
		// Most pieces hop in 0.42s (~25 frames at 60fps); the knight's L-move
		// takes 0.62s (~38 frames). Later tap/shot frames scale from moveDur so
		// the script does not tap mid-animation.
		moveDur := 25
		if strings.ToLower(*piece) == "knight" {
			moveDur = 38
		}
		base := 60 + wait
		steps = []step{
			{frame: 30, shot: "01-idle"},
			{frame: 34, do: tapPiece},
			{frame: 46, shot: "02a-looking"},
			{frame: 55 + wait, shot: "02-hints"},
			{frame: base, do: tapTowardStar},
			{frame: base + 10, shot: "03-moving"},
			{frame: base + moveDur + 8, do: tapTowardStar},
			{frame: base + moveDur + 20, shot: "04-second-hop"},
			{frame: base + 2*moveDur + 16, do: tapTowardStar},
			{frame: base + 2*moveDur + 28, shot: "05-reward-pop"},
			{frame: base + 2*moveDur + 51, shot: "06-reward-fly"},
			{frame: base + 2*moveDur + 76, shot: "07-milestone"},
			{frame: base + 2*moveDur + 111, shot: "08-next"},
		}
	} else {
		g = game.New()
		g.SeedStickers(*seed)
		steps = []step{
			{frame: 30, shot: "01-home"},
			{frame: 90, shot: "02-home-later"},
		}
	}

	g.SetScaleOverride(*dpScale)
	sg := &shotGame{g: g, steps: steps, outDir: *out, w: *w, h: *h}
	ebiten.SetWindowTitle("shot")
	ebiten.SetWindowSize(*w, *h)
	if err := ebiten.RunGame(sg); err != nil && err != ebiten.Termination {
		panic(err)
	}
}
