package domain

import (
	"errors"
	"fmt"
)

// Kind classifies a domain error into the small set of outcomes the transport
// layer knows how to present. Services return *Error (or wrap one); handlers
// never pick HTTP status codes ad hoc.
type Kind int

const (
	KindInternal      Kind = iota // unclassified — presented as 500 without details
	KindNotFound                  // entity does not exist (404)
	KindUnauthorized              // no/invalid credentials (401)
	KindForbidden                 // authenticated but not allowed (403)
	KindConflict                  // valid request, wrong state (409) — e.g. illegal transition
	KindValidation                // malformed/missing input (400)
	KindUnprocessable             // well-formed but business rules refuse it (422)
)

// Error is the domain error type. Msg is safe to show to API clients.
type Error struct {
	Kind Kind
	Msg  string
	Err  error // optional wrapped cause (not exposed to clients)
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.Err)
	}
	return e.Msg
}

func (e *Error) Unwrap() error { return e.Err }

// Is makes two domain errors comparable by identity first (sentinels), and
// lets errors.Is(err, &Error{Kind: K}) match by kind when Msg is empty.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	if t.Msg == "" {
		return e.Kind == t.Kind
	}
	return e.Kind == t.Kind && e.Msg == t.Msg
}

func newErr(kind Kind, format string, args ...any) *Error {
	return &Error{Kind: kind, Msg: fmt.Sprintf(format, args...)}
}

func NotFoundf(format string, args ...any) *Error { return newErr(KindNotFound, format, args...) }
func Unauthorizedf(format string, args ...any) *Error {
	return newErr(KindUnauthorized, format, args...)
}
func Forbiddenf(format string, args ...any) *Error  { return newErr(KindForbidden, format, args...) }
func Conflictf(format string, args ...any) *Error   { return newErr(KindConflict, format, args...) }
func Validationf(format string, args ...any) *Error { return newErr(KindValidation, format, args...) }
func Unprocessablef(format string, args ...any) *Error {
	return newErr(KindUnprocessable, format, args...)
}

// Internalf wraps an unexpected cause; Msg stays generic for clients.
func Internalf(err error, format string, args ...any) *Error {
	return &Error{Kind: KindInternal, Msg: fmt.Sprintf(format, args...), Err: err}
}

// KindOf extracts the Kind from any error chain; unclassified errors are internal.
func KindOf(err error) Kind {
	var de *Error
	if errors.As(err, &de) {
		return de.Kind
	}
	return KindInternal
}

// MessageOf returns the client-safe message for classified errors, or a
// generic fallback for internal ones.
func MessageOf(err error) string {
	var de *Error
	if errors.As(err, &de) && de.Kind != KindInternal {
		return de.Msg
	}
	return "internal error"
}
