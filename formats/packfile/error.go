package packfile

import "fmt"

// Error specifies errors returned during packfile parsing.
type Error struct {
	reason, details string
}

func NewError(reason string) *Error {
	return &Error{reason: reason}
}

func (e *Error) Error() string {
	if e.details == "" {
		return e.reason
	}

	return fmt.Sprintf("%s: %s", e.reason, e.details)
}

func (e *Error) AddDetails(format string, args ...interface{}) *Error {
	return &Error{
		reason:  e.reason,
		details: fmt.Sprintf(format, args...),
	}
}
