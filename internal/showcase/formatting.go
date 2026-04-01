package showcase

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// SectionHeader prints a section header with underline.
func SectionHeader(w io.Writer, title string) {
	fmt.Fprintln(w, title)
	fmt.Fprintln(w, strings.Repeat("=", len(title)))
	fmt.Fprintln(w)
}

// SubHeader prints a sub-section header with dashes.
func SubHeader(w io.Writer, title string) {
	fmt.Fprintln(w, title)
	fmt.Fprintln(w, strings.Repeat("-", len(title)))
}

// NewTable creates a tabwriter for aligned table output.
func NewTable(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
}

// FormatNumber formats an integer with commas (e.g., 125000 -> "125,000").
func FormatNumber(n int64) string {
	if n < 0 {
		return "-" + FormatNumber(-n)
	}
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

// FormatSpeedup formats a speedup multiplier (e.g., 250.0 -> "250x").
func FormatSpeedup(factor float64) string {
	if factor >= 100 {
		return fmt.Sprintf("%.0fx", factor)
	}
	if factor >= 10 {
		return fmt.Sprintf("%.0fx", factor)
	}
	return fmt.Sprintf("%.1fx", factor)
}

// FormatSavings formats a percentage savings (e.g., 91.2 -> "91%").
func FormatSavings(pct float64) string {
	return fmt.Sprintf("%.0f%%", pct)
}

// StatusLine prints a status line for demo output.
func StatusLine(w io.Writer, status, message string) {
	fmt.Fprintf(w, "  [%s] %s\n", status, message)
}

// PrintDivider prints a divider line.
func PrintDivider(w io.Writer, width int) {
	fmt.Fprintln(w, strings.Repeat("-", width))
}
