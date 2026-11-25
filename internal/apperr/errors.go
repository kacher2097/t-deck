package apperr

import (
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
)

// Code classifies domain and infrastructure errors returned across the Wails boundary.
type Code string

const (
	CodeValidation Code = "VALIDATION"
	CodeNotFound   Code = "NOT_FOUND"
	CodeNotReady   Code = "NOT_READY"
	CodeConflict   Code = "CONFLICT"
	CodeCloud      Code = "CLOUD"
	CodeIO         Code = "IO"
	CodeInternal   Code = "INTERNAL"
)

// AppError carries a machine-readable code, a safe user-facing message, and an optional wrapped cause.
type AppError struct {
	Code    Code
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return fmt.Sprintf("[%s] %s", e.Code, e.Message)
	}
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s", e.Code, e.Cause.Error())
	}
	return string(e.Code)
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// Wrap constructs a typed AppError preserving err via errors.Is / errors.As.
func Wrap(code Code, message string, err error) error {
	return &AppError{
		Code:    code,
		Message: strings.TrimSpace(message),
		Cause:   err,
	}
}

func Validation(msg string) error { return Wrap(CodeValidation, msg, nil) }
func NotFound(msg string) error   { return Wrap(CodeNotFound, msg, nil) }
func NotReady(msg string) error   { return Wrap(CodeNotReady, msg, nil) }
func Conflict(msg string) error   { return Wrap(CodeConflict, msg, nil) }
func Cloud(msg string, err error) error {
	return Wrap(CodeCloud, msg, err)
}
func IO(msg string, err error) error {
	return Wrap(CodeIO, msg, err)
}

// Recover converts a panic into a typed internal AppError so background workers and Wails handlers never crash the process.
func Recover(scope string, errPtr *error, onPanic func(scope string, r any, stack string)) {
	if r := recover(); r != nil {
		stack := string(debug.Stack())
		if onPanic != nil {
			onPanic(scope, r, stack)
		}
		if errPtr != nil {
			*errPtr = &AppError{
				Code:    CodeInternal,
				Message: fmt.Sprintf("Lỗi hệ thống tại %s. Vui lòng thử lại.", scope),
				Cause:   fmt.Errorf("panic: %v", r),
			}
		}
	}
}

// IsCode reports whether err or any wrapped error matches code.
func IsCode(err error, code Code) bool {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae.Code == code
	}
	return false
}
