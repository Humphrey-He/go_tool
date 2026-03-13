package baseline

import (
	"encoding/json"
	"os"
	"path/filepath"

	"go_tool/internal/report"
)

type Entry struct {
	Code   string `json:"code"`
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

type File struct {
	Entries []Entry `json:"entries"`
}

func Load(path string) (File, error) {
	var f File
	if path == "" {
		return f, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return f, err
	}
	if err := json.Unmarshal(data, &f); err != nil {
		return f, err
	}
	return f, nil
}

func (f File) Ignore(diag report.Diagnostic) bool {
	for _, e := range f.Entries {
		if e.Code != "" && e.Code != diag.Code {
			continue
		}
		if e.File != "" {
			if filepath.Clean(e.File) != filepath.Clean(diag.File) {
				continue
			}
		}
		if e.Line != 0 && e.Line != diag.Line {
			continue
		}
		if e.Column != 0 && e.Column != diag.Column {
			continue
		}
		return true
	}
	return false
}

func Filter(diags []report.Diagnostic, base File) []report.Diagnostic {
	if len(base.Entries) == 0 {
		return diags
	}
	out := make([]report.Diagnostic, 0, len(diags))
	for _, d := range diags {
		if base.Ignore(d) {
			continue
		}
		out = append(out, d)
	}
	return out
}
