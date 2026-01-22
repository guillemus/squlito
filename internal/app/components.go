package app

import (
	"fmt"
	"strings"

	"github.com/awesome-gocui/gocui"

	"squlito/internal/tableformat"
)

func Sidebar(app *App, view *gocui.View) {
	view.Clear()

	width, height := view.Size()
	if width < 1 || height < 1 {
		return
	}

	app.updateSidebarScroll(height)
	_ = view.SetOrigin(0, app.sidebarScroll)

	for index, table := range app.tables {
		prefix := "  "
		if index == app.selectedTableIndex {
			prefix = "> "
		}

		line := prefix + table.Name
		_, _ = fmt.Fprintln(view, line)
	}
}

func RowsHeader(app *App, view *gocui.View, tableView tableformat.TableRender) {
	view.Clear()

	if tableView.Header == "" {
		return
	}

	_ = view.SetOrigin(app.scrollX, 0)
	_, _ = fmt.Fprintln(view, tableView.Header)
}

func RowsBody(app *App, view *gocui.View, tableView tableformat.TableRender, viewOffset int, messageView bool) {
	view.Clear()

	rowScrollDelta := 0
	if app.viewMode == viewTable {
		rowScrollDelta = viewOffset - app.tableState.BufferStart
	} else {
		rowScrollDelta = viewOffset
	}

	if rowScrollDelta < 0 {
		rowScrollDelta = 0
	}

	if messageView {
		rowScrollDelta = 0
	}

	_ = view.SetOrigin(app.scrollX, rowScrollDelta)

	if tableView.Body == "" {
		return
	}

	_, _ = fmt.Fprint(view, tableView.Body)
}

func QueryPanel(app *App, view *gocui.View) {
	if app.queryState.SQL == "" {
		view.Title = "Query"
		return
	}

	view.Title = truncateTitle(app.queryState.SQL)
}

func StatusBar(app *App, view *gocui.View) {
	view.Clear()
	width, _ := view.Size()

	left := app.buildStatusLeft()
	right := app.buildStatusRight()
	line := renderStatusLine(width, left, right)
	_, _ = fmt.Fprint(view, line)
}

func Modal(app *App, view *gocui.View) {
	view.Clear()
	view.Title = app.modalTitle

	if app.modalScroll < 0 {
		app.modalScroll = 0
	}

	_ = view.SetOrigin(0, app.modalScroll)
	_, _ = fmt.Fprint(view, app.modalBody)
}

func ModalBackdrop(view *gocui.View) {
	if view == nil {
		return
	}

	view.Clear()
	width, height := view.Size()
	if width < 1 || height < 1 {
		return
	}

	line := strings.Repeat(" ", width)
	for row := 0; row < height; row += 1 {
		_, _ = fmt.Fprint(view, line)
		if row < height-1 {
			_, _ = fmt.Fprint(view, "\n")
		}
	}
}
