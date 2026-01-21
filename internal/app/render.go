package app

import (
	"fmt"
	"strings"

	"squlito/internal/tableformat"
)

func (app *App) render() error {
	sidebarView, err := app.gui.View("sidebar")
	if err != nil {
		return nil
	}
	rowsHeaderView, err := app.gui.View("rowsHeader")
	if err != nil {
		return nil
	}
	rowsBodyView, err := app.gui.View("rowsBody")
	if err != nil {
		return nil
	}
	statusView, err := app.gui.View("status")
	if err != nil {
		return nil
	}

	queryView, _ := app.gui.View("query")
	if queryView != nil {
		QueryPanel(app, queryView)
	}

	Sidebar(app, sidebarView)

	viewportWidth, viewportHeight := rowsBodyView.Size()
	if viewportWidth < 1 {
		viewportWidth = 1
	}
	if viewportHeight < 1 {
		viewportHeight = 1
	}

	viewRowCount := app.currentRowCount()

	app.scrollState = ScrollState{
		ViewportRows:      viewportHeight,
		ViewportWidth:     viewportWidth,
		OverflowY:         viewRowCount > viewportHeight,
		OverflowX:         false,
		TableContentWidth: 0,
	}

	app.syncOffsets(viewRowCount, viewportHeight)

	viewOffset := app.currentOffset()

	tableView, messageView := app.buildTableView()

	contentWidth := tableView.Width
	app.scrollState.TableContentWidth = contentWidth
	app.scrollState.OverflowX = contentWidth > viewportWidth

	if messageView {
		app.scrollX = 0
	}

	if !app.scrollState.OverflowX {
		app.scrollX = 0
	}

	maxScrollX := maxInt(0, contentWidth-viewportWidth)
	app.scrollX = clampInt(app.scrollX, 0, maxScrollX)

	rowsHeaderView.Title = app.getRowsTitle()

	RowsHeader(app, rowsHeaderView, tableView)
	RowsBody(app, rowsBodyView, tableView, viewOffset, messageView)
	StatusBar(app, statusView)

	return nil
}

func (app *App) buildTableView() (tableformat.TableRender, bool) {
	isQueryMode := app.viewMode == viewQuery

	visibleRows := app.tableState.Rows
	visibleColumns := app.tableState.Columns
	visibleError := app.tableState.Error

	if isQueryMode {
		visibleRows = app.queryState.AllRows
		visibleColumns = app.queryState.Columns
		visibleError = app.queryState.Error
	}

	if visibleError != "" {
		width := measureMessageWidth(visibleError)
		return tableformat.TableRender{Header: "", Body: visibleError, Width: width, RowCount: 0}, true
	}

	if !isQueryMode && app.tableState.Name == "" {
		body := "No table selected"
		width := measureMessageWidth(body)
		return tableformat.TableRender{Header: "", Body: body, Width: width, RowCount: 0}, true
	}

	if len(visibleColumns) == 0 {
		body := "(empty)"
		width := measureMessageWidth(body)
		return tableformat.TableRender{Header: "", Body: body, Width: width, RowCount: 0}, true
	}

	tableView := tableformat.ComputeTable(tableformat.ComputeTableConfig{
		Columns: visibleColumns,
		Rows:    visibleRows,
		MaxRows: 0,
	})

	return tableView, false
}

func (app *App) currentRowCount() int {
	if app.viewMode == viewQuery {
		return len(app.queryState.AllRows)
	}

	return app.tableState.TotalRows
}

func (app *App) currentOffset() int {
	if app.viewMode == viewQuery {
		return app.queryState.Offset
	}

	return app.tableState.Offset
}

