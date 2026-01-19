import {
    createCliRenderer,
    type KeyEvent,
    type ScrollBoxRenderable,
    TextAttributes,
    type TextareaRenderable,
    type KeyBinding,
} from '@opentui/core'
import { createRoot, useKeyboard, useTerminalDimensions } from '@opentui/react'
import { type RefObject, useEffect, useMemo, useRef, useState } from 'react'

const BUFFER_SIZE = 200
const SCROLL_STEP_DIVISOR = 5
import {
    type SqliteRow,
    type SqliteTable,
    getTableColumns,
    getTablePage,
    listUserTables,
    openDatabase,
    parseDatabasePathFromArgs,
} from './db'
import { computeTable } from './table-format'

type FocusArea = 'sidebar' | 'rows' | 'query'

type TableState = {
    name: string
    totalRows: number
    offset: number
    bufferStart: number
    bufferSize: number
    rows: SqliteRow[]
    columns: string[]
    error: string | null
}

type ViewMode = 'table' | 'query'

type QueryState = {
    sql: string
    allRows: SqliteRow[]
    columns: string[]
    error: string | null
    running: boolean
}

function App(props: { dbPath: string; requestExit: () => void }) {
    const dims = useTerminalDimensions()

    const [focusArea, setFocusArea] = useState<FocusArea>('sidebar')

    const queryRef = useRef<TextareaRenderable>(null)

    const [viewMode, setViewMode] = useState<ViewMode>('table')
    const [queryState, setQueryState] = useState<QueryState>({
        sql: '',
        allRows: [],
        columns: [],
        error: null,
        running: false,
    })

    const [tables, setTables] = useState<SqliteTable[]>([])
    const [selectedTableIndex, setSelectedTableIndex] = useState(0)

    const rowsScrollRef = useRef<ScrollBoxRenderable>(null)
    const headerScrollRef = useRef<ScrollBoxRenderable>(null)

    const [tableState, setTableState] = useState<TableState>({
        name: '',
        totalRows: 0,
        offset: 0,
        bufferStart: 0,
        bufferSize: BUFFER_SIZE,
        rows: [],
        columns: [],
        error: null,
    })

    const QUERY_BOX_HEIGHT = 7

    const sidebarWidth = clamp(Math.floor(dims.width * 0.28), 22, 40)
    const mainWidth = Math.max(20, dims.width - sidebarWidth)
    const tableViewportWidth = Math.max(10, mainWidth - 4)
    const tableViewportHeight = Math.max(3, dims.height - 8 - QUERY_BOX_HEIGHT)

    const isQueryMode = viewMode === 'query'

    const visibleRows = isQueryMode ? queryState.allRows : tableState.rows
    const visibleColumns = isQueryMode ? queryState.columns : tableState.columns
    const visibleError = isQueryMode ? queryState.error : tableState.error

    const viewRowCount = isQueryMode ? queryState.allRows.length : tableState.totalRows
    const viewOffset = isQueryMode ? 0 : tableState.offset

    const tableView = useMemo(() => {
        if (visibleError) {
            const width = measureMessageWidth(visibleError)
            return {
                header: '',
                body: visibleError,
                width,
                rowCount: 0,
            }
        }

        if (!isQueryMode && tableState.name.length === 0) {
            const body = 'No table selected'
            return {
                header: '',
                body,
                width: measureMessageWidth(body),
                rowCount: 0,
            }
        }

        if (visibleColumns.length === 0) {
            const body = '(empty)'
            return {
                header: '',
                body,
                width: measureMessageWidth(body),
                rowCount: 0,
            }
        }

        return computeTable({
            columns: visibleColumns,
            rows: visibleRows,
        })
    }, [isQueryMode, tableState.name, visibleColumns, visibleError, visibleRows])

    const tableContentWidth = tableView.width + 2

    const scrollState = useMemo((): ScrollState => {
        const overflowY = tableView.rowCount > tableViewportHeight
        const overflowX = tableContentWidth > tableViewportWidth

        return {
            overflowY,
            overflowX,
            viewportRows: tableViewportHeight,
            viewportWidth: tableViewportWidth,
            tableContentWidth,
        }
    }, [tableView.rowCount, tableContentWidth, tableViewportHeight, tableViewportWidth])

    const sidebarFocused = focusArea === 'sidebar'
    const rowsFocused = focusArea === 'rows'
    const queryFocused = focusArea === 'query'

    const rowScrollDelta = viewOffset - tableState.bufferStart
    const showStart = viewRowCount === 0 ? 0 : viewOffset + 1
    let showEnd = viewOffset + tableViewportHeight
    if (viewRowCount > 0) {
        showEnd = Math.min(viewRowCount, showEnd)
    }

    useEffect(() => {
        const db = openDatabase(props.dbPath)

        try {
            const nextTables = listUserTables(db)
            setTables(nextTables)

            if (nextTables.length > 0) {
                const firstName = nextTables[0]?.name ?? ''
                if (firstName.length > 0) {
                    const page = getTablePage(db, firstName, BUFFER_SIZE, 0)

                    const columns: string[] = []
                    for (const c of getTableColumns(db, firstName)) {
                        columns.push(c.name)
                    }

                    setSelectedTableIndex(0)
                    setTableState((prev) => ({
                        ...prev,
                        name: firstName,
                        totalRows: page.totalRows,
                        offset: page.offset,
                        bufferStart: page.offset,
                        rows: page.rows,
                        columns,
                        error: null,
                    }))
                }
            }
        } catch (err) {
            const message = err instanceof Error ? err.message : String(err)
            setTableState((prev) => ({ ...prev, error: message }))
        } finally {
            db.close()
        }
    }, [props.dbPath])

    useEffect(() => {
        if (viewMode !== 'table') {
            return
        }

        const selectedName = tables[selectedTableIndex]?.name
        if (!selectedName) {
            return
        }

        if (tableState.name === selectedName) {
            return
        }

        setTableState((prev) => ({
            ...prev,
            name: selectedName,
            offset: 0,
            bufferStart: 0,
            error: null,
        }))

        if (rowsScrollRef.current) {
            rowsScrollRef.current.scrollTop = 0
        }
    }, [selectedTableIndex, tables, tableState.name, viewMode])

    useEffect(() => {
        if (viewMode !== 'table') {
            return
        }

        if (tableState.name.length === 0) {
            return
        }

        setTableState((prev) => {
            const maxOffset = Math.max(0, prev.totalRows - scrollState.viewportRows)
            const nextOffset = clamp(prev.offset, 0, maxOffset)
            let nextBufferStart = prev.bufferStart
            const bufferEnd = nextBufferStart + prev.bufferSize

            if (!scrollState.overflowY) {
                nextBufferStart = 0
            }

            if (nextOffset < nextBufferStart) {
                nextBufferStart = nextOffset
            }

            if (nextOffset >= bufferEnd) {
                nextBufferStart = Math.max(0, nextOffset - prev.bufferSize + 1)
            }

            const shouldUpdateOffset = nextOffset !== prev.offset
            const shouldUpdateBuffer = nextBufferStart !== prev.bufferStart

            if (!shouldUpdateOffset && !shouldUpdateBuffer) {
                return prev
            }

            return {
                ...prev,
                offset: nextOffset,
                bufferStart: nextBufferStart,
            }
        })
    }, [scrollState.viewportRows, tableState.name, tableState.totalRows, viewMode])

    useEffect(() => {
        if (viewMode !== 'table') {
            return
        }

        if (tableState.name.length === 0) {
            return
        }

        const db = openDatabase(props.dbPath)

        try {
            const page = getTablePage(
                db,
                tableState.name,
                tableState.bufferSize,
                tableState.bufferStart,
            )

            const columns: string[] = []
            for (const c of getTableColumns(db, tableState.name)) {
                columns.push(c.name)
            }

            const bufferStart = page.offset
            setTableState((prev) => ({
                ...prev,
                totalRows: page.totalRows,
                bufferStart,
                rows: page.rows,
                columns,
                error: null,
            }))
        } catch (err) {
            const message = err instanceof Error ? err.message : String(err)
            setTableState((prev) => ({
                ...prev,
                rows: [],
                columns: [],
                totalRows: 0,
                error: message,
            }))
        } finally {
            db.close()
        }
    }, [props.dbPath, tableState.name, tableState.bufferSize, tableState.bufferStart, viewMode])

    useEffect(() => {
        if (!rowsScrollRef.current) {
            return
        }

        const scrollbox = rowsScrollRef.current

        if (!scrollState.overflowY) {
            scrollbox.scrollTop = 0
        } else {
            scrollbox.scrollTop = Math.max(0, rowScrollDelta)
        }

        if (!scrollState.overflowX) {
            scrollbox.scrollLeft = 0
        }

        if (headerScrollRef.current) {
            headerScrollRef.current.scrollLeft = scrollbox.scrollLeft
        }
    }, [rowScrollDelta, scrollState.overflowX, scrollState.overflowY, tableState.bufferStart])

    useEffect(() => {
        const scrollbox = rowsScrollRef.current
        const headerBox = headerScrollRef.current
        if (!scrollbox || !headerBox) {
            return
        }

        const syncHeader = () => {
            headerBox.scrollLeft = scrollbox.scrollLeft
        }

        scrollbox.onMouse = syncHeader
        scrollbox.onKeyDown = syncHeader

        return () => {
            scrollbox.onMouse = undefined
            scrollbox.onKeyDown = undefined
        }
    }, [])

    useKeyboard((key) => {
        if (key.eventType === 'release') {
            return
        }

        if (focusArea === 'query') {
            if (key.name === 'tab') {
                setFocusArea('sidebar')
                key.preventDefault()
                key.stopPropagation()
                return
            }

            return
        }

        const handled = handleGlobalKey(key, {
            focusArea,
            setFocusArea,
            viewMode,
            setViewMode,
            tablesCount: tables.length,
            selectedTableIndex,
            setSelectedTableIndex,
            tableState,
            setTableState,
            scrollState,
            rowsScrollRef,
            headerScrollRef,
            requestExit: props.requestExit,
        })

        if (handled) {
            key.preventDefault()
            key.stopPropagation()
        }
    })

    const queryKeyBindings: KeyBinding[] = [
        { name: 'return', shift: true, action: 'newline' },
        { name: 'return', action: 'submit' },
    ]

    const runQuery = (sql: string): void => {
        const trimmed = sql.trim()
        if (trimmed.length === 0) {
            setQueryState((prev) => ({
                ...prev,
                sql: '',
                allRows: [],
                columns: [],
                error: 'Query is empty',
                running: false,
            }))
            return
        }

        setViewMode('query')
        setQueryState((prev) => ({
            ...prev,
            sql: trimmed,
            error: null,
            running: true,
        }))

        const db = openDatabase(props.dbPath)
        try {
            const stmt = db.query<SqliteRow, []>(trimmed)
            const rows = stmt.all()

            let columns = stmt.columnNames
            if (columns.length === 0 && rows.length > 0) {
                const first = rows[0]
                if (first) {
                    columns = Object.keys(first)
                }
            }

            setQueryState((prev) => ({
                ...prev,
                sql: trimmed,
                allRows: rows,
                columns,
                error: null,
                running: false,
            }))
        } catch (err) {
            const message = err instanceof Error ? err.message : String(err)
            setQueryState((prev) => ({
                ...prev,
                sql: trimmed,
                allRows: [],
                columns: [],
                error: message,
                running: false,
            }))
        } finally {
            db.close()
        }
    }

    return (
        <box flexDirection="row" width="100%" height="100%" backgroundColor="#0b1020">
            <box
                title={'Tables'}
                border={true}
                borderStyle="single"
                borderColor="#2a355a"
                focusedBorderColor={sidebarFocused ? '#7bdff2' : '#2a355a'}
                width={sidebarWidth}
                height="100%"
                flexDirection="column"
                padding={1}
            >
                <text attributes={TextAttributes.DIM} fg="#9aa4c5">
                    {props.dbPath}
                </text>
                <text attributes={TextAttributes.DIM} fg="#9aa4c5">
                    {sidebarFocused ? '[Tab] rows' : '[Tab] tables'}
                </text>
                <box height={1} />

                <scrollbox
                    focused={sidebarFocused}
                    style={{ flexGrow: 1, scrollY: true }}
                    viewportOptions={{ backgroundColor: '#0b1020' }}
                >
                    <box flexDirection="column" width="100%">
                        {tables.map((table, index) => {
                            const isSelected = index === selectedTableIndex
                            const rowBackground = isSelected ? '#1e2b4f' : '#0b1020'
                            const rowTextColor = isSelected ? '#d4defc' : '#cbd5f0'

                            return (
                                <box
                                    key={table.name}
                                    width="100%"
                                    paddingLeft={1}
                                    paddingRight={1}
                                    backgroundColor={rowBackground}
                                    onMouseDown={() => {
                                        setSelectedTableIndex(index)
                                        setViewMode('table')
                                        setFocusArea('sidebar')
                                    }}
                                >
                                    <text fg={rowTextColor}>{table.name}</text>
                                </box>
                            )
                        })}
                    </box>
                </scrollbox>

                <box height={1} />
                <text attributes={TextAttributes.DIM} fg="#9aa4c5">
                    {'↑↓ select  click to focus'}
                </text>
            </box>

            <box
                title={getRowsTitle({ tableState, queryState, viewMode })}
                border={true}
                borderStyle="single"
                borderColor="#2a355a"
                focusedBorderColor={rowsFocused ? '#f7c948' : '#2a355a'}
                flexGrow={1}
                height="100%"
                flexDirection="column"
                padding={1}
            >
                <box flexDirection="row" justifyContent="space-between" width="100%">
                    <text attributes={TextAttributes.DIM} fg="#9aa4c5">
                        {`Rows ${viewRowCount}  Showing ${showStart}-${showEnd}`}
                    </text>
                    <text attributes={TextAttributes.DIM} fg="#9aa4c5">
                        {rowsFocused ? '[Tab] tables' : '[Tab] rows'}
                    </text>
                </box>

                <box height={1} />

                {tableView.header.length > 0 && (
                    <scrollbox
                        ref={headerScrollRef}
                        focused={false}
                        style={{ height: 1, scrollX: scrollState.overflowX }}
                        viewportOptions={{ backgroundColor: '#121a33' }}
                    >
                        <box paddingLeft={1} paddingRight={1} height={1} width={tableContentWidth}>
                            <text fg="#d4defc" attributes={TextAttributes.BOLD}>
                                {tableView.header}
                            </text>
                        </box>
                    </scrollbox>
                )}

                <scrollbox
                    ref={rowsScrollRef}
                    focused={rowsFocused}
                    style={{
                        flexGrow: 1,
                        scrollY: scrollState.overflowY,
                        scrollX: scrollState.overflowX,
                        viewportCulling: true,
                    }}
                    viewportOptions={{ backgroundColor: '#0b1020' }}
                    onMouseScroll={(event) => {
                        if (!rowsFocused) {
                            return
                        }

                        if (!event.scroll) {
                            return
                        }

                        if (!scrollState.overflowY) {
                            return
                        }

                        const direction = event.scroll.direction
                        const delta = Math.max(1, Math.floor(event.scroll.delta))

                        if (direction === 'down') {
                            setTableState((prev) => ({ ...prev, offset: prev.offset + delta }))
                            return
                        }

                        if (direction === 'up') {
                            setTableState((prev) => ({
                                ...prev,
                                offset: Math.max(0, prev.offset - delta),
                            }))
                            return
                        }
                    }}
                >
                    <box
                        flexDirection="column"
                        width={tableContentWidth}
                        paddingLeft={1}
                        paddingRight={1}
                    >
                        <text fg="#cbd5f0">{tableView.body}</text>
                    </box>
                </scrollbox>

                <box height={1} />

                <box height={1} />

                <box
                    title={'Query'}
                    border={true}
                    borderStyle="single"
                    borderColor="#2a355a"
                    focusedBorderColor={queryFocused ? '#f7c948' : '#2a355a'}
                    height={QUERY_BOX_HEIGHT}
                    width="100%"
                    flexDirection="column"
                    paddingLeft={1}
                    paddingRight={1}
                    onMouseDown={() => {
                        setFocusArea('query')
                        queryRef.current?.focus()
                    }}
                >
                    <box height={1} />
                    <textarea
                        ref={queryRef}
                        focused={queryFocused}
                        placeholder={"Write SQL... (Enter runs, Shift+Enter newline)"}
                        keyBindings={queryKeyBindings}
                        onMouseDown={() => {
                            setFocusArea('query')
                            queryRef.current?.focus()
                        }}
                        onSubmit={() => {
                            const sql = queryRef.current?.plainText ?? ''
                            runQuery(sql)
                        }}
                        backgroundColor={'#0b1020'}
                        focusedBackgroundColor={'#121a33'}
                        textColor={'#cbd5f0'}
                        focusedTextColor={'#d4defc'}
                        style={{ flexGrow: 1 }}
                    />

                    <box height={1} />
                    <box flexDirection="row" justifyContent="space-between" width="100%">
                        <text attributes={TextAttributes.DIM} fg="#9aa4c5">
                            {queryState.error
                                ? `Error: ${queryState.error}`
                                : queryState.running
                                  ? 'Running query...'
                                  : 'Enter run  Shift+Enter newline  Tab focus'}
                        </text>
                        <text attributes={TextAttributes.DIM} fg="#9aa4c5">
                            {'q quit'}
                        </text>
                    </box>
                </box>
            </box>
        </box>
    )
}

