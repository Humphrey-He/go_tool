package output

import (
	"encoding/json"
	"io"
	"path/filepath"

	"go_tool/internal/report"
)

type sarifReport struct {
	Version string       `json:"version"`
	Schema  string       `json:"$schema"`
	Runs    []sarifRun   `json:"runs"`
}

type sarifRun struct {
	Tool       sarifTool      `json:"tool"`
	Results    []sarifResult  `json:"results"`
	Properties sarifProps     `json:"properties,omitempty"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string       `json:"name"`
	InformationURI string       `json:"informationUri,omitempty"`
	Rules          []sarifRule   `json:"rules"`
}

type sarifRule struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	ShortDescription sarifText `json:"shortDescription"`
}

type sarifResult struct {
	RuleID   string       `json:"ruleId"`
	Level    string       `json:"level"`
	Message  sarifText    `json:"message"`
	Locations []sarifLocation `json:"locations"`
	Properties sarifProps `json:"properties,omitempty"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           sarifRegion           `json:"region"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn"`
}

type sarifText struct {
	Text string `json:"text"`
}

type sarifProps map[string]string

func WriteSARIF(w io.Writer, diags []report.Diagnostic) error {
	rules := map[string]sarifRule{}
	results := make([]sarifResult, 0, len(diags))
	for _, d := range diags {
		rule := sarifRule{ID: d.Code, Name: d.Rule, ShortDescription: sarifText{Text: d.Message}}
		rules[d.Code] = rule
		results = append(results, sarifResult{
			RuleID:  d.Code,
			Level:   toSarifLevel(d.Severity),
			Message: sarifText{Text: d.Message},
			Locations: []sarifLocation{{
				PhysicalLocation: sarifPhysicalLocation{
					ArtifactLocation: sarifArtifactLocation{URI: filepath.ToSlash(d.File)},
					Region: sarifRegion{StartLine: d.Line, StartColumn: d.Column},
				},
			}},
			Properties: sarifProps{
				"suggestion": d.Suggestion,
				"confidence": string(d.Confidence),
				"snippet": d.Snippet,
			},
		})
	}

	list := make([]sarifRule, 0, len(rules))
	for _, r := range rules {
		list = append(list, r)
	}

	report := sarifReport{
		Version: "2.1.0",
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Runs: []sarifRun{{
			Tool: sarifTool{Driver: sarifDriver{Name: "sqlsafelint", Rules: list}},
			Results: results,
		}},
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func toSarifLevel(sev report.Severity) string {
	switch sev {
	case report.SeverityError:
		return "error"
	case report.SeverityWarn:
		return "warning"
	default:
		return "note"
	}
}
