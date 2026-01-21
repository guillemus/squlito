package db

import (
    "database/sql"
    "fmt"
    "net/url"
    "strings"
)

type SqliteValue any

type SqliteRow map[string]SqliteValue

type SqliteTable struct {
    Name string
}

type SqliteColumn struct {
    Cid          int
    Name         string
    Type         string
    NotNull      int
    DefaultValue sql.NullString
    PrimaryKey   int
}

type TablePage struct {
    TotalRows int
    Offset    int
    Rows      []SqliteRow
}

type QueryRowsResult struct {
    Columns   []string
    Rows      []SqliteRow
    Truncated bool
}

func ParseDatabasePathFromArgs(args []string) string {
    path := "data/seed.db"

    for _, arg := range args {
        if arg == "--" {
            continue
        }

        if strings.HasPrefix(arg, "-") {
            continue
        }

        path = arg
        break
    }

    return path
}

func OpenDatabase(dbPath string) (*sql.DB, error) {
    dsn := makeReadonlyDsn(dbPath)
    db, err := sql.Open("sqlite", dsn)
    if err != nil {
        return nil, err
    }

    _, err = db.Exec("PRAGMA foreign_keys = ON")
    if err != nil {
        closeErr := db.Close()
        if closeErr != nil {
            return nil, fmt.Errorf("open db: %w; close error: %v", err, closeErr)
        }
        return nil, err
    }

    return db, nil
}

func ListUserTables(db *sql.DB) ([]SqliteTable, error) {
    sqlText := "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name"
    rows, err := db.Query(sqlText)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    tables := []SqliteTable{}
    for rows.Next() {
        var name string
        err = rows.Scan(&name)
        if err != nil {
            return nil, err
        }

        tables = append(tables, SqliteTable{Name: name})
    }

    err = rows.Err()
    if err != nil {
        return nil, err
    }

    return tables, nil
}

func GetTableColumns(db *sql.DB, tableName string) ([]SqliteColumn, error) {
    sqlText := fmt.Sprintf("PRAGMA table_info(%s)", quoteIdentifier(tableName))
    rows, err := db.Query(sqlText)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    columns := []SqliteColumn{}
    for rows.Next() {
        column := SqliteColumn{}
        err = rows.Scan(
            &column.Cid,
            &column.Name,
            &column.Type,
            &column.NotNull,
            &column.DefaultValue,
            &column.PrimaryKey,
        )
        if err != nil {
            return nil, err
        }

        columns = append(columns, column)
    }

    err = rows.Err()
    if err != nil {
        return nil, err
    }

    return columns, nil
}

func GetTablePage(db *sql.DB, tableName string, limit int, offset int) (TablePage, error) {
    safeLimit := clampInt(limit, 1, 500)
    safeOffset := offset
    if safeOffset < 0 {
        safeOffset = 0
    }

    countSql := fmt.Sprintf("SELECT COUNT(*) AS count FROM %s", quoteIdentifier(tableName))
    countRow := db.QueryRow(countSql)
    totalRows := 0
    err := countRow.Scan(&totalRows)
    if err != nil {
        return TablePage{}, err
    }

    pageSql := fmt.Sprintf("SELECT * FROM %s LIMIT ? OFFSET ?", quoteIdentifier(tableName))
    result, err := QueryRows(db, pageSql, 0, safeLimit, safeOffset)
    if err != nil {
        return TablePage{}, err
    }

    page := TablePage{
        TotalRows: totalRows,
        Offset:    safeOffset,
        Rows:      result.Rows,
    }

    return page, nil
}

func QueryRows(db *sql.DB, sqlText string, limit int, args ...any) (QueryRowsResult, error) {
    rows, err := db.Query(sqlText, args...)
    if err != nil {
        return QueryRowsResult{}, err
    }
    defer rows.Close()

    return scanRows(rows, limit)
}

func scanRows(rows *sql.Rows, limit int) (QueryRowsResult, error) {
    columns, err := rows.Columns()
    if err != nil {
        return QueryRowsResult{}, err
    }

    values := make([]any, len(columns))
    valuePtrs := make([]any, len(columns))
    for i := range valuePtrs {
        valuePtrs[i] = &values[i]
    }

    resultRows := []SqliteRow{}
    truncated := false

    for rows.Next() {
        if limit > 0 && len(resultRows) >= limit {
            truncated = true
            break
        }

        err = rows.Scan(valuePtrs...)
        if err != nil {
            return QueryRowsResult{}, err
        }

        row := make(SqliteRow, len(columns))
        for i, col := range columns {
            row[col] = normalizeValue(values[i])
        }

        resultRows = append(resultRows, row)
    }

    err = rows.Err()
    if err != nil {
        return QueryRowsResult{}, err
    }

    result := QueryRowsResult{
        Columns:   columns,
        Rows:      resultRows,
        Truncated: truncated,
    }

    return result, nil
}

func normalizeValue(value any) SqliteValue {
    if value == nil {
        return nil
    }

    switch typed := value.(type) {
    case []byte:
        copyValue := make([]byte, len(typed))
        copy(copyValue, typed)
        return copyValue
    case string:
        return typed
    case int64:
        return typed
    case float64:
        return typed
    case bool:
        return typed
    default:
        return fmt.Sprint(value)
    }
}

func makeReadonlyDsn(dbPath string) string {
    if strings.HasPrefix(dbPath, "file:") {
        if strings.Contains(dbPath, "mode=") {
            return dbPath
        }

        separator := "?"
        if strings.Contains(dbPath, "?") {
            separator = "&"
        }

        return dbPath + separator + "mode=ro"
    }

    escaped := url.PathEscape(dbPath)
    return "file:" + escaped + "?mode=ro"
}

func quoteIdentifier(identifier string) string {
    escaped := strings.ReplaceAll(identifier, "\"", "\"\"")
    return "\"" + escaped + "\""
}

func clampInt(value int, min int, max int) int {
    if value < min {
        return min
    }

    if value > max {
        return max
    }

    return value
}
