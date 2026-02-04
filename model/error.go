package model

import "fmt"

type Error struct {
	error error
	code  int
}

func (e Error) Error() string {
	return e.error.Error()
}

func (e Error) Code() int {
	return e.code
}

func NewError(err string, code int) Error {
	return Error{error: fmt.Errorf(err), code: code}
}

var UnknownError = NewError("something went wrong", 500)
