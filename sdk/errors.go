package sdk

import (
	"errors"
	"fmt"
	"strings"
)

// Error codes returned by the InfraKitchen API in errors[].extensions.code,
// plus a few client-side codes.
const (
	CodeNotFound         = "NOT_FOUND"
	CodeAccessDenied     = "ACCESS_DENIED"
	CodeUnauthorized     = "UNAUTHORIZED"
	CodeWrongState       = "WRONG_STATE"
	CodeDependency       = "DEPENDENCY"
	CodeConflict         = "CONFLICT"
	CodeChildrenNotReady = "CHILDREN_NOT_READY"
	CodeCannotProceed    = "CANNOT_PROCEED"
	CodeValidation       = "VALIDATION"
	CodeCredentialsError = "CREDENTIALS_ERROR"
	CodeNotImplemented   = "NOT_IMPLEMENTED"
	CodeInternal         = "INTERNAL"

	// CodeHTTP is used when the server answered with an unexpected HTTP status.
	CodeHTTP = "HTTP_ERROR"
)

// ErrNotConfigured is returned when no server URL is available.
var ErrNotConfigured = errors.New("no server configured: run 'ikctl login --server <url>' or set IKCTL_SERVER")

// APIError is an error reported by the InfraKitchen server.
type APIError struct {
	Code       string   `json:"code"`
	Message    string   `json:"message"`
	Path       []string `json:"path,omitempty"`
	HTTPStatus int      `json:"httpStatus,omitempty"`
}

func (e *APIError) Error() string {
	var b strings.Builder
	if e.Code != "" {
		b.WriteString(e.Code)
		b.WriteString(": ")
	}
	b.WriteString(e.Message)
	if len(e.Path) > 0 {
		fmt.Fprintf(&b, " (path: %s)", strings.Join(e.Path, "."))
	}
	return b.String()
}

// AmbiguousError is returned when a name lookup matches more than one object.
type AmbiguousError struct {
	Kind    string
	Name    string
	Matches []string // IDs of the matching objects
}

func (e *AmbiguousError) Error() string {
	return fmt.Sprintf("%s name %q is ambiguous: matches %d objects (%s); use an id instead",
		e.Kind, e.Name, len(e.Matches), strings.Join(e.Matches, ", "))
}

// ActionNotAllowedError is returned when an action is not currently allowed for an object.
type ActionNotAllowedError struct {
	Kind    string
	ID      string
	Action  string
	Allowed []string
}

func (e *ActionNotAllowedError) Error() string {
	allowed := "none"
	if len(e.Allowed) > 0 {
		allowed = strings.Join(e.Allowed, ", ")
	}
	return fmt.Sprintf("action %q is not allowed for %s %s; allowed actions: %s", e.Action, e.Kind, e.ID, allowed)
}

// UnsupportedError is returned when a kind does not support an operation.
type UnsupportedError struct {
	Kind string
	Verb string
}

func (e *UnsupportedError) Error() string {
	return fmt.Sprintf("kind %q does not support %q", e.Kind, e.Verb)
}

func notFound(kind, ref string) error {
	return &APIError{Code: CodeNotFound, Message: fmt.Sprintf("%s %q not found", kind, ref)}
}

// ErrorCode returns the API error code for err, or "" if err is not an API error.
func ErrorCode(err error) string {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code
	}
	return ""
}

// IsNotFound reports whether err means the requested object does not exist.
func IsNotFound(err error) bool { return ErrorCode(err) == CodeNotFound }

// IsUnauthorized reports whether err is an authentication or authorization failure.
func IsUnauthorized(err error) bool {
	if errors.Is(err, ErrNotConfigured) {
		return true
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.HTTPStatus == 401 || apiErr.HTTPStatus == 403 ||
			apiErr.Code == CodeUnauthorized || apiErr.Code == CodeAccessDenied
	}
	return false
}

// IsConflict reports whether err means the request conflicts with the object's
// current state (wrong state, validation, dependency, ambiguity, ...).
func IsConflict(err error) bool {
	var amb *AmbiguousError
	var notAllowed *ActionNotAllowedError
	if errors.As(err, &amb) || errors.As(err, &notAllowed) {
		return true
	}
	switch ErrorCode(err) {
	case CodeConflict, CodeWrongState, CodeValidation, CodeDependency, CodeChildrenNotReady, CodeCannotProceed:
		return true
	}
	return false
}
