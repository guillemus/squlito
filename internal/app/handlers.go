package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/awesome-gocui/gocui"
)

func (app *App) bindKeys() error {
	gui := app.gui

	if err := gui.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, app.quit); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", gocui.KeyEsc, gocui.ModNone, app.handleGlobalEsc); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", 'q', gocui.ModNone, app.quit); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", gocui.KeyTab, gocui.ModNone, app.handleTab); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", gocui.KeyCtrlH, gocui.ModNone, app.handlePaneLeft); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", gocui.KeyCtrlJ, gocui.ModNone, app.handlePaneDown); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", gocui.KeyCtrlK, gocui.ModNone, app.handlePaneUp); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", gocui.KeyCtrlL, gocui.ModNone, app.handlePaneRight); err != nil {
		return err
	}

	if err := gui.SetKeybinding("sidebar", gocui.KeyArrowDown, gocui.ModNone, app.handleSidebarDown); err != nil {
		return err
	}
	if err := gui.SetKeybinding("sidebar", 'j', gocui.ModNone, app.handleSidebarDown); err != nil {
		return err
	}
	if err := gui.SetKeybinding("sidebar", gocui.KeyArrowUp, gocui.ModNone, app.handleSidebarUp); err != nil {
		return err
	}
	if err := gui.SetKeybinding("sidebar", 'k', gocui.ModNone, app.handleSidebarUp); err != nil {
		return err
	}
	if err := gui.SetKeybinding("sidebar", gocui.KeyEnter, gocui.ModNone, app.handleSidebarEnter); err != nil {
		return err
	}

	if err := gui.SetKeybinding("rowsBody", gocui.KeyArrowDown, gocui.ModNone, app.handleRowsDown); err != nil {
		return err
	}
	if err := gui.SetKeybinding("rowsBody", 'j', gocui.ModNone, app.handleRowsDown); err != nil {
		return err
	}
	if err := gui.SetKeybinding("rowsBody", gocui.KeyArrowUp, gocui.ModNone, app.handleRowsUp); err != nil {
		return err
	}
	if err := gui.SetKeybinding("rowsBody", 'k', gocui.ModNone, app.handleRowsUp); err != nil {
		return err
	}
	if err := gui.SetKeybinding("rowsBody", gocui.KeyArrowLeft, gocui.ModNone, app.handleRowsLeft); err != nil {
		return err
	}
	if err := gui.SetKeybinding("rowsBody", 'h', gocui.ModNone, app.handleRowsLeft); err != nil {
		return err
	}
	if err := gui.SetKeybinding("rowsBody", gocui.KeyArrowRight, gocui.ModNone, app.handleRowsRight); err != nil {
		return err
	}
	if err := gui.SetKeybinding("rowsBody", 'l', gocui.ModNone, app.handleRowsRight); err != nil {
		return err
	}

	if err := gui.SetKeybinding("query", gocui.KeyEnter, gocui.ModNone, app.handleQuerySubmit); err != nil {
		return err
	}
	if err := gui.SetKeybinding("query", gocui.KeyEnter, gocui.ModShift, app.handleQueryNewline); err != nil {
		return err
	}
	if err := gui.SetKeybinding("query", gocui.KeyCtrlJ, gocui.ModNone, app.handleQueryNewline); err != nil {
		return err
	}
	if err := gui.SetKeybinding("query", gocui.KeyArrowUp, gocui.ModNone, app.handleQueryHistoryPrev); err != nil {
		return err
	}
	if err := gui.SetKeybinding("query", gocui.KeyArrowDown, gocui.ModNone, app.handleQueryHistoryNext); err != nil {
		return err
	}

	if err := gui.SetKeybinding("sidebar", gocui.MouseLeft, gocui.ModNone, app.handleSidebarClick); err != nil {
		return err
	}
	if err := gui.SetKeybinding("rowsBody", gocui.MouseLeft, gocui.ModNone, app.handleRowsClick); err != nil {
		return err
	}
	if err := gui.SetKeybinding("query", gocui.MouseLeft, gocui.ModNone, app.handleQueryClick); err != nil {
		return err
	}
	if err := gui.SetKeybinding("rowsBody", gocui.MouseWheelDown, gocui.ModNone, app.handleRowsWheelDown); err != nil {
		return err
	}
	if err := gui.SetKeybinding("rowsBody", gocui.MouseWheelUp, gocui.ModNone, app.handleRowsWheelUp); err != nil {
		return err
	}

	if err := gui.SetKeybinding(modalViewName, gocui.KeyEsc, gocui.ModNone, app.handleModalClose); err != nil {
		return err
	}
	if err := gui.SetKeybinding(modalViewName, gocui.KeyEnter, gocui.ModNone, app.handleModalClose); err != nil {
		return err
	}
	if err := gui.SetKeybinding(modalViewName, 'q', gocui.ModNone, app.handleModalClose); err != nil {
		return err
	}
	if err := gui.SetKeybinding(modalViewName, gocui.KeyArrowDown, gocui.ModNone, app.handleModalDown); err != nil {
		return err
	}
	if err := gui.SetKeybinding(modalViewName, 'j', gocui.ModNone, app.handleModalDown); err != nil {
		return err
	}
	if err := gui.SetKeybinding(modalViewName, gocui.KeyArrowUp, gocui.ModNone, app.handleModalUp); err != nil {
		return err
	}
	if err := gui.SetKeybinding(modalViewName, 'k', gocui.ModNone, app.handleModalUp); err != nil {
		return err
	}

	return nil
}

