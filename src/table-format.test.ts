import { describe, expect, test } from 'bun:test'
import { computeTable } from './table-format'
import type { SqliteRow } from './db'

describe('computeTable', () => {
    test('renders header and rows', () => {
        const rows: SqliteRow[] = [
            { id: 1, name: 'Ava', active: true, note: null },
            { id: 2, name: 'Mateo', active: false, note: 'hello' },
        ]

        const out = computeTable({
            columns: ['id', 'name', 'active', 'note'],
            rows,
        })

        expect(out.header).toContain('id')
        expect(out.header).toContain('name')
        expect(out.body).toContain('Ava')
        expect(out.body).toContain('Mateo')
        expect(out.body).toContain('NULL')
    })

    test('returns width and rowCount metadata', () => {
        const rows: SqliteRow[] = [
            {
                id: 1,
                name: 'This is a very long name that should be truncated',
            },
        ]

        const out = computeTable({
            columns: ['id', 'name'],
            rows,
        })

        expect(out.width).toBeGreaterThan(0)
        expect(out.rowCount).toBe(1)
    })
})
