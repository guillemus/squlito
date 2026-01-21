package tableformat

import (
	"fmt"
	"strconv"
	"strings"

	"squlito/internal/db"
)

type TableRender struct {
	Header   string
	Body     string
	Width    int
	RowCount int
	ColumnWidths  []int
	SeparatorWidth int
}

type ComputeTableConfig struct {
    Columns []string
    Rows    []db.SqliteRow
    MaxRows int
}

func ComputeTable(config ComputeTableConfig) TableRender {
    visibleRows := config.Rows
    if config.MaxRows > 0 {
        max := clampInt(config.MaxRows, 1, 500)
        if len(config.Rows) > max {
            visibleRows = config.Rows[:max]
        }
    }

    widths := []int{}
    for _, col := range config.Columns {
        widths = append(widths, stringWidth(col))
    }

    for _, row := range visibleRows {
        for i := 0; i < len(config.Columns); i += 1 {
            key := config.Columns[i]
            value, ok := row[key]
            if !ok {
                continue
            }

            normalized := formatCell(value)
            w := stringWidth(normalized)
            prev := widths[i]
            if w > prev {
                widths[i] = w
            }
        }
    }

    minColWidth := 4
    for i := 0; i < len(widths); i += 1 {
        widths[i] = clampInt(widths[i], minColWidth, 50)
    }

	separatorWidth := columnSeparatorWidth
	totalSeparators := max(0, len(config.Columns)-1)

    totalWidth := 0
    for _, w := range widths {
        totalWidth += w
    }
    totalWidth += totalSeparators * separatorWidth

    headerCells := []string{}
    for i := 0; i < len(config.Columns); i += 1 {
        key := config.Columns[i]
        if key == "" {
            continue
        }

        colWidth := widths[i]
        headerCells = append(headerCells, padRight(truncateString(key, colWidth), colWidth))
    }

    header := ""
	if len(headerCells) > 0 {
		header = joinCells(headerCells)
	}

    bodyLines := []string{}
    for _, row := range visibleRows {
        cells := []string{}

        for i := 0; i < len(config.Columns); i += 1 {
            key := config.Columns[i]
            value := row[key]
            raw := formatCell(value)
            clipped := truncateString(raw, widths[i])
            cell := padRight(clipped, widths[i])
            cells = append(cells, cell)
        }

        bodyLines = append(bodyLines, joinCells(cells))
    }

    body := ""
    if len(bodyLines) > 0 {
        body = joinLines(bodyLines)
    }

	return TableRender{
		Header:   header,
		Body:     body,
		Width:    totalWidth,
		RowCount: len(visibleRows),
		ColumnWidths:  widths,
		SeparatorWidth: separatorWidth,
	}
}

func FormatCell(value db.SqliteValue) string {
	return formatCell(value)
}

func formatCell(value db.SqliteValue) string {
    if value == nil {
        return "NULL"
    }

    switch typed := value.(type) {
    case string:
        return typed
    case []byte:
        return fmt.Sprintf("BLOB(%d)", len(typed))
    case int:
        return strconv.Itoa(typed)
    case int32:
        return strconv.FormatInt(int64(typed), 10)
    case int64:
        return strconv.FormatInt(typed, 10)
    case float32:
        return strconv.FormatFloat(float64(typed), 'f', -1, 32)
    case float64:
        return strconv.FormatFloat(typed, 'f', -1, 64)
    case bool:
        if typed {
            return "1"
        }
        return "0"
    default:
        return fmt.Sprint(value)
    }
}

func truncateString(value string, maxChars int) string {
    if maxChars <= 0 {
        return ""
    }

    if len(value) <= maxChars {
        return value
    }

    if maxChars <= 3 {
        return value[:maxChars]
    }

    return value[:maxChars-3] + "..."
}

func padRight(value string, width int) string {
    w := stringWidth(value)
    if w >= width {
        return value
    }

    return value + spaces(width-w)
}

func stringWidth(value string) int {
    return len(value)
}

func joinCells(cells []string) string {
	if len(cells) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString(cells[0])
	for i := 1; i < len(cells); i += 1 {
		builder.WriteString(columnSeparator)
		builder.WriteString(cells[i])
	}
	return builder.String()
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString(lines[0])
	for i := 1; i < len(lines); i += 1 {
		builder.WriteString("\n")
		builder.WriteString(lines[i])
	}
	return builder.String()
}

func spaces(count int) string {
    if count <= 0 {
        return ""
    }

    return fmt.Sprintf("%*s", count, "")
}

const columnSeparator = " | "

const columnSeparatorWidth = 3

func clampInt(value int, min int, max int) int {
    if value < min {
        return min
    }

    if value > max {
        return max
    }

    return value
}
