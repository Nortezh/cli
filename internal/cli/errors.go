package cli

import (
	"errors"
	"fmt"

	"nortezh-cli/internal/api"
)

func formatCLIError(err error) string {
	if errors.Is(err, api.ErrUnauthenticated) {
		return "Error: not logged in. Run 'ntzh login'."
	}
	var apiErr *api.Error
	if errors.As(err, &apiErr) {
		return fmt.Sprintf("Error: %s: %s", apiErr.Code, apiErr.Message)
	}
	return fmt.Sprintf("Error: %s", err.Error())
}

// FormatCLIError is the exported entry point used by cmd/ntzh.
func FormatCLIError(err error) string { return formatCLIError(err) }
