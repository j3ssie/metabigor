package output

import (
	"fmt"
	"slices"
	"strings"
)

// Format identifies how results are serialized to stdout and to disk.
type Format string

const (
	// FormatText renders human-readable lines. This is the default.
	FormatText Format = "text"
	// FormatJSON renders one JSON object per line.
	FormatJSON Format = "json"
	// FormatCSV renders comma-separated rows behind a single header row.
	FormatCSV Format = "csv"
	// FormatFlat renders the bare primary value, one per line, for piping.
	FormatFlat Format = "flat"
)

// Formats lists every supported format, in the order help text shows them.
// Adding a format here is enough for ParseFormat and the help text to pick it up.
var Formats = []Format{FormatText, FormatJSON, FormatCSV, FormatFlat}

// FormatNames returns the supported format names for help text and errors.
func FormatNames() []string {
	names := make([]string, len(Formats))
	for i, f := range Formats {
		names[i] = string(f)
	}
	return names
}

// ParseFormat validates a user-supplied format name.
func ParseFormat(s string) (Format, error) {
	f := Format(strings.ToLower(strings.TrimSpace(s)))
	if slices.Contains(Formats, f) {
		return f, nil
	}
	return "", fmt.Errorf("unsupported --format %q (pick one of: %s)", s, strings.Join(FormatNames(), ", "))
}

// Columns joins the non-empty values into one " | " separated line. Every
// command's text output has this shape, so the empty-field handling lives here
// rather than in each Record implementation.
func Columns(values ...string) string {
	present := make([]string, 0, len(values))
	for _, v := range values {
		if v != "" {
			present = append(present, v)
		}
	}
	return strings.Join(present, " | ")
}

// Record is a single result that knows how to render itself in every format.
// Implementations are plain structs so FormatJSON can marshal them directly,
// which keeps the JSON shape identical to the struct's field tags.
type Record interface {
	// Text returns human-readable lines. Returning a trailing empty string
	// separates this record from the next with a blank line.
	Text() []string
	// Flat returns the bare primary values, for piping into other tools.
	Flat() []string
	// CSV returns the column header and this record's rows. The writer emits
	// the header once, and again only if a later record's header differs.
	CSV() (header []string, rows [][]string)
}
