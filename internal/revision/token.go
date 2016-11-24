package revision

// token represents a entity extracted from string parsing
type token int

const (
	eof token = iota

	caret
	cbrace
	char
	colon
	control
	dot
	minus
	number
	obrace
	slash
	space
	tilde
	word
)
