package cli

import (
	"errors"
	"fmt"

	"github.com/nortezh/cli/internal/api"
)

// formatCLIError renders an error in AXI §6 structured form: a lowercase
// `error:` line, plus a `help:` line with the fix command when one is known.
func formatCLIError(err error) string {
	if errors.Is(err, api.ErrUnauthenticated) {
		return "error: not logged in\nhelp: run 'ntzh login'"
	}
	var apiErr *api.Error
	if errors.As(err, &apiErr) {
		return fmt.Sprintf("error: %s: %s", apiErr.Code, apiErr.Message)
	}
	return fmt.Sprintf("error: %s", err.Error())
}

// FormatCLIError is the exported entry point used by cmd/ntzh.
func FormatCLIError(err error) string { return formatCLIError(err) }
