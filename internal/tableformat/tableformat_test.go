package tableformat

import (
	"strings"
	"testing"

	"squlito/internal/db"
)

func TestComputeTable_RendersHeaderAndRows(t *testing.T) {
	rows := []db.SqliteRow{
		{"id": int64(1), "name": "Ava", "active": true, "note": nil},
		{"id": int64(2), "name": "Mateo", "active": false, "note": "hello"},
	}

	out := ComputeTable(ComputeTableConfig{
		Columns: []string{"id", "name", "active", "note"},
		Rows:    rows,
		MaxRows: 0,
	})

	if out.Header == "" {
		t.Fatalf("expected header output")
	}

	if out.Body == "" {
		t.Fatalf("expected body output")
	}

	if out.RowCount != 2 {
		t.Fatalf("expected 2 rows, got %d", out.RowCount)
	}
}

func TestComputeTable_Metadata(t *testing.T) {
	rows := []db.SqliteRow{
		{"id": int64(1), "name": "This is a very long name that should be truncated"},
	}

	out := ComputeTable(ComputeTableConfig{
		Columns: []string{"id", "name"},
		Rows:    rows,
		MaxRows: 0,
	})

	if out.Width <= 0 {
		t.Fatalf("expected width > 0")
	}

	if out.RowCount != 1 {
		t.Fatalf("expected row count 1, got %d", out.RowCount)
	}
}

func TestComputeTable_TruncatesCells(t *testing.T) {
	rows := []db.SqliteRow{
		{"id": int64(1), "note": "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz"},
	}

	out := ComputeTable(ComputeTableConfig{
		Columns: []string{"id", "note"},
		Rows:    rows,
		MaxRows: 0,
	})

	if len(out.Header) == 0 {
		t.Fatalf("expected header")
	}

	if len(out.Body) == 0 {
		t.Fatalf("expected body")
	}

	if !strings.Contains(out.Body, "...") {
		t.Fatalf("expected truncated body")
	}
}
