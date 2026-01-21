package app

import "math"

func calculateLayout(maxX int, maxY int) layoutMetrics {
    headerHeight := rowsHeaderHeight
    queryHeight := queryBoxHeight
    availableHeight := maxY - statusHeight

    minTotal := headerHeight + minimumRowsHeight + queryHeight
    if availableHeight < minTotal {
        queryHeight = availableHeight - headerHeight - minimumRowsHeight
    }

    if queryHeight < 3 {
        queryHeight = 3
    }

    rowsHeight := availableHeight - headerHeight - queryHeight
    if rowsHeight < minimumRowsHeight {
        rowsHeight = minimumRowsHeight
        queryHeight = availableHeight - headerHeight - rowsHeight
    }

    sidebarWidth := int(math.Round(float64(maxX) * sidebarWidthRatio))
    sidebarWidth = clampInt(sidebarWidth, sidebarWidthMin, sidebarWidthMax)

	maxSidebar := maxX - minimumMainWidth
	maxSidebar = max(maxSidebar, sidebarWidthMin)

    if sidebarWidth > maxSidebar {
        sidebarWidth = maxSidebar
    }

    if sidebarWidth < 10 {
        sidebarWidth = 10
    }

    mainWidth := maxX - sidebarWidth

    return layoutMetrics{
        sidebarWidth: sidebarWidth,
        mainWidth:    mainWidth,
        headerHeight: headerHeight,
        rowsHeight:   rowsHeight,
        queryHeight:  queryHeight,
        statusHeight: statusHeight,
    }
}
