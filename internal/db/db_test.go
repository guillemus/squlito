package db

import (
    "database/sql"
    "fmt"
    "testing"

    _ "modernc.org/sqlite"
)

func createTestDb(t *testing.T) *sql.DB {
    t.Helper()

    db, err := sql.Open("sqlite", "file:test.db?mode=memory&cache=shared")
    if err != nil {
        t.Fatalf("open db: %v", err)
    }

    _, err = db.Exec("PRAGMA foreign_keys = ON")
    if err != nil {
        t.Fatalf("pragma: %v", err)
    }

    return db
}

func TestParseDatabasePathFromArgs_Default(t *testing.T) {
    path := ParseDatabasePathFromArgs([]string{})
    if path != "data/seed.db" {
        t.Fatalf("expected default path, got %q", path)
    }
}

func TestParseDatabasePathFromArgs_Positional(t *testing.T) {
    path := ParseDatabasePathFromArgs([]string{"example.db"})
    if path != "example.db" {
        t.Fatalf("expected positional path, got %q", path)
    }
}

func TestParseDatabasePathFromArgs_SkipsFlags(t *testing.T) {
    path := ParseDatabasePathFromArgs([]string{"--verbose", "file.db"})
    if path != "file.db" {
        t.Fatalf("expected file.db, got %q", path)
    }
}

func TestParseDatabasePathFromArgs_IgnoresAfterFirst(t *testing.T) {
    path := ParseDatabasePathFromArgs([]string{"first.db", "second.db"})
    if path != "first.db" {
        t.Fatalf("expected first.db, got %q", path)
    }
}

func TestParseDatabasePathFromArgs_SkipsSeparator(t *testing.T) {
    path := ParseDatabasePathFromArgs([]string{"--", "db.sqlite"})
    if path != "db.sqlite" {
        t.Fatalf("expected db.sqlite, got %q", path)
    }
}

func TestListUserTables(t *testing.T) {
    db := createTestDb(t)
    defer db.Close()

    _, err := db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
    if err != nil {
        t.Fatalf("create table: %v", err)
    }

    _, err = db.Exec("CREATE TABLE posts (id INTEGER PRIMARY KEY, title TEXT)")
    if err != nil {
        t.Fatalf("create table: %v", err)
    }

    tables, err := ListUserTables(db)
    if err != nil {
        t.Fatalf("list tables: %v", err)
    }

    if len(tables) != 2 {
        t.Fatalf("expected 2 tables, got %d", len(tables))
    }

    if tables[0].Name != "posts" || tables[1].Name != "users" {
        t.Fatalf("unexpected table order: %v", tables)
    }
}

func TestGetTableColumns(t *testing.T) {
    db := createTestDb(t)
    defer db.Close()

    _, err := db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, active INTEGER)")
    if err != nil {
        t.Fatalf("create table: %v", err)
    }

    cols, err := GetTableColumns(db, "users")
    if err != nil {
        t.Fatalf("get columns: %v", err)
    }

    if len(cols) != 3 {
        t.Fatalf("expected 3 cols, got %d", len(cols))
    }

    if cols[0].Name != "id" || cols[1].Name != "name" || cols[2].Name != "active" {
        t.Fatalf("unexpected cols: %v", cols)
    }
}

func TestGetTablePage(t *testing.T) {
    db := createTestDb(t)
    defer db.Close()

    _, err := db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
    if err != nil {
        t.Fatalf("create table: %v", err)
    }

    stmt, err := db.Prepare("INSERT INTO users (id, name) VALUES (?, ?)")
    if err != nil {
        t.Fatalf("prepare: %v", err)
    }
    defer stmt.Close()

    for i := 1; i <= 10; i += 1 {
        name := fmt.Sprintf("User %d", i)
        _, err = stmt.Exec(i, name)
        if err != nil {
            t.Fatalf("insert: %v", err)
        }
    }

    page, err := GetTablePage(db, "users", 3, 4)
    if err != nil {
        t.Fatalf("get page: %v", err)
    }

    if page.TotalRows != 10 {
        t.Fatalf("expected total rows 10, got %d", page.TotalRows)
    }

    if page.Offset != 4 {
        t.Fatalf("expected offset 4, got %d", page.Offset)
    }

    if len(page.Rows) != 3 {
        t.Fatalf("expected 3 rows, got %d", len(page.Rows))
    }

    if page.Rows[0]["id"] != int64(5) {
        t.Fatalf("expected first row id 5, got %v", page.Rows[0]["id"])
    }
}

func TestGetTablePage_OffsetBeyondTotal(t *testing.T) {
    db := createTestDb(t)
    defer db.Close()

    _, err := db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
    if err != nil {
        t.Fatalf("create table: %v", err)
    }

    _, err = db.Exec("INSERT INTO users (id, name) VALUES (1, 'A')")
    if err != nil {
        t.Fatalf("insert: %v", err)
    }

    page, err := GetTablePage(db, "users", 4, 999)
    if err != nil {
        t.Fatalf("get page: %v", err)
    }

    if page.TotalRows != 1 {
        t.Fatalf("expected total rows 1, got %d", page.TotalRows)
    }

    if page.Offset != 999 {
        t.Fatalf("expected offset 999, got %d", page.Offset)
    }

    if len(page.Rows) != 0 {
        t.Fatalf("expected 0 rows, got %d", len(page.Rows))
    }
}
