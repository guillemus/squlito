package app

import (
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"

	"github.com/awesome-gocui/gocui"
)

const historyTableName = "query_history"

func (app *App) initHistory() {
	dbConn, err := openHistoryDB()
	if err != nil {
		return
	}

	err = ensureHistorySchema(dbConn)
	if err != nil {
		_ = dbConn.Close()
		return
	}

	entries, err := loadQueryHistory(dbConn, historyLimit)
	if err != nil {
		entries = nil
	}

	app.historyDB = dbConn
	app.historyEntries = entries
	app.historyIndex = -1
	app.historyDraft = ""
}

func (app *App) recordHistory(sqlText string) {
	if app.historyDB == nil {
		return
	}

	entry, err := insertQueryHistory(app.historyDB, sqlText)
	if err != nil {
		return
	}

	app.historyEntries = append([]QueryHistoryEntry{entry}, app.historyEntries...)
	if len(app.historyEntries) > historyLimit {
		app.historyEntries = app.historyEntries[:historyLimit]
	}
}

func (app *App) resetHistorySelection() {
	if app.historyIndex == -1 && app.historyDraft == "" {
		return
	}

	app.historyIndex = -1
	app.historyDraft = ""
}

func (app *App) moveHistorySelection(view *gocui.View, delta int) error {
	if view == nil {
		return nil
	}

	if len(app.historyEntries) == 0 {
		return nil
	}
	if delta == 0 {
		return nil
	}
	if delta < 0 && app.historyIndex == -1 {
		return nil
	}

	if app.historyIndex == -1 {
		app.historyDraft = view.Buffer()
	}

	var next int
	if delta > 0 {
		if app.historyIndex == -1 {
			next = 0
		} else {
			next = app.historyIndex + 1
		}
		if next > len(app.historyEntries)-1 {
			next = len(app.historyEntries) - 1
		}
	} else {
		next = app.historyIndex - 1
		if next < 0 {
			next = -1
		}
	}
	if next == app.historyIndex {
		return nil
	}

	app.historyIndex = next
	if app.historyIndex == -1 {
		app.setQueryViewContent(view, app.historyDraft)
		return app.render()
	}

	app.setQueryViewContent(view, app.historyEntries[app.historyIndex].SQL)
	return app.render()
}

func (app *App) setQueryViewContent(view *gocui.View, value string) {
	if view == nil {
		return
	}

	view.Clear()
	view.WriteString(value)
	lines := view.BufferLines()
	if len(lines) == 0 {
		_ = view.SetCursor(0, 0)
		return
	}

	lastIndex := len(lines) - 1
	lastLine := lines[lastIndex]
	lastColumn := utf8.RuneCountInString(lastLine)
	_ = view.SetCursor(0, 0)
	view.MoveCursor(lastColumn, lastIndex)
}

func openHistoryDB() (*sql.DB, error) {
	path, err := historyDBPath()
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		return nil, err
	}

	dsn := makeHistoryDsn(path)
	dbConn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	_, err = dbConn.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		closeErr := dbConn.Close()
		if closeErr != nil {
			return nil, closeErr
		}
		return nil, err
	}

	return dbConn, nil
}

func historyDBPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "squlito", "history.db"), nil
}

func makeHistoryDsn(path string) string {
	escaped := url.PathEscape(path)
	return "file:" + escaped + "?mode=rwc"
}

func ensureHistorySchema(dbConn *sql.DB) error {
	createTable := "CREATE TABLE IF NOT EXISTS " + historyTableName + " (id INTEGER PRIMARY KEY AUTOINCREMENT, sql TEXT NOT NULL, created_at TEXT NOT NULL)"
	_, err := dbConn.Exec(createTable)
	if err != nil {
		return err
	}

	createIndex := "CREATE INDEX IF NOT EXISTS query_history_created_at ON " + historyTableName + " (created_at DESC)"
	_, err = dbConn.Exec(createIndex)
	return err
}

func loadQueryHistory(dbConn *sql.DB, limit int) ([]QueryHistoryEntry, error) {
	if limit <= 0 {
		return []QueryHistoryEntry{}, nil
	}

	rows, err := dbConn.Query("SELECT id, sql, created_at FROM "+historyTableName+" ORDER BY created_at DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	entries := []QueryHistoryEntry{}
	for rows.Next() {
		var entry QueryHistoryEntry
		err = rows.Scan(&entry.ID, &entry.SQL, &entry.CreatedAt)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return entries, nil
}

func insertQueryHistory(dbConn *sql.DB, sqlText string) (QueryHistoryEntry, error) {
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := dbConn.Exec("INSERT INTO "+historyTableName+" (sql, created_at) VALUES (?, ?)", sqlText, createdAt)
	if err != nil {
		return QueryHistoryEntry{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return QueryHistoryEntry{}, err
	}

	return QueryHistoryEntry{
		ID:        id,
		SQL:       sqlText,
		CreatedAt: createdAt,
	}, nil
}
