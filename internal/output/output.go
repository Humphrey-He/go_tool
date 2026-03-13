package output

import (
	"encoding/json"
	"fmt"
	"io"
	"go_tool/internal/report"
)

func WriteJSON(w io.Writer, diags []report.Diagnostic) error {
	payload := report.Report{Diagnostics: diags}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func WriteSummary(w io.Writer, diags []report.Diagnostic) {
	if len(diags) == 0 {
		_, _ = fmt.Fprintln(w, "No issues found.")
		return
	}

	counts := map[report.Severity]int{
		report.SeverityInfo:  0,
		report.SeverityWarn:  0,
		report.SeverityError: 0,
	}
	for _, d := range diags {
		counts[d.Severity]++
	}

	_, _ = fmt.Fprintf(w, "Issues: error=%d warn=%d info=%d\n", counts[report.SeverityError], counts[report.SeverityWarn], counts[report.SeverityInfo])
}
