package report

type Severity string

const (
	SeverityInfo  Severity = "info"
	SeverityWarn  Severity = "warn"
	SeverityError Severity = "error"
)

type Diagnostic struct {
	File       string   `json:"file"`
	Line       int      `json:"line"`
	Column     int      `json:"column"`
	Snippet    string   `json:"snippet"`
	Rule       string   `json:"rule"`
	Message    string   `json:"message"`
	Suggestion string   `json:"suggestion"`
	Severity   Severity `json:"severity"`
	Code       string   `json:"code"`
}

type Report struct {
	Diagnostics []Diagnostic `json:"diagnostics"`
}
