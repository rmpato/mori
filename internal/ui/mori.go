package ui

// mori's face.
//
// tuki and mori are siblings, so they share a face and wear it differently:
// tuki's eyes are open and looking for the next thing, mori's are closed.
// That is the whole of the character difference, and it is enough — a second
// piece of artwork would be a second thing to look at, which is the opposite
// of what this is for.
//
// The face is deliberately rare. It appears when you arrive and when you
// leave, and nowhere in between. It does not react to what you write, it
// never sits at the top of an empty page looking expectant, and there is no
// expression for disappointment, because mori has nothing to be disappointed
// about.
const (
	// FaceCalm is mori at rest, which is nearly always.
	FaceCalm = "(-.-)"
	// FaceHere is mori looking up, for a hello.
	FaceHere = "(·.·)"
)

// The other mark mori makes is the season, which is in season.go: a leaf that
// changes with the month, so a year of pages quietly changes weather as you
// scroll back through it. That is the visual identity — a face at the edges,
// a leaf at the top, and otherwise your own writing.
