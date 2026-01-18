import { describe, expect, test } from "bun:test";
import { computeTable } from "../src/table-format";
import type { SqliteRow } from "../src/db";

describe("computeTable", () => {
    test("renders header and rows", () => {
        const rows: SqliteRow[] = [
            { id: 1, name: "Ava", active: true, note: null },
            { id: 2, name: "Mateo", active: false, note: "hello" },
        ];

        const out = computeTable({
            columns: ["id", "name", "active", "note"],
            rows,
            maxWidth: 200,
        });

        expect(out.header).toContain("id");
        expect(out.header).toContain("name");
        expect(out.body).toContain("Ava");
        expect(out.body).toContain("Mateo");
        expect(out.body).toContain("NULL");
    });

    test("truncates cells to fit maxWidth", () => {
        const rows: SqliteRow[] = [
            {
                id: 1,
                name: "This is a very long name that should be truncated",
            },
        ];

        const out = computeTable({
            columns: ["id", "name"],
            rows,
            maxWidth: 20,
        });

        expect(out.header.length).toBeLessThanOrEqual(20);
        const lines = out.body.split("\n");
        expect(lines[0]?.length ?? 0).toBeLessThanOrEqual(20);
        expect(out.body).toContain("...");
    });
});
