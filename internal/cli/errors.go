package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/rustyrien/ikctl/internal/printers"
	"github.com/rustyrien/ikctl/sdk"
)

// Exit codes.
const (
	ExitOK       = 0
	ExitError    = 1
	ExitUsage    = 2
	ExitAuth     = 3
	ExitNotFound = 4
	ExitConflict = 5
)

// usageError marks errors caused by invalid command-line usage.
type usageError struct{ error }

func (e usageError) Unwrap() error { return e.error }

func usageErrorf(format string, args ...any) error { return usageError{fmt.Errorf(format, args...)} }

// usageArgs wraps a positional-argument validator so its errors are usage errors.
func usageArgs(fn cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := fn(cmd, args); err != nil {
			return usageError{err}
		}
		return nil
	}
}

func exitCode(err error) int {
	var ue usageError
	switch {
	case err == nil:
		return ExitOK
	case errors.As(err, &ue):
		return ExitUsage
	case sdk.IsUnauthorized(err):
		return ExitAuth
	case sdk.IsNotFound(err):
		return ExitNotFound
	case sdk.IsConflict(err):
		return ExitConflict
	default:
		return ExitError
	}
}

func errorCode(err error) string {
	var ue usageError
	var amb *sdk.AmbiguousError
	var na *sdk.ActionNotAllowedError
	switch {
	case errors.As(err, &ue):
		return "USAGE"
	case errors.As(err, &amb):
		return "AMBIGUOUS"
	case errors.As(err, &na):
		return "ACTION_NOT_ALLOWED"
	case errors.Is(err, sdk.ErrNotConfigured):
		return "NOT_CONFIGURED"
	case errors.Is(err, context.Canceled):
		return "CANCELED"
	case errors.Is(err, context.DeadlineExceeded):
		return "TIMEOUT"
	}
	if code := sdk.ErrorCode(err); code != "" {
		return code
	}
	return "ERROR"
}

// reportError prints err to w. When the command was asked for -o json or
// -o yaml, the error is printed as a structured document so that automation
// can parse it.
func reportError(w io.Writer, cmd *cobra.Command, err error) {
	if cmd != nil {
		if fl := cmd.Flags().Lookup("output"); fl != nil {
			if out, perr := printers.ParseOutput(fl.Value.String()); perr == nil && out.IsStructured() {
				msg := err.Error()
				if apiErr, ok := err.(*sdk.APIError); ok { //nolint:errorlint // only an unwrapped error's text is "CODE: message"
					msg = apiErr.Message
				}
				doc := map[string]any{"error": map[string]any{"code": errorCode(err), "message": msg, "exitCode": exitCode(err)}}
				_ = printers.PrintStructured(w, out.Format, doc)
				return
			}
		}
	}
	fmt.Fprintf(w, "Error: %v\n", err)
	var ue usageError
	if errors.As(err, &ue) && cmd != nil {
		fmt.Fprintf(w, "Run '%s --help' for usage.\n", cmd.CommandPath())
	}
}

func warnf(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, "Warning: "+format+"\n", args...)
}