type ScrollState = {
    overflowY: boolean
    overflowX: boolean
    viewportRows: number
    viewportWidth: number
    tableContentWidth: number
}

type KeyHandlingState = {
    focusArea: FocusArea
    setFocusArea: (area: FocusArea) => void

    viewMode: ViewMode
    setViewMode: (mode: ViewMode) => void

    tablesCount: number
    selectedTableIndex: number
    setSelectedTableIndex: (index: number) => void

    tableState: TableState
    setTableState: (updater: (prev: TableState) => TableState) => void

    scrollState: ScrollState
    rowsScrollRef: RefObject<ScrollBoxRenderable | null>
    headerScrollRef: RefObject<ScrollBoxRenderable | null>

    requestExit: () => void
}

function handleGlobalKey(key: KeyEvent, state: KeyHandlingState): boolean {
    if (key.ctrl && key.name === 'c') {
        state.requestExit()
        return true
    }

    if (key.name === 'q') {
        state.requestExit()
        return true
    }

    if (key.name === 'escape') {
        state.requestExit()
        return true
    }

    if (key.name === 'tab') {
        let next: FocusArea = 'sidebar'
        if (state.focusArea === 'sidebar') {
            next = 'rows'
        }

        if (state.focusArea === 'rows') {
            next = 'query'
        }

        if (state.focusArea === 'query') {
            next = 'sidebar'
        }

        state.setFocusArea(next)
        return true
    }

    if (state.focusArea === 'sidebar') {
        return handleSidebarKey(key, state)
    }

    if (state.focusArea === 'rows') {
        return handleRowsKey(key, state)
    }

    return false
}

