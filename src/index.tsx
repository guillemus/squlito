import { createCliRenderer, type KeyEvent, type ScrollBoxRenderable, type SelectOption, TextAttributes } from "@opentui/core";
import { createRoot, useKeyboard, useTerminalDimensions } from "@opentui/react";
import { useEffect, useMemo, useRef, useState } from "react";
import { type SqliteRow, type SqliteTable, getTableColumns, getTablePage, listUserTables, openDatabase, parseDatabasePathFromArgs } from "./db";
import { computeTable } from "./table-format";

type FocusArea = "sidebar" | "rows";

type TableState = {
    name: string;
    totalRows: number;
    offset: number;
    limit: number;
    rows: SqliteRow[];
    columns: string[];
    error: string | null;
};

function App(props: { dbPath: string; requestExit: () => void }) {
    const dims = useTerminalDimensions();

    const [focusArea, setFocusArea] = useState<FocusArea>("sidebar");

    const [tables, setTables] = useState<SqliteTable[]>([]);
    const [selectedTableIndex, setSelectedTableIndex] = useState(0);

    const rowsScrollRef = useRef<ScrollBoxRenderable>(null);

    const [tableState, setTableState] = useState<TableState>({
        name: "",
        totalRows: 0,
        offset: 0,
        limit: 25,
        rows: [],
        columns: [],
        error: null,
    });

    useEffect(() => {
        const db = openDatabase(props.dbPath);

        try {
            const nextTables = listUserTables(db);
            setTables(nextTables);

            if (nextTables.length > 0) {
                const firstName = nextTables[0]?.name ?? "";
                    if (firstName.length > 0) {
                        const page = getTablePage(db, firstName, 25, 0);

                        const columns: string[] = [];
                        for (const c of getTableColumns(db, firstName)) {
                            columns.push(c.name);
                        }

                        setSelectedTableIndex(0);
                        setTableState({
                            name: firstName,
                            totalRows: page.totalRows,
                            offset: page.offset,
                            limit: page.limit,
                            rows: page.rows,
                            columns,
                            error: null,
                        });
                    }

            }
        } catch (err) {
            const message = err instanceof Error ? err.message : String(err);
            setTableState((prev) => ({ ...prev, error: message }));
        } finally {
            db.close();
        }
    }, [props.dbPath]);

    useEffect(() => {
        const selectedName = tables[selectedTableIndex]?.name;
        if (!selectedName) {
            return;
        }

        if (tableState.name === selectedName) {
            return;
        }

        setTableState((prev) => ({
            ...prev,
            name: selectedName,
            offset: 0,
            error: null,
        }));

        if (rowsScrollRef.current) {
            rowsScrollRef.current.scrollTop = 0;
        }
    }, [selectedTableIndex, tables, tableState.name]);

    useEffect(() => {
        if (tableState.name.length === 0) {
            return;
        }

        const db = openDatabase(props.dbPath);

        try {
            const page = getTablePage(db, tableState.name, tableState.limit, tableState.offset);

            const columns: string[] = [];
            for (const c of getTableColumns(db, tableState.name)) {
                columns.push(c.name);
            }

            setTableState((prev) => ({
                ...prev,
                totalRows: page.totalRows,
                offset: page.offset,
                limit: page.limit,
                rows: page.rows,
                columns,
                error: null,
            }));

            const rowIndexWithinPage = page.offset - tableState.offset;
            if (rowsScrollRef.current && rowIndexWithinPage !== 0) {
                const nextScrollTop = Math.max(0, rowIndexWithinPage);
                rowsScrollRef.current.scrollTop = nextScrollTop;
            }
        } catch (err) {
            const message = err instanceof Error ? err.message : String(err);
            setTableState((prev) => ({
                ...prev,
                rows: [],
                columns: [],
                totalRows: 0,
                error: message,
            }));
        } finally {
            db.close();
        }
    }, [props.dbPath, tableState.name, tableState.limit, tableState.offset]);

    useKeyboard((key) => {
        if (key.eventType === "release") {
            return;
        }

        if (key.name === "h") {
            setTableState((prev) => {
                const nextError = prev.error ? null : cliHelp.trim();
                return { ...prev, error: nextError };
            });
            key.preventDefault();
            key.stopPropagation();
            return;
        }

        const handled = handleGlobalKey(key, {
            focusArea,
            setFocusArea,
            tablesCount: tables.length,
            selectedTableIndex,
            setSelectedTableIndex,
            tableState,
            setTableState,
            requestExit: props.requestExit,
        });

        if (handled) {
            key.preventDefault();
            key.stopPropagation();
        }
    });

    const sidebarWidth = clamp(Math.floor(dims.width * 0.28), 22, 40);
    const mainWidth = Math.max(20, dims.width - sidebarWidth);


    const tableView = useMemo(() => {
        if (tableState.error) {
            return {
                header: "",
                body: tableState.error,
            };
        }

        if (tableState.name.length === 0) {
            return {
                header: "",
                body: "No table selected",
            };
        }

        if (tableState.columns.length === 0) {
            return {
                header: "",
                body: "(empty)",
            };
        }

        return computeTable({
            columns: tableState.columns,
            rows: tableState.rows,
            maxWidth: Math.max(20, mainWidth - 4),
        });
    }, [mainWidth, tableState.columns, tableState.error, tableState.name, tableState.rows]);

    const sidebarFocused = focusArea === "sidebar";
    const rowsFocused = focusArea === "rows";

    const pageEnd = Math.min(tableState.totalRows, tableState.offset + tableState.limit);
    const showStart = tableState.totalRows === 0 ? 0 : tableState.offset + 1;

    const cliHelp = `
usage:
  bun run src/index.tsx [path/to/db]

keys:
  tab: switch focus
  up/down or j/k: move
  enter: focus rows
  pgup/pgdn: page
  left/right: limit
  q/esc/ctrl+c: quit
`;

    return (
        <box flexDirection="row" width="100%" height="100%" backgroundColor="#0b1020">
            <box
                title={"Tables"}
                border={true}
                borderStyle="single"
                borderColor="#2a355a"
                focusedBorderColor={sidebarFocused ? "#7bdff2" : "#2a355a"}
                width={sidebarWidth}
                height="100%"
                flexDirection="column"
                padding={1}
            >
                <text attributes={TextAttributes.DIM} fg="#9aa4c5">{props.dbPath}</text>
                <text attributes={TextAttributes.DIM} fg="#9aa4c5">{sidebarFocused ? "[Tab] rows" : "[Tab] tables"}</text>
                <box height={1} />

                <select
                    focused={sidebarFocused}
                    options={useMemo((): SelectOption[] => {
                        const options: SelectOption[] = [];
                        for (const t of tables) {
                            options.push({ name: t.name, description: "", value: t.name });
                        }
                        return options;
                    }, [tables])}
                    selectedIndex={selectedTableIndex}
                    wrapSelection={true}
                    showDescription={false}
                    showScrollIndicator={true}
                    style={{ flexGrow: 1 }}
                    onChange={(index) => {
                        setSelectedTableIndex(index);
                    }}
                    onSelect={() => {
                        setFocusArea("rows");
                    }}
                />

                <box height={1} />
                <text attributes={TextAttributes.DIM} fg="#9aa4c5">
                    {"↑↓ select  Enter open"}
                </text>
            </box>

            <box
                title={tableState.name.length > 0 ? tableState.name : "Rows"}
                border={true}
                borderStyle="single"
                borderColor="#2a355a"
                focusedBorderColor={rowsFocused ? "#f7c948" : "#2a355a"}
                flexGrow={1}
                height="100%"
                flexDirection="column"
                padding={1}
            >
                <box flexDirection="row" justifyContent="space-between" width="100%">
                    <text attributes={TextAttributes.DIM} fg="#9aa4c5">
                        {`Rows ${tableState.totalRows}  Showing ${showStart}-${pageEnd}  Limit ${tableState.limit}`}
                    </text>
                    <text attributes={TextAttributes.DIM} fg="#9aa4c5">
                        {rowsFocused ? "[Tab] tables" : "[Tab] rows"}
                    </text>
                </box>

                <box height={1} />

                {tableView.header.length > 0 ? (
                    <box backgroundColor="#121a33" paddingLeft={1} paddingRight={1} height={1} width="100%">
                        <text fg="#d4defc" attributes={TextAttributes.BOLD}>
                            {tableView.header}
                        </text>
                    </box>
                ) : null}

                <scrollbox
                    ref={rowsScrollRef}
                    focused={rowsFocused}
                    style={{ flexGrow: 1, scrollY: true, viewportCulling: true }}
                    viewportOptions={{ backgroundColor: "#0b1020" }}
                    onMouseScroll={(event) => {
                        if (!rowsFocused) {
                            return;
                        }

                        if (!event.scroll) {
                            return;
                        }

                        const direction = event.scroll.direction;
                        const delta = Math.max(1, Math.floor(event.scroll.delta));

                        if (direction === "down") {
                            setTableState((prev) => ({ ...prev, offset: prev.offset + delta }));
                            return;
                        }

                        if (direction === "up") {
                            setTableState((prev) => ({ ...prev, offset: Math.max(0, prev.offset - delta) }));
                            return;
                        }
                    }}
                >
                    <box flexDirection="column" width="100%" paddingLeft={1} paddingRight={1}>
                        <text fg="#cbd5f0">{tableView.body}</text>
                    </box>
                </scrollbox>

                <box height={1} />

                <box flexDirection="row" justifyContent="space-between" width="100%">
                    <text attributes={TextAttributes.DIM} fg="#9aa4c5">
                        {"PgUp/PgDn page  Left/Right limit  j/k scroll  h help"}
                    </text>
                    <text attributes={TextAttributes.DIM} fg="#9aa4c5">
                        {"q quit"}
                    </text>
                </box>
            </box>
        </box>
    );
}

