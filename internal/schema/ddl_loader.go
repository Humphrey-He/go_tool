package schema

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
)

type DDLLoader struct {
	Path string
}

func (l *DDLLoader) Load(ctx context.Context) (Schema, error) {
	_ = ctx
	if l.Path == "" {
		return Schema{}, fmt.Errorf("ddl path is required")
	}

	f, err := os.Open(l.Path)
	if err != nil {
		return Schema{}, err
	}
	defer f.Close()

	builder := NewSchema()

	var currentTable string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		upper := strings.ToUpper(line)
		if strings.HasPrefix(upper, "CREATE TABLE") {
			name := extractTableName(line)
			if name != "" {
				currentTable = name
				builder.Tables[currentTable] = Table{Name: currentTable, Columns: map[string]Column{}, Indexes: map[string]Index{}}
			}
			continue
		}

		if currentTable == "" {
			continue
		}

		if strings.HasPrefix(upper, ")") || strings.HasPrefix(upper, "PRIMARY KEY") || strings.HasPrefix(upper, "UNIQUE") || strings.HasPrefix(upper, "KEY") || strings.HasPrefix(upper, "CONSTRAINT") {
			if strings.HasPrefix(upper, ")") {
				currentTable = ""
			}
			continue
		}

		col := extractColumnName(line)
		if col == "" {
			continue
		}

		table := builder.Tables[currentTable]
		table.Columns[col] = Column{Name: col}
		builder.Tables[currentTable] = table
	}

	if err := scanner.Err(); err != nil {
		return Schema{}, err
	}

	return builder, nil
}

func extractTableName(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(strings.ToUpper(line), "CREATE TABLE")
	line = strings.TrimSpace(line)
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return ""
	}
	name := strings.Trim(parts[0], "`\"")
	return name
}

func extractColumnName(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimSuffix(line, ",")
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return ""
	}
	name := strings.Trim(parts[0], "`\"")
	if name == "" || strings.HasPrefix(strings.ToUpper(name), "PRIMARY") || strings.HasPrefix(strings.ToUpper(name), "UNIQUE") || strings.HasPrefix(strings.ToUpper(name), "KEY") || strings.HasPrefix(strings.ToUpper(name), "CONSTRAINT") {
		return ""
	}
	return name
}