function handleSidebarKey(key: KeyEvent, state: KeyHandlingState): boolean {
    if (state.tablesCount === 0) {
        return false
    }

    if (key.name === 'down' || key.name === 'j') {
        const next = clamp(state.selectedTableIndex + 1, 0, state.tablesCount - 1)
        state.setSelectedTableIndex(next)
        return true
    }

    if (key.name === 'up' || key.name === 'k') {
        const next = clamp(state.selectedTableIndex - 1, 0, state.tablesCount - 1)
        state.setSelectedTableIndex(next)
        return true
    }

    if (key.name === 'return') {
        state.setViewMode('table')
        state.setFocusArea('rows')
        return true
    }

    return false
}

function handleRowsKey(key: KeyEvent, state: KeyHandlingState): boolean {
    if (state.tableState.name.length === 0) {
        return false
    }

    if (key.name === 'j' || key.name === 'down') {
        if (!state.scrollState.overflowY) {
            return true
        }

        state.setTableState((prev) => ({
            ...prev,
            offset: prev.offset + 1,
        }))
        return true
    }

    if (key.name === 'k' || key.name === 'up') {
        if (!state.scrollState.overflowY) {
            return true
        }

        state.setTableState((prev) => ({
            ...prev,
            offset: Math.max(0, prev.offset - 1),
        }))
        return true
    }

    if (key.name === 'h') {
        if (!state.scrollState.overflowX) {
            return true
        }

        const step = Math.max(1, Math.floor(state.scrollState.viewportWidth / SCROLL_STEP_DIVISOR))
        const scrollbox = state.rowsScrollRef.current
        if (!scrollbox) {
            return true
        }

        const maxScrollLeft = Math.max(
            0,
            state.scrollState.tableContentWidth - state.scrollState.viewportWidth,
        )
        const nextScrollLeft = Math.max(0, scrollbox.scrollLeft - step)
        scrollbox.scrollLeft = Math.min(nextScrollLeft, maxScrollLeft)
        if (state.headerScrollRef.current) {
            state.headerScrollRef.current.scrollLeft = scrollbox.scrollLeft
        }
        return true
    }

    if (key.name === 'l') {
        if (!state.scrollState.overflowX) {
            return true
        }

        const step = Math.max(1, Math.floor(state.scrollState.viewportWidth / SCROLL_STEP_DIVISOR))
        const scrollbox = state.rowsScrollRef.current
        if (!scrollbox) {
            return true
        }

        const maxScrollLeft = Math.max(
            0,
            state.scrollState.tableContentWidth - state.scrollState.viewportWidth,
        )
        const nextScrollLeft = scrollbox.scrollLeft + step
        scrollbox.scrollLeft = Math.min(nextScrollLeft, maxScrollLeft)
        if (state.headerScrollRef.current) {
            state.headerScrollRef.current.scrollLeft = scrollbox.scrollLeft
        }
        return true
    }

    return false
}