type KeyHandlingState = {
    focusArea: FocusArea;
    setFocusArea: (area: FocusArea) => void;

    tablesCount: number;
    selectedTableIndex: number;
    setSelectedTableIndex: (index: number) => void;

    tableState: TableState;
    setTableState: (updater: (prev: TableState) => TableState) => void;

    requestExit: () => void;
};

function handleGlobalKey(key: KeyEvent, state: KeyHandlingState): boolean {
    if (key.ctrl && key.name === "c") {
        state.requestExit();
        return true;
    }

    if (key.name === "q") {
        state.requestExit();
        return true;
    }

    if (key.name === "escape") {
        state.requestExit();
        return true;
    }

    if (key.name === "tab") {
        const next: FocusArea = state.focusArea === "sidebar" ? "rows" : "sidebar";
        state.setFocusArea(next);
        return true;
    }

    if (state.focusArea === "sidebar") {
        return handleSidebarKey(key, state);
    }

    return handleRowsKey(key, state);
}

function handleSidebarKey(key: KeyEvent, state: KeyHandlingState): boolean {
    if (state.tablesCount === 0) {
        return false;
    }

    if (key.name === "down" || key.name === "j") {
        const next = clamp(state.selectedTableIndex + 1, 0, state.tablesCount - 1);
        state.setSelectedTableIndex(next);
        return true;
    }

    if (key.name === "up" || key.name === "k") {
        const next = clamp(state.selectedTableIndex - 1, 0, state.tablesCount - 1);
        state.setSelectedTableIndex(next);
        return true;
    }

    if (key.name === "return") {
        state.setFocusArea("rows");
        return true;
    }

    return false;
}