func (app *App) syncOffsets(viewRowCount int, viewportRows int) {
	if app.viewMode == viewQuery {
		maxOffset := maxInt(0, viewRowCount-viewportRows)
		app.queryState.Offset = clampInt(app.queryState.Offset, 0, maxOffset)
		return
	}

	if app.tableState.Name == "" {
		app.tableState.Offset = 0
		app.tableState.BufferStart = 0
		return
	}

	maxOffset := maxInt(0, app.tableState.TotalRows-viewportRows)
	nextOffset := clampInt(app.tableState.Offset, 0, maxOffset)
	nextBufferStart := app.tableState.BufferStart
	bufferEnd := nextBufferStart + app.tableState.BufferSize

	if !app.scrollState.OverflowY {
		nextBufferStart = 0
	}

	if nextOffset < nextBufferStart {
		nextBufferStart = nextOffset
	}

	if nextOffset >= bufferEnd {
		nextBufferStart = maxInt(0, nextOffset-app.tableState.BufferSize+1)
	}

	app.tableState.Offset = nextOffset
	if nextBufferStart == app.tableState.BufferStart {
		return
	}

	app.tableState.BufferStart = nextBufferStart
	_ = app.reloadTableBuffer()
}

func (app *App) updateSidebarScroll(viewHeight int) {
	if viewHeight <= 0 {
		return
	}

	if app.selectedTableIndex < app.sidebarScroll {
		app.sidebarScroll = app.selectedTableIndex
	}

	if app.selectedTableIndex >= app.sidebarScroll+viewHeight {
		app.sidebarScroll = app.selectedTableIndex - viewHeight + 1
	}

	if app.sidebarScroll < 0 {
		app.sidebarScroll = 0
	}
}

func (app *App) getRowsTitle() string {
	if app.viewMode == viewQuery {
		return truncateTitle(app.queryState.SQL)
	}

	if app.tableState.Name != "" {
		return app.tableState.Name
	}

	return "Rows"
}

func (app *App) buildStatusLeft() string {
	if app.viewMode == viewQuery {
		if app.queryState.Error != "" {
			return "Error: " + app.queryState.Error
		}

		if app.queryState.Running {
			return "Running query..."
		}

		count := len(app.queryState.AllRows)
		if app.queryState.Truncated {
			return fmt.Sprintf("Query rows %d (truncated at %d)", count, queryRowCap)
		}

		return fmt.Sprintf("Query rows %d", count)
	}

	if app.tableState.Error != "" {
		return "Error: " + app.tableState.Error
	}

	if app.tableState.Name == "" {
		return "No table selected"
	}

	showStart, showEnd := app.currentRowRange()
	return fmt.Sprintf("Rows %d  Showing %d-%d", app.tableState.TotalRows, showStart, showEnd)
}

func (app *App) buildStatusRight() string {
	if app.focusArea == focusSidebar {
		return "Tab rows  Enter open  q quit"
	}

	if app.focusArea == focusRows {
		return "Tab query  j/k scroll  h/l pan  q quit"
	}

	if app.focusArea == focusQuery {
		return "Enter run  Shift+Enter newline  Tab tables  q quit"
	}

	return "Tab cycle  q quit"
}

func (app *App) currentRowRange() (int, int) {
	viewRowCount := app.currentRowCount()
	viewOffset := app.currentOffset()
	viewportRows := app.scrollState.ViewportRows

	if viewRowCount == 0 {
		return 0, 0
	}

	showStart := viewOffset + 1
	showEnd := viewOffset + viewportRows
	if viewRowCount > 0 {
		showEnd = minInt(viewRowCount, showEnd)
	}

	return showStart, showEnd
}

func renderStatusLine(width int, left string, right string) string {
	if width <= 0 {
		return ""
	}

	if right == "" {
		return truncateLine(left, width)
	}

	combined := len(left) + len(right) + 1
	if combined > width {
		availableLeft := width - len(right) - 1
		if availableLeft < 0 {
			return truncateLine(right, width)
		}
		left = truncateLine(left, availableLeft)
	}

	padding := width - len(left) - len(right)
	if padding < 1 {
		padding = 1
	}

	return left + strings.Repeat(" ", padding) + right
}
