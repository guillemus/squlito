import { describe, expect, test } from 'bun:test'
import { Database } from 'bun:sqlite'
import { getTableColumns, getTablePage, listUserTables, parseDatabasePathFromArgs } from './db'

function createDb(): Database {
    const db = new Database(':memory:')
    db.exec('PRAGMA foreign_keys = ON')

    return db
}

describe('parseDatabasePathFromArgs', () => {
    test('uses seed db by default', () => {
        const path = parseDatabasePathFromArgs(['bun', 'app'])
        expect(path).toBe('data/seed.db')
    })

    test('uses first non-flag arg', () => {
        const path = parseDatabasePathFromArgs(['bun', 'app', 'example.db'])
        expect(path).toBe('example.db')
    })

    test('skips flags', () => {
        const path = parseDatabasePathFromArgs(['bun', 'app', '--verbose', 'file.db'])
        expect(path).toBe('file.db')
    })

    test('ignores args after first positional', () => {
        const path = parseDatabasePathFromArgs(['bun', 'app', 'first.db', 'second.db'])
        expect(path).toBe('first.db')
    })

    test('skips literal -- separator', () => {
        const path = parseDatabasePathFromArgs(['bun', 'app', '--', 'db.sqlite'])
        expect(path).toBe('db.sqlite')
    })
})

describe('listUserTables', () => {
    test('returns user tables and excludes sqlite internal tables', () => {
        const db = createDb()
        db.exec('CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)')
        db.exec('CREATE TABLE posts (id INTEGER PRIMARY KEY, title TEXT)')

        const tables = listUserTables(db).map((t) => t.name)
        expect(tables).toEqual(['posts', 'users'])

        db.close()
    })
})

describe('getTableColumns', () => {
    test('returns column names in table order', () => {
        const db = createDb()
        db.exec('CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, active INTEGER)')

        const cols = getTableColumns(db, 'users').map((c) => c.name)
        expect(cols).toEqual(['id', 'name', 'active'])

        db.close()
    })
})

describe('getTablePage', () => {
    test('paginates with limit and offset', () => {
        const db = createDb()
        db.exec('CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)')

        const insert = db.prepare<unknown, [number, string]>(
            'INSERT INTO users (id, name) VALUES (?, ?)',
        )
        for (let i = 1; i <= 10; i += 1) {
            insert.run(i, `User ${i}`)
        }

        const page = getTablePage(db, 'users', 3, 4)
        expect(page.totalRows).toBe(10)
        expect(page.offset).toBe(4)
        expect(page.limit).toBe(3)

        const ids = page.rows.map((r) => r.id)
        expect(ids).toEqual([5, 6, 7])

        db.close()
    })

    test('clamps offset when it exceeds total rows', () => {
        const db = createDb()
        db.exec('CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)')

        const insert = db.prepare<unknown, [number, string]>(
            'INSERT INTO users (id, name) VALUES (?, ?)',
        )
        for (let i = 1; i <= 10; i += 1) {
            insert.run(i, `User ${i}`)
        }

        const page = getTablePage(db, 'users', 4, 999)
        expect(page.totalRows).toBe(10)
        expect(page.offset).toBe(6)

        const ids = page.rows.map((r) => r.id)
        expect(ids).toEqual([7, 8, 9, 10])

        db.close()
    })
})
