package chess

type File int
type Rank int

type Square struct {
	File File
	Rank Rank
}

func Sq(f, r int) Square {
	return Square{File(f), Rank(r)}
}