function getRowsTitle(props: {
    tableState: TableState
    queryState: QueryState
    viewMode: ViewMode
}): string {
    if (props.viewMode === 'query') {
        const singleLine = props.queryState.sql.replaceAll('\n', ' ').trim()
        if (singleLine.length === 0) {
            return 'Query'
        }

        return truncateTitle(singleLine, 60)
    }

    if (props.tableState.name.length > 0) {
        return props.tableState.name
    }

    return 'Rows'
}

function truncateTitle(value: string, maxChars: number): string {
    if (value.length <= maxChars) {
        return value
    }

    const safeMax = Math.max(0, maxChars - 3)
    return `${value.slice(0, safeMax)}...`
}

function measureMessageWidth(value: string): number {
    if (value.length === 0) {
        return 0
    }

    let max = 0
    for (const line of value.split('\n')) {
        if (line.length > max) {
            max = line.length
        }
    }
    return max
}

function clamp(value: number, min: number, max: number): number {
    if (value < min) {
        return min
    }

    if (value > max) {
        return max
    }

    return value
}

const path = parseDatabasePathFromArgs(process.argv)

const renderer = await createCliRenderer({
    useMouse: true,
    exitOnCtrlC: true,
    useKittyKeyboard: {},
})

const root = createRoot(renderer)

let exitRequested = false

const requestExit = () => {
    if (exitRequested) {
        return
    }

    exitRequested = true
    root.unmount()
    renderer.destroy()
}

process.once('SIGINT', () => {
    requestExit()
})

process.once('SIGTERM', () => {
    requestExit()
})

root.render(<App dbPath={path} requestExit={requestExit} />)
