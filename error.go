package ioc

import (
	"errors"
	"fmt"
)

var (
	ErrObjectNotFound          = errors.New("not found in container")
	ErrArgsNotInstanced        = errors.New("args not instanced")
	ErrInvalidReturnValueCount = errors.New("invalid return value count")
	ErrRepeatedBind            = errors.New("repeated bind")
	ErrInvalidArgs             = errors.New("invalid args")
)

//func isErrorType(t reflect.Type) bool {
//	return t.Implements(reflect.TypeOf((*error)(nil)).Elem())
//}

// buildObjectNotFoundError is an error object represent object not found
func buildObjectNotFoundError(msg string) error {
	return fmt.Errorf("%w: %s", ErrObjectNotFound, msg)
}

// buildArgNotInstancedError is an error object represent arg not instanced
func buildArgNotInstancedError(msg string) error {
	return fmt.Errorf("%w: %s", ErrArgsNotInstanced, msg)
}

// buildInvalidReturnValueCountError is an error object represent return values count not match
func buildInvalidReturnValueCountError(msg string) error {
	return fmt.Errorf("%w: %s", ErrInvalidReturnValueCount, msg)
}

// buildRepeatedBindError is an error object represent bind a value repeated
func buildRepeatedBindError(msg string) error {
	return fmt.Errorf("%w: %s", ErrRepeatedBind, msg)
}

// buildInvalidArgsError is an error object represent invalid args
func buildInvalidArgsError(msg string) error {
	return fmt.Errorf("%w: %s", ErrInvalidArgs, msg)
}
