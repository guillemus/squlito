package app

import (
	"strings"

	"squlito/internal/db"
)

const (
	bufferSize        = 200
	scrollStepDivisor = 5
	queryRowCap       = 10000
	queryBoxHeight    = 7
	historyLimit      = 200
	sidebarWidthMin   = 22
	sidebarWidthMax   = 40
	sidebarWidthRatio = 0.28
	rowsHeaderHeight  = 3
	statusHeight      = 2
	minimumRowsHeight = 3
	minimumMainWidth  = 20
	titleMaxChars     = 60
)

type FocusArea string

const (
	focusSidebar FocusArea = "sidebar"
	focusRows    FocusArea = "rows"
	focusQuery   FocusArea = "query"
	focusModal   FocusArea = "modal"
)

type ViewMode string

const (
	viewTable ViewMode = "table"
	viewQuery ViewMode = "query"
)

type TableState struct {
	Name        string
	TotalRows   int
	Offset      int
	BufferStart int
	BufferSize  int
	Rows        []db.SqliteRow
	Columns     []string
	Error       string
}

type QueryState struct {
	SQL       string
	AllRows   []db.SqliteRow
	Columns   []string
	Error     string
	Running   bool
	Truncated bool
	Offset    int
}

type QueryHistoryEntry struct {
	ID        int64
	SQL       string
	CreatedAt string
}

type ScrollState struct {
	OverflowY         bool
	OverflowX         bool
	ViewportRows      int
	ViewportWidth     int
	TableContentWidth int
}

type layoutMetrics struct {
	sidebarWidth int
	mainWidth    int
	headerHeight int
	rowsHeight   int
	queryHeight  int
	statusHeight int
}

func clampInt(value int, min int, max int) int {
	if value < min {
		return min
	}

	if value > max {
		return max
	}

	return value
}

func truncateTitle(value string) string {
	trimmed := strings.ReplaceAll(value, "\n", " ")
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return "Query"
	}

	if len(trimmed) <= titleMaxChars {
		return trimmed
	}

	safeMax := max(0, titleMaxChars-3)
	return trimmed[:safeMax] + "..."
}

func measureMessageWidth(value string) int {
	if value == "" {
		return 0
	}

	max := 0
	for part := range strings.SplitSeq(value, "\n") {
		if len(part) > max {
			max = len(part)
		}
	}

	return max
}

func truncateLine(value string, maxChars int) string {
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
