import { Database } from 'bun:sqlite'

export type SqliteValue = string | number | bigint | boolean | null | Uint8Array
export type SqliteRow = Record<string, SqliteValue>

export type SqliteTable = {
    name: string
}

export type SqliteColumn = {
    cid: number
    name: string
    type: string
    notnull: 0 | 1
    dflt_value: string | null
    pk: 0 | 1
}

export type OpenDatabaseResult = {
    db: Database
    path: string
}

export function parseDatabasePathFromArgs(argv: string[]): string {
    const args = argv.slice(2)

    let path = 'data/seed.db'

    for (const arg of args) {
        if (arg === '--') {
            continue
        }

        if (arg.startsWith('-')) {
            continue
        }

        path = arg
        break
    }

    return path
}

export function openDatabase(dbPath: string): Database {
    const db = new Database(dbPath, { readonly: true })
    db.exec('PRAGMA foreign_keys = ON')
    return db
}

export function openDatabaseFromArgs(argv: string[]): OpenDatabaseResult {
    const path = parseDatabasePathFromArgs(argv)
    const db = openDatabase(path)

    return { db, path }
}

export function listUserTables(db: Database): SqliteTable[] {
    const stmt = db.query<SqliteTable, []>(
        "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name",
    )

    return stmt.all()
}

export function getTableColumns(db: Database, tableName: string): SqliteColumn[] {
    const sql = `PRAGMA table_info(${quoteIdentifier(tableName)})`
    const stmt = db.query<SqliteColumn, []>(sql)

    return stmt.all()
}

export type TablePage = {
    totalRows: number
    offset: number
    rows: SqliteRow[]
}

export function getTablePage(
    db: Database,
    tableName: string,
    limit: number,
    offset: number,
): TablePage {
    const safeLimit = clamp(limit, 1, 500)
    const safeOffset = Math.max(0, offset)

    const countSql = `SELECT COUNT(*) AS count FROM ${quoteIdentifier(tableName)}`
    const countStmt = db.query<{ count: number }, []>(countSql)
    const countRow = countStmt.get()

    const totalRows = countRow?.count ?? 0

    const pageSql = `SELECT * FROM ${quoteIdentifier(tableName)} LIMIT ? OFFSET ?`
    const pageStmt = db.query<SqliteRow, [number, number]>(pageSql)

    const rows = pageStmt.all(safeLimit, safeOffset)

    return { totalRows, offset: safeOffset, rows }
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

function quoteIdentifier(identifier: string): string {
    const escaped = identifier.replaceAll('"', '""')
    return `"${escaped}"`
}
