package apperrors

import "fmt"

type Error struct {
	Code    string
	Status  int
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }

func New(code string, status int, message string, err error) *Error {
	return &Error{Code: code, Status: status, Message: message, Err: err}
}
