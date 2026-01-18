import type { SqliteRow } from './db'

export type TableRender = {
    header: string
    body: string
    width: number
    rowCount: number
}

type ComputeTableConfig = {
    columns: string[]
    rows: SqliteRow[]
    maxRows?: number
}

export function computeTable(config: ComputeTableConfig): TableRender {
    const safeMaxRows = config.maxRows

    let visibleRows = config.rows
    if (safeMaxRows !== undefined) {
        const max = clamp(safeMaxRows, 1, 500)
        visibleRows = config.rows.slice(0, max)
    }

    const widths: number[] = []

    for (const col of config.columns) {
        widths.push(stringWidth(col))
    }

    for (const row of visibleRows) {
        for (let i = 0; i < config.columns.length; i += 1) {
            const key = config.columns[i]
            if (key === undefined) {
                continue
            }

            const value = row[key]
            const normalizedValue = value ?? null
            const str = formatCell(normalizedValue)
            const w = stringWidth(str)

            const prev = widths[i] ?? 0
            if (w > prev) {
                widths[i] = w
            }
        }
    }

    // Clamp column widths to fit maxWidth.
    const minColWidth = 4
    for (let i = 0; i < widths.length; i += 1) {
        const w = widths[i] ?? minColWidth
        widths[i] = clamp(w, minColWidth, 60)
    }

    const separatorWidth = 3 // " | "
    const totalSeparators = Math.max(0, config.columns.length - 1)

    let totalWidth = 0
    for (const w of widths) {
        totalWidth += w
    }
    totalWidth += totalSeparators * separatorWidth

    const headerCells: string[] = []
    for (let i = 0; i < config.columns.length; i += 1) {
        const key = config.columns[i]
        if (key === undefined) {
            continue
        }

        headerCells.push(
            padRight(truncateString(key, widths[i] ?? minColWidth), widths[i] ?? minColWidth),
        )
    }

    const header = headerCells.join(' | ')

    const bodyLines: string[] = []
    for (const row of visibleRows) {
        const cells: string[] = []

        for (let i = 0; i < config.columns.length; i += 1) {
            const key = config.columns[i]
            if (key === undefined) {
                continue
            }

            const value = row[key]
            const normalizedValue = value ?? null
            const raw = formatCell(normalizedValue)
            const clipped = truncateString(raw, widths[i] ?? minColWidth)
            const cell = padRight(clipped, widths[i] ?? minColWidth)

            cells.push(cell)
        }

        bodyLines.push(cells.join(' | '))
    }

    return {
        header,
        body: bodyLines.join('\n'),
        width: totalWidth,
        rowCount: visibleRows.length,
    }
}

function formatCell(value: SqliteRow[string]): string {
    if (value === null) {
        return 'NULL'
    }

    if (typeof value === 'string') {
        return value
    }

    if (typeof value === 'number') {
        return String(value)
    }

    if (typeof value === 'bigint') {
        return String(value)
    }

    if (typeof value === 'boolean') {
        return value ? '1' : '0'
    }

    return `BLOB(${value.byteLength})`
}

function truncateString(value: string, maxChars: number): string {
    if (maxChars <= 0) {
        return ''
    }

    if (value.length <= maxChars) {
        return value
    }

    if (maxChars <= 3) {
        return value.slice(0, maxChars)
    }

    return value.slice(0, maxChars - 3) + '...'
}

function padRight(value: string, width: number): string {
    const w = stringWidth(value)
    if (w >= width) {
        return value
    }

    return value + ' '.repeat(width - w)
}

// Terminal monospace width approximation (ASCII-focused). This keeps us fast and predictable.
function stringWidth(value: string): number {
    return value.length
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
