package chess

import "sort"

type Move struct {
	From, To Square
	Capture  bool
}

type dir struct{ df, dr int }

var (
	orthoDirs = []dir{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
	diagDirs  = []dir{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}}
	knightDirs = []dir{
		{1, 2}, {2, 1}, {2, -1}, {1, -2},
		{-1, -2}, {-2, -1}, {-2, 1}, {-1, 2},
	}
)

func (b *Board) Moves(dst []Move, from Square) []Move {
	p := b.At(from)
	if p.IsEmpty() || !b.Contains(from) {
		return dst[:0]
	}
	switch p.Type {
	case Pawn:
		dst = b.pawnMoves(dst, from, p)
	case Knight:
		dst = b.stepMoves(dst, from, p, knightDirs)
	case Bishop:
		dst = b.slideMoves(dst, from, p, diagDirs)
	case Rook:
		dst = b.slideMoves(dst, from, p, orthoDirs)
	case Queen:
		dst = b.slideMoves(dst, from, p, append(orthoDirs, diagDirs...))
	case King:
		dst = b.stepMoves(dst, from, p, append(orthoDirs, diagDirs...))
	}
	sortMoves(dst)
	return dst
}

func (b *Board) MoveTargets(from Square) []Square {
	moves := b.Moves(nil, from)
	out := make([]Square, len(moves))
	for i, m := range moves {
		out[i] = m.To
	}
	return out
}

func (b *Board) CanMove(from, to Square) bool {
	for _, m := range b.Moves(nil, from) {
		if m.To == to {
			return true
		}
	}
	return false
}

func (b *Board) pawnMoves(dst []Move, from Square, p Piece) []Move {
	fwd := 1
	if p.Color == Black {
		fwd = -1
	}
	homeRank := 1
	if p.Color == Black {
		homeRank = b.height - 2
	}

	next := Sq(int(from.File), int(from.Rank)+fwd)
	if b.Contains(next) && b.At(next).IsEmpty() {
		dst = append(dst, Move{From: from, To: next})
		double := Sq(int(from.File), int(from.Rank)+2*fwd)
		if int(from.Rank) == homeRank && b.Contains(double) && b.At(double).IsEmpty() {
			dst = append(dst, Move{From: from, To: double})
		}
	}

	for _, d := range []dir{{1, fwd}, {-1, fwd}} {
		cap := Sq(int(from.File)+d.df, int(from.Rank)+d.dr)
		if !b.Contains(cap) {
			continue
		}
		target := b.At(cap)
		if !target.IsEmpty() && target.Color != p.Color {
			dst = append(dst, Move{From: from, To: cap, Capture: true})
		}
	}
	return dst
}

func (b *Board) stepMoves(dst []Move, from Square, p Piece, dirs []dir) []Move {
	for _, d := range dirs {
		to := Sq(int(from.File)+d.df, int(from.Rank)+d.dr)
		if !b.Contains(to) {
			continue
		}
		target := b.At(to)
		if target.IsEmpty() {
			dst = append(dst, Move{From: from, To: to})
		} else if target.Color != p.Color {
			dst = append(dst, Move{From: from, To: to, Capture: true})
		}
	}
	return dst
}

func (b *Board) slideMoves(dst []Move, from Square, p Piece, dirs []dir) []Move {
	for _, d := range dirs {
		f, r := int(from.File)+d.df, int(from.Rank)+d.dr
		for {
			to := Sq(f, r)
			if !b.Contains(to) {
				break
			}
			target := b.At(to)
			if target.IsEmpty() {
				dst = append(dst, Move{From: from, To: to})
			} else {
				if target.Color != p.Color {
					dst = append(dst, Move{From: from, To: to, Capture: true})
				}
				break
			}
			f += d.df
			r += d.dr
		}
	}
	return dst
}

func sortMoves(moves []Move) {
	sort.Slice(moves, func(i, j int) bool {
		if moves[i].To.Rank != moves[j].To.Rank {
			return moves[i].To.Rank < moves[j].To.Rank
		}
		if moves[i].To.File != moves[j].To.File {
			return moves[i].To.File < moves[j].To.File
		}
		if moves[i].From.Rank != moves[j].From.Rank {
			return moves[i].From.Rank < moves[j].From.Rank
		}
		return moves[i].From.File < moves[j].From.File
	})
}
