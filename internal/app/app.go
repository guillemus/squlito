package app

import (
	"database/sql"
	"fmt"

	"github.com/awesome-gocui/gocui"

	"squlito/internal/db"
)

type App struct {
	dbPath string
	db     *sql.DB
	gui    *gocui.Gui

	focusArea FocusArea
	viewMode  ViewMode

	tables             []db.SqliteTable
	selectedTableIndex int

	tableState TableState
	queryState QueryState

	scrollState   ScrollState
	scrollX       int
	sidebarScroll int
}

func Run(dbPath string) error {
	gui, err := gocui.NewGui(gocui.OutputNormal, false)
	if err != nil {
		return err
	}
	defer gui.Close()

	app := NewApp(dbPath, gui)
	err = app.Init()
	if err != nil {
		return err
	}
	defer app.Close()

	gui.SetManagerFunc(app.layout)
	gui.Mouse = true

	err = app.bindKeys()
	if err != nil {
		return err
	}

	err = app.setFocus(focusSidebar)
	if err != nil {
		return err
	}

	err = gui.MainLoop()
	if err != nil && err != gocui.ErrQuit {
		return err
	}

	return nil
}

func NewApp(dbPath string, gui *gocui.Gui) *App {
	return &App{
		dbPath:    dbPath,
		gui:       gui,
		focusArea: focusSidebar,
		viewMode:  viewTable,
		tableState: TableState{
			BufferSize: bufferSize,
		},
	}
}

func (app *App) Init() error {
	dbConn, err := db.OpenDatabase(app.dbPath)
	if err != nil {
		app.tableState.Error = err.Error()
		return err
	}

	app.db = dbConn

	tables, err := db.ListUserTables(app.db)
	if err != nil {
		app.tableState.Error = err.Error()
		return err
	}

	app.tables = tables
	if len(tables) == 0 {
		return nil
	}

	err = app.setSelectedTable(0)
	if err != nil {
		return err
	}

	return nil
}

func (app *App) Close() {
	if app.db == nil {
		return
	}

	err := app.db.Close()
	if err != nil {
		fmt.Println(err)
	}
}

func (app *App) layout(gui *gocui.Gui) error {
	maxX, maxY := gui.Size()
	if maxX < 40 || maxY < 10 {
		return app.renderTiny(gui, maxX, maxY)
	}

	metrics := calculateLayout(maxX, maxY)
	err := app.layoutViews(gui, metrics, maxX, maxY)
	if err != nil {
		return err
	}

	return app.render()
}

func (app *App) layoutViews(gui *gocui.Gui, metrics layoutMetrics, maxX int, maxY int) error {
	sidebarX1 := metrics.sidebarWidth - 1
	mainX0 := metrics.sidebarWidth
	usableHeight := maxY - metrics.statusHeight

	headerY1 := metrics.headerHeight - 1
	rowsY0 := metrics.headerHeight
	rowsY1 := rowsY0 + metrics.rowsHeight - 1
	queryY0 := rowsY1 + 1
	queryY1 := usableHeight - 1
	statusY0 := usableHeight
	statusY1 := maxY - 1

	sidebarView, err := gui.SetView("sidebar", 0, 0, sidebarX1, usableHeight-1, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	if err == gocui.ErrUnknownView {
		sidebarView.Title = "Tables"
		sidebarView.Wrap = false
	}

	rowsHeaderView, err := gui.SetView("rowsHeader", mainX0, 0, maxX-1, headerY1, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	if err == gocui.ErrUnknownView {
		rowsHeaderView.Wrap = false
	}

	rowsBodyView, err := gui.SetView("rowsBody", mainX0, rowsY0, maxX-1, rowsY1, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	if err == gocui.ErrUnknownView {
		rowsBodyView.Wrap = false
	}

	queryView, err := gui.SetView("query", mainX0, queryY0, maxX-1, queryY1, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	if err == gocui.ErrUnknownView {
		queryView.Title = "Query"
		queryView.Wrap = true
		queryView.Editable = true
		queryView.Editor = loggingEditor{next: gocui.DefaultEditor}
	}

	statusView, err := gui.SetView("status", 0, statusY0, maxX-1, statusY1, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}
	if err == gocui.ErrUnknownView {
		statusView.Frame = false
		statusView.Wrap = false
	}

	return nil
}

func (app *App) renderTiny(gui *gocui.Gui, maxX int, maxY int) error {
	if maxX < 2 || maxY < 2 {
		return nil
	}

	statusView, err := gui.SetView("status", 0, 0, maxX-1, maxY-1, 0)
	if err != nil && err != gocui.ErrUnknownView {
		return err
	}

	statusView.Clear()
	_, _ = fmt.Fprintln(statusView, "terminal too small")
	return nil
}
