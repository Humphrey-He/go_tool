package schema

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLLoader struct {
	DSN string
}

func (l *MySQLLoader) Load(ctx context.Context) (Schema, error) {
	if l.DSN == "" {
		return Schema{}, fmt.Errorf("mysql dsn is required")
	}
	db, err := sql.Open("mysql", l.DSN)
	if err != nil {
		return Schema{}, err
	}
	defer db.Close()

	builder := NewSchema()

	rows, err := db.QueryContext(ctx, `
		SELECT table_schema, table_name, column_name, data_type
		FROM information_schema.columns
		WHERE table_schema = DATABASE()
		ORDER BY table_name, ordinal_position
	`)
	if err != nil {
		return Schema{}, err
	}
	defer rows.Close()

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
		SELECT table_schema, table_name, index_name, column_name, non_unique
		FROM information_schema.statistics
		WHERE table_schema = DATABASE()
		ORDER BY table_name, index_name, seq_in_index
	`)
	if err != nil {
		return Schema{}, err
	}
	defer idxRows.Close()

	for idxRows.Next() {
		var schemaName, tableName, indexName, columnName string
		var nonUnique int
		if err := idxRows.Scan(&schemaName, &tableName, &indexName, &columnName, &nonUnique); err != nil {
			return Schema{}, err
		}
		fullTable := fmt.Sprintf("%s.%s", schemaName, tableName)
		table, ok := builder.Tables[fullTable]
		if !ok {
			table = Table{Name: fullTable, Columns: map[string]Column{}, Indexes: map[string]Index{}}
		}
		idx := table.Indexes[indexName]
		idx.Name = indexName
		idx.Unique = nonUnique == 0
		idx.Method = "BTREE"
		idx.Columns = append(idx.Columns, columnName)
		table.Indexes[indexName] = idx
		builder.Tables[fullTable] = table
	}
	if err := idxRows.Err(); err != nil {
		return Schema{}, err
	}

	return builder, nil
}
