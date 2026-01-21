package app

import (
	"github.com/awesome-gocui/gocui"
)

func (app *App) bindKeys() error {
    gui := app.gui

    if err := gui.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, app.quit); err != nil {
        return err
    }
    if err := gui.SetKeybinding("", gocui.KeyEsc, gocui.ModNone, app.quit); err != nil {
        return err
    }
    if err := gui.SetKeybinding("", 'q', gocui.ModNone, app.quit); err != nil {
        return err
    }
    if err := gui.SetKeybinding("", gocui.KeyTab, gocui.ModNone, app.handleTab); err != nil {
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

    if viewName != "" {
        _, err := app.gui.SetCurrentView(viewName)
        if err != nil && err != gocui.ErrUnknownView {
            return err
        }
    }

    app.gui.Cursor = true
    return nil
}

func (app *App) quit(gui *gocui.Gui, view *gocui.View) error {
    return gocui.ErrQuit
}

func (app *App) handleTab(gui *gocui.Gui, view *gocui.View) error {
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
    app.viewMode = viewTable
    err := app.setFocus(focusRows)
    if err != nil {
        return err
    }

    return app.render()
}

func (app *App) handleRowsDown(gui *gocui.Gui, view *gocui.View) error {
    return app.scrollRows(1)
}

func (app *App) handleRowsUp(gui *gocui.Gui, view *gocui.View) error {
    return app.scrollRows(-1)
}

func (app *App) handleRowsLeft(gui *gocui.Gui, view *gocui.View) error {
    return app.scrollHorizontal(-1)
}

func (app *App) handleRowsRight(gui *gocui.Gui, view *gocui.View) error {
    return app.scrollHorizontal(1)
}

func (app *App) handleQuerySubmit(gui *gocui.Gui, view *gocui.View) error {
    content := view.Buffer()
    err := app.runQuery(content)
    if err != nil {
        return nil
    }

    return app.render()
}

func (app *App) handleQueryNewline(gui *gocui.Gui, view *gocui.View) error {
    view.EditWrite('\n')
    return nil
}

func (app *App) handleSidebarClick(gui *gocui.Gui, view *gocui.View) error {
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
    err := app.setFocus(focusRows)
    if err != nil {
        return err
    }

    return app.render()
}

func (app *App) handleQueryClick(gui *gocui.Gui, view *gocui.View) error {
    err := app.setFocus(focusQuery)
    if err != nil {
        return err
    }

    return app.render()
}

func (app *App) handleRowsWheelDown(gui *gocui.Gui, view *gocui.View) error {
    return app.scrollRows(3)
}

func (app *App) handleRowsWheelUp(gui *gocui.Gui, view *gocui.View) error {
    return app.scrollRows(-3)
}

func (app *App) scrollRows(delta int) error {
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
    if delta == 0 {
        return nil
    }

    step := maxInt(1, app.scrollState.ViewportWidth/scrollStepDivisor)
    app.scrollX += delta * step
    return app.render()
}
