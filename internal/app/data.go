package app

import (
	"strings"

	"squlito/internal/db"
)

func (app *App) setSelectedTable(index int) error {
	if index < 0 || index >= len(app.tables) {
		return nil
	}

	app.selectedTableIndex = index
	tableName := app.tables[index].Name
	app.tableState.Name = tableName
	app.tableState.Offset = 0
	app.tableState.BufferStart = 0
	app.tableState.Error = ""
	app.viewMode = viewTable
	app.queryState.Error = ""
	app.queryState.Running = false

	err := app.reloadTableBuffer()
	if err != nil {
		return err
	}

	return nil
}

func (app *App) reloadTableBuffer() error {
	if app.tableState.Name == "" {
		return nil
	}

	page, err := db.GetTablePage(app.db, app.tableState.Name, app.tableState.BufferSize, app.tableState.BufferStart)
	if err != nil {
		app.tableState.Rows = nil
		app.tableState.Columns = nil
		app.tableState.TotalRows = 0
		app.tableState.Error = err.Error()
		return err
	}

	cols, err := db.GetTableColumns(app.db, app.tableState.Name)
	if err != nil {
		app.tableState.Rows = nil
		app.tableState.Columns = nil
		app.tableState.TotalRows = 0
		app.tableState.Error = err.Error()
		return err
	}

	columnNames := []string{}
	for _, col := range cols {
		columnNames = append(columnNames, col.Name)
	}

	app.tableState.TotalRows = page.TotalRows
	app.tableState.BufferStart = page.Offset
	app.tableState.Rows = page.Rows
	app.tableState.Columns = columnNames
	app.tableState.Error = ""

	return nil
}

func (app *App) runQuery(sqlText string) error {
	trimmed := strings.TrimSpace(sqlText)
	app.viewMode = viewQuery
	app.queryState.Offset = 0

	if trimmed == "" {
		app.queryState.SQL = ""
		app.queryState.AllRows = nil
		app.queryState.Columns = nil
		app.queryState.Error = "Query is empty"
		app.queryState.Running = false
		app.queryState.Truncated = false
		return nil
	}

	app.resetHistorySelection()
	app.recordHistory(trimmed)

	app.queryState.SQL = trimmed
	app.queryState.Running = true
	app.queryState.Error = ""
	app.queryState.Truncated = false

	result, err := db.QueryRows(app.db, trimmed, queryRowCap)
	if err != nil {
		app.queryState.AllRows = nil
		app.queryState.Columns = nil
		app.queryState.Error = err.Error()
		app.queryState.Running = false
		return err
	}

	app.queryState.AllRows = result.Rows
	app.queryState.Columns = result.Columns
	app.queryState.Truncated = result.Truncated
	app.queryState.Running = false
	app.queryState.Error = ""

	return nil
}
