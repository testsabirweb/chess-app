package chess

type Color uint8

const (
	White Color = iota
	Black
)

type PieceType uint8

const (
	NoPiece PieceType = iota
	Pawn
	Knight
	Bishop
	Rook
	Queen
	King
)

type Piece struct {
	Type  PieceType
	Color Color
}

func (p Piece) IsEmpty() bool {
	return p.Type == NoPiece
}

func (c Color) Opponent() Color {
	if c == White {
		return Black
	}
	return White
}

func (p Piece) Opponent() Color {
	return p.Color.Opponent()
}