func (app *App) setFocus(area FocusArea) error {
	app.focusArea = area

	var viewName string
	if area == focusSidebar {
		viewName = "sidebar"
	}
	if area == focusRows {
		viewName = "rowsBody"
	}
	if area == focusQuery {
		viewName = "query"
	}
	if area == focusModal {
		viewName = modalViewName
	}

	if viewName != "" {
		_, err := app.gui.SetCurrentView(viewName)
		if err != nil && err != gocui.ErrUnknownView {
			return err
		}
	}

	app.gui.Cursor = area == focusQuery
	return nil
}

func (app *App) quit(gui *gocui.Gui, view *gocui.View) error {
	logEvent("quit")
	return gocui.ErrQuit
}

func (app *App) handleTab(gui *gocui.Gui, view *gocui.View) error {
	logEvent("tab")
	next := focusSidebar
	if app.focusArea == focusSidebar {
		next = focusRows
	}
	if app.focusArea == focusRows {
		next = focusQuery
	}
	if app.focusArea == focusQuery {
		next = focusSidebar
	}

	err := app.setFocus(next)
	if err != nil {
		return err
	}

	return app.render()
}

func (app *App) handleSidebarDown(gui *gocui.Gui, view *gocui.View) error {
	logEvent("sidebar-down")
	if len(app.tables) == 0 {
		return nil
	}

	next := clampInt(app.selectedTableIndex+1, 0, len(app.tables)-1)
	err := app.setSelectedTable(next)
	if err != nil {
		return nil
	}

	return app.render()
}

func (app *App) handleSidebarUp(gui *gocui.Gui, view *gocui.View) error {
	logEvent("sidebar-up")
	if len(app.tables) == 0 {
		return nil
	}

	next := clampInt(app.selectedTableIndex-1, 0, len(app.tables)-1)
	err := app.setSelectedTable(next)
	if err != nil {
		return nil
	}

	return app.render()
}

func (app *App) handleSidebarEnter(gui *gocui.Gui, view *gocui.View) error {
	logEvent("sidebar-enter")
	app.viewMode = viewTable
	err := app.setFocus(focusRows)
	if err != nil {
		return err
	}

	return app.render()
}

func (app *App) handleRowsDown(gui *gocui.Gui, view *gocui.View) error {
	logEvent("rows-down")
	return app.scrollRows(1)
}

func (app *App) handleRowsUp(gui *gocui.Gui, view *gocui.View) error {
	logEvent("rows-up")
	return app.scrollRows(-1)
}

func (app *App) handleRowsLeft(gui *gocui.Gui, view *gocui.View) error {
	logEvent("rows-left")
	return app.scrollHorizontal(-1)
}

func (app *App) handleRowsRight(gui *gocui.Gui, view *gocui.View) error {
	logEvent("rows-right")
	return app.scrollHorizontal(1)
}

func (app *App) handlePaneLeft(gui *gocui.Gui, view *gocui.View) error {
	logEvent("pane-left")
	if app.focusArea == focusSidebar {
		return nil
	}

	err := app.setFocus(focusSidebar)
	if err != nil {
		return err
	}

	return app.render()
}

func (app *App) handleGlobalEsc(gui *gocui.Gui, view *gocui.View) error {
	logEvent("esc")
	if app.modalOpen {
		return app.handleModalClose(gui, view)
	}

	return app.quit(gui, view)
}

func (app *App) handlePaneRight(gui *gocui.Gui, view *gocui.View) error {
	logEvent("pane-right")
	if app.focusArea == focusRows {
		return nil
	}

	err := app.setFocus(focusRows)
	if err != nil {
		return err
	}

	return app.render()
}

func (app *App) handlePaneDown(gui *gocui.Gui, view *gocui.View) error {
	logEvent("pane-down")
	if app.focusArea == focusQuery {
		return nil
	}

	err := app.setFocus(focusQuery)
	if err != nil {
		return err
	}

	return app.render()
}

