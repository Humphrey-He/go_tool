package schema

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

type PostgresLoader struct {
	DSN        string
	SearchPath []string
}

func (l *PostgresLoader) Load(ctx context.Context) (Schema, error) {
	if l.DSN == "" {
		return Schema{}, fmt.Errorf("postgres dsn is required")
	}

	db, err := sql.Open("postgres", l.DSN)
	if err != nil {
		return Schema{}, err
	}
	defer db.Close()

	searchPath := l.SearchPath
	if len(searchPath) == 0 {
		searchPath = []string{"public"}
	}
	_, err = db.ExecContext(ctx, "SET search_path TO "+strings.Join(searchPath, ", "))
	if err != nil {
		return Schema{}, err
	}

	rows, err := db.QueryContext(ctx, `
		SELECT table_schema, table_name, column_name, data_type
		FROM information_schema.columns
		WHERE table_schema = ANY($1)
		ORDER BY table_schema, table_name, ordinal_position
	`, pqArray(searchPath))
	if err != nil {
		return Schema{}, err
	}
	defer rows.Close()

	builder := NewSchema()
	for rows.Next() {
		var schemaName, tableName, columnName, dataType string
		if err := rows.Scan(&schemaName, &tableName, &columnName, &dataType); err != nil {
			return Schema{}, err
		}
		fullTable := fmt.Sprintf("%s.%s", schemaName, tableName)
		table, ok := builder.Tables[fullTable]
		if !ok {
			table = Table{Name: fullTable, Columns: map[string]Column{}, Indexes: map[string]Index{}}
		}
		table.Columns[columnName] = Column{Name: columnName, Type: dataType}
		builder.Tables[fullTable] = table
	}
	if err := rows.Err(); err != nil {
		return Schema{}, err
	}

	idxRows, err := db.QueryContext(ctx, `
		SELECT schemaname, tablename, indexname, indexdef
		FROM pg_indexes
		WHERE schemaname = ANY($1)
	`, pqArray(searchPath))
	if err != nil {
		return Schema{}, err
	}
	defer idxRows.Close()

	for idxRows.Next() {
		var schemaName, tableName, indexName, indexDef string
		if err := idxRows.Scan(&schemaName, &tableName, &indexName, &indexDef); err != nil {
			return Schema{}, err
		}
		fullTable := fmt.Sprintf("%s.%s", schemaName, tableName)
		table, ok := builder.Tables[fullTable]
		if !ok {
			table = Table{Name: fullTable, Columns: map[string]Column{}, Indexes: map[string]Index{}}
		}
		idx := parseIndexDef(indexDef)
		idx.Name = indexName
		table.Indexes[indexName] = idx
		builder.Tables[fullTable] = table
	}
	if err := idxRows.Err(); err != nil {
		return Schema{}, err
	}

	return builder, nil
}

func pqArray(items []string) interface{} {
	return fmt.Sprintf("{%s}", strings.Join(items, ","))
}

func parseIndexDef(def string) Index {
	upper := strings.ToUpper(def)
	method := "BTREE"
	if strings.Contains(upper, "USING GIN") {
		method = "GIN"
	} else if strings.Contains(upper, "USING HASH") {
		method = "HASH"
	}
	cols := parseIndexColumns(def)
	return Index{Columns: cols, Method: method}
}

func parseIndexColumns(def string) []string {
	open := strings.Index(def, "(")
	close := strings.LastIndex(def, ")")
	if open == -1 || close == -1 || close <= open {
		return nil
	}
	inside := def[open+1 : close]
	parts := strings.Split(inside, ",")
	cols := make([]string, 0, len(parts))
	for _, p := range parts {
		c := strings.TrimSpace(p)
		if c == "" {
			continue
		}
		cols = append(cols, strings.Trim(c, "\"`"))
	}
	return cols
}
