package hclutil

import (
	"fmt"
	"log/slog"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// ParseFile parses raw HCL bytes into an hclwrite.File.
// Returns a user-friendly error if the HCL is malformed.
// Non-error diagnostics (warnings) are logged via slog.
func ParseFile(src []byte) (*hclwrite.File, error) {
	f, diags := hclwrite.ParseConfig(src, "generated.tf", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL: %s", diags.Error())
	}
	// Log non-error diagnostics (warnings)
	for _, d := range diags {
		if d.Severity != hcl.DiagError {
			slog.Warn("HCL diagnostic", "summary", d.Summary, "detail", d.Detail)
		}
	}
	return f, nil
}

// FormatFile formats an hclwrite.File to canonical HCL.
func FormatFile(f *hclwrite.File) []byte {
	return hclwrite.Format(f.Bytes())
}