func (app *App) handlePaneUp(gui *gocui.Gui, view *gocui.View) error {
	logEvent("pane-up")
	if app.focusArea != focusQuery {
		return nil
	}

	err := app.setFocus(focusRows)
	if err != nil {
		return err
	}

	return app.render()
}

func (app *App) handleQuerySubmit(gui *gocui.Gui, view *gocui.View) error {
	logEvent("query-submit")
	content := view.Buffer()
	err := app.runQuery(content)
	if err != nil {
		return nil
	}

	return app.render()
}

func (app *App) handleQueryHistoryPrev(gui *gocui.Gui, view *gocui.View) error {
	logEvent("query-history-prev")
	return app.moveHistorySelection(view, 1)
}

func (app *App) handleQueryHistoryNext(gui *gocui.Gui, view *gocui.View) error {
	logEvent("query-history-next")
	return app.moveHistorySelection(view, -1)
}

func (app *App) handleQueryNewline(gui *gocui.Gui, view *gocui.View) error {
	start := time.Now()
	app.resetHistorySelection()
	view.EditWrite('\n')
	appendLatencyLog(start, time.Since(start))
	return nil
}

func (app *App) handleSidebarClick(gui *gocui.Gui, view *gocui.View) error {
	logEvent("sidebar-click")
	err := app.setFocus(focusSidebar)
	if err != nil {
		return err
	}

	_, cursorY := view.Cursor()
	index := app.sidebarScroll + cursorY
	if index < 0 || index >= len(app.tables) {
		return nil
	}

	err = app.setSelectedTable(index)
	if err != nil {
		return nil
	}

	return app.render()
}

func (app *App) handleRowsClick(gui *gocui.Gui, view *gocui.View) error {
	logEvent("rows-click")
	err := app.setFocus(focusRows)
	if err != nil {
		return err
	}

	opened, err := app.openModalForCell(view)
	if err != nil {
		return err
	}
	if opened {
		return app.render()
	}

	return app.render()
}

func (app *App) handleQueryClick(gui *gocui.Gui, view *gocui.View) error {
	logEvent("query-click")
	err := app.setFocus(focusQuery)
	if err != nil {
		return err
	}

	return app.render()
}

func (app *App) handleRowsWheelDown(gui *gocui.Gui, view *gocui.View) error {
	logEvent("rows-wheel-down")
	return app.scrollRows(3)
}

func (app *App) handleRowsWheelUp(gui *gocui.Gui, view *gocui.View) error {
	logEvent("rows-wheel-up")
	return app.scrollRows(-3)
}

func (app *App) handleModalClose(gui *gocui.Gui, view *gocui.View) error {
	logEvent("modal-close")
	err := app.closeModal()
	if err != nil {
		return err
	}

	return app.render()
}

func (app *App) handleModalDown(gui *gocui.Gui, view *gocui.View) error {
	logEvent("modal-down")
	app.modalScroll += 1
	return app.render()
}

func (app *App) handleModalUp(gui *gocui.Gui, view *gocui.View) error {
	logEvent("modal-up")
	app.modalScroll -= 1
	return app.render()
}

func (app *App) scrollRows(delta int) error {
	logEvent("scroll-rows")
	if delta == 0 {
		return nil
	}

	if !app.scrollState.OverflowY {
		return nil
	}

	if app.viewMode == viewQuery {
		app.queryState.Offset += delta
		return app.render()
	}

	if app.tableState.Name == "" {
		return nil
	}

	app.tableState.Offset += delta
	return app.render()
}

func (app *App) scrollHorizontal(delta int) error {
	logEvent("scroll-horizontal")
	if delta == 0 {
		return nil
	}

	step := maxInt(1, app.scrollState.ViewportWidth/scrollStepDivisor)
	app.scrollX += delta * step
	return app.render()
}

func appendLatencyLog(start time.Time, duration time.Duration) {
	file, err := openLogFile()
	if err != nil {
		return
	}
	defer func() {
		_ = file.Close()
	}()

	_, _ = fmt.Fprintf(file, "%s shift-enter %s\n", start.Format(time.RFC3339Nano), duration)
}

func logEvent(name string) {
	file, err := openLogFile()
	if err != nil {
		return
	}
	defer func() {
		_ = file.Close()
	}()

	_, _ = fmt.Fprintf(file, "%s event %s\n", time.Now().Format(time.RFC3339Nano), name)
}

func logKeyEvent(source string, key gocui.Key, ch rune, mod gocui.Modifier) {
	file, err := openLogFile()
	if err != nil {
		return
	}
	defer func() {
		_ = file.Close()
	}()

	_, _ = fmt.Fprintf(file, "%s key %s key=%d ch=%d mod=%d\n", time.Now().Format(time.RFC3339Nano), source, key, ch, mod)
}

func openLogFile() (*os.File, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return os.OpenFile("debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	}

	path := filepath.Join(cwd, "debug.log")
	return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
}