function handleRowsKey(key: KeyEvent, state: KeyHandlingState): boolean {
    if (state.tableState.name.length === 0) {
        return false;
    }

    const pageStep = state.tableState.limit;

    if (key.name === "pageup") {
        state.setTableState((prev) => ({
            ...prev,
            offset: Math.max(0, prev.offset - pageStep),
        }));
        return true;
    }

    if (key.name === "pagedown") {
        state.setTableState((prev) => ({
            ...prev,
            offset: prev.offset + pageStep,
        }));
        return true;
    }

    if (key.name === "j" || key.name === "down") {
        state.setTableState((prev) => ({
            ...prev,
            offset: prev.offset + 1,
        }));
        return true;
    }

    if (key.name === "k" || key.name === "up") {
        state.setTableState((prev) => ({
            ...prev,
            offset: Math.max(0, prev.offset - 1),
        }));
        return true;
    }

    if (key.name === "left") {
        state.setTableState((prev) => ({
            ...prev,
            limit: clamp(prev.limit - 5, 5, 200),
            offset: Math.max(0, prev.offset),
        }));
        return true;
    }

    if (key.name === "right") {
        state.setTableState((prev) => ({
            ...prev,
            limit: clamp(prev.limit + 5, 5, 200),
            offset: Math.max(0, prev.offset),
        }));
        return true;
    }

    return false;
}

function clamp(value: number, min: number, max: number): number {
    if (value < min) {
        return min;
    }

    if (value > max) {
        return max;
    }

    return value;
}

const path = parseDatabasePathFromArgs(process.argv);

const renderer = await createCliRenderer({
    useMouse: true,
    exitOnCtrlC: true,
});

const root = createRoot(renderer);

let exitRequested = false;

const requestExit = () => {
    if (exitRequested) {
        return;
    }

    exitRequested = true;
    root.unmount();
    renderer.destroy();
};

process.once("SIGINT", () => {
    requestExit();
});

process.once("SIGTERM", () => {
    requestExit();
});

root.render(<App dbPath={path} requestExit={requestExit} />);
