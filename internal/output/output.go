package output

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/atoolz/terraxi/pkg/types"
)

// Format is the output format for discovery results.
type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
)

// Writer writes discovery results in the specified format.
type Writer struct {
	format Format
	w      io.Writer
}

// NewWriter creates a new output writer.
func NewWriter(w io.Writer, format Format) *Writer {
	return &Writer{format: format, w: w}
}

// WriteResult outputs a discovery result.
func (ow *Writer) WriteResult(result *types.DiscoveryResult) error {
	switch ow.format {
	case FormatJSON:
		return ow.writeJSON(result)
	case FormatTable:
		return ow.writeTable(result)
	default:
		return ow.writeTable(result)
	}
}

func (ow *Writer) writeJSON(result *types.DiscoveryResult) error {
	enc := json.NewEncoder(ow.w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func (ow *Writer) writeTable(result *types.DiscoveryResult) error {
	tw := tabwriter.NewWriter(ow.w, 0, 0, 2, ' ', 0)

	_, _ = fmt.Fprintf(tw, "TYPE\tID\tNAME\tREGION\n")
	_, _ = fmt.Fprintf(tw, "----\t--\t----\t------\n")

	for _, r := range result.Resources {
		name := r.Name
		if name == "" {
			name = "-"
		}
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", r.Type, r.ID, name, r.Region)
	}

	_, _ = fmt.Fprintf(tw, "\nTotal: %d resources discovered\n", len(result.Resources))

	if len(result.Errors) > 0 {
		_, _ = fmt.Fprintf(tw, "\nErrors (%d):\n", len(result.Errors))
		for _, e := range result.Errors {
			_, _ = fmt.Fprintf(tw, "  - %s\n", e)
		}
	}

	return tw.Flush()
}
