package revision

// token represents a entity extracted from string parsing
type token int

const (
	eof token = iota

	at
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
