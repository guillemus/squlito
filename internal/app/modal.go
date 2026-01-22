package app

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/awesome-gocui/gocui"

	"squlito/internal/tableformat"
)

const modalViewName = "modal"

const modalBackdropViewName = "modalBackdrop"

func (app *App) layoutModal(gui *gocui.Gui, maxX int, maxY int) error {
	width := int(float64(maxX) * 0.7)
	height := int(float64(maxY) * 0.6)

	if width < 30 {
		width = 30
	}
	if height < 6 {
		height = 6
	}

	if width > maxX-4 {
		width = maxX - 4
	}
	if height > maxY-4 {
		height = maxY - 4
	}

	if width < 2 || height < 2 {
		return nil
	}

	x0 := (maxX - width) / 2
	y0 := (maxY - height) / 2
	x1 := x0 + width - 1
	y1 := y0 + height - 1

	view, err := gui.SetView(modalViewName, x0, y0, x1, y1, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	if err == gocui.ErrUnknownView {
		view.Title = app.modalTitle
		view.Wrap = true
		view.Frame = true
	}

	_, _ = gui.SetViewOnTop(modalViewName)
	return nil
}

func (app *App) layoutModalBackdrop(gui *gocui.Gui, maxX int, maxY int) error {
	if maxX < 2 || maxY < 2 {
		return nil
	}

	view, err := gui.SetView(modalBackdropViewName, 0, 0, maxX-1, maxY-1, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	if err == gocui.ErrUnknownView {
		view.Frame = false
		view.Wrap = false
		view.BgColor = gocui.ColorBlack
		view.FgColor = gocui.ColorBlack
	}

	_, _ = gui.SetViewOnTop(modalBackdropViewName)
	return nil
}

func (app *App) clearModal(gui *gocui.Gui) {
	if gui == nil {
		return
	}

	_, err := gui.View(modalViewName)
	if err != nil {
		app.clearModalBackdrop(gui)
		return
	}

	_ = gui.DeleteView(modalViewName)
	app.clearModalBackdrop(gui)
}

func (app *App) clearModalBackdrop(gui *gocui.Gui) {
	if gui == nil {
		return
	}

	_, err := gui.View(modalBackdropViewName)
	if err != nil {
		return
	}

	_ = gui.DeleteView(modalBackdropViewName)
}

func (app *App) openModal(title string, body string) error {
	app.modalOpen = true
	app.modalTitle = title
	app.modalBody = body
	app.modalScroll = 0
	app.modalPrevFocus = app.focusArea

	if strings.TrimSpace(app.modalTitle) == "" {
		app.modalTitle = "Value"
	}

	return app.setFocus(focusModal)
}

func (app *App) closeModal() error {
	if !app.modalOpen {
		return nil
	}

	app.modalOpen = false
	app.modalTitle = ""
	app.modalBody = ""
	app.modalScroll = 0

	return app.setFocus(app.modalPrevFocus)
}

func (app *App) openModalForCell(view *gocui.View) (bool, error) {
	if view == nil {
		return false, nil
	}

	if app.modalOpen {
		return false, nil
	}

	if app.viewMode == viewTable && app.tableState.Name == "" {
		return false, nil
	}

	if app.viewMode == viewQuery && app.queryState.Error != "" {
		return false, nil
	}

	cursorX, cursorY := view.Cursor()
	originX, originY := view.Origin()
	x := cursorX + originX
	y := cursorY + originY

	if x < 0 || y < 0 {
		return false, nil
	}

	tableView, messageView := app.buildTableView()
	if messageView {
		return false, nil
	}

	if y >= tableView.RowCount {
		return false, nil
	}

	colIndex := hitTestColumn(tableView, x)
	if colIndex < 0 {
		return false, nil
	}

	columns := app.tableState.Columns
	rows := app.tableState.Rows
	if app.viewMode == viewQuery {
		columns = app.queryState.Columns
		rows = app.queryState.AllRows
	}

	if colIndex >= len(columns) || y >= len(rows) {
		return false, nil
	}

	columnName := columns[colIndex]
	row := rows[y]
	value, ok := row[columnName]
	if !ok {
		return false, nil
	}

	raw := tableformat.FormatCell(value)
	columnWidth := tableView.ColumnWidths[colIndex]
	if len(raw) <= columnWidth {
		return false, nil
	}

	formatted := maybeIndentJSON(raw)

	title := "Value"
	if columnName != "" {
		title = "Value: " + columnName
	}

	return true, app.openModal(title, formatted)
}

func hitTestColumn(tableView tableformat.TableRender, x int) int {
	if x < 0 {
		return -1
	}

	currentX := 0
	for i, width := range tableView.ColumnWidths {
		start := currentX
		end := start + width
		if x >= start && x < end {
			return i
		}

		currentX = end + tableView.SeparatorWidth
		if x < currentX {
			return -1
		}
	}

	return -1
}

func maybeIndentJSON(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw
	}

	first := trimmed[0]
	if first != '{' && first != '[' {
		return raw
	}

	var buffer bytes.Buffer
	err := json.Indent(&buffer, []byte(trimmed), "", "    ")
	if err != nil {
		return raw
	}

	return buffer.String()
}
