package view

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"

	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/metrics"
	"github.com/clawscli/claws/internal/render"
)

const (
	markColWidth    = 3
	profileColWidth = 16
	accountColWidth = 14
	regionColWidth  = 14
	columnGap       = 1
)

func (r *ResourceBrowser) Cursor() int {
	return r.tc.Cursor()
}

func (r *ResourceBrowser) SetCursor(n int) {
	r.tc.SetCursor(n, len(r.filtered))
}

func (r *ResourceBrowser) buildTable() {
	if r.renderer == nil {
		r.tableContent = ""
		return
	}

	r.tc.SetCursor(r.tc.Cursor(), len(r.filtered))

	cols := r.renderer.Columns()
	if len(cols) == 0 {
		r.tableContent = ""
		return
	}

	effectiveMetricsEnabled := r.metricsEnabled && r.getMetricSpec() != nil
	isMultiProfile := config.Global().IsMultiProfile()
	isMultiRegion := config.Global().IsMultiRegion()

	numCols := len(cols) + 1
	if isMultiProfile {
		numCols += 3
	} else if isMultiRegion {
		numCols++
	}
	if effectiveMetricsEnabled {
		numCols++
	}

	headers := make([]string, numCols)
	headers[0] = ""
	colIdx := 1
	for i, col := range cols {
		headers[colIdx] = col.Name + r.getSortIndicator(i)
		colIdx++
	}

	if isMultiProfile {
		headers[colIdx] = "PROFILE"
		colIdx++
		headers[colIdx] = "ACCOUNT"
		colIdx++
		headers[colIdx] = "REGION"
		colIdx++
	} else if isMultiRegion {
		headers[colIdx] = "REGION"
		colIdx++
	}

	if effectiveMetricsEnabled {
		spec := r.getMetricSpec()
		header := "METRICS"
		if spec != nil {
			header = spec.ColumnHeader
		}
		headers[colIdx] = header
	}

	var summaryFields []render.SummaryField
	cursor := r.tc.Cursor()
	if len(r.filtered) > 0 && cursor >= 0 && cursor < len(r.filtered) {
		summaryFields = r.renderer.RenderSummary(dao.UnwrapResource(r.filtered[cursor]))
	}
	headerStr := r.headerPanel.Render(r.service, r.resourceType, summaryFields)
	headerHeight := r.headerPanel.Height(headerStr)

	tableHeight := r.height - headerHeight - 1
	if tableHeight < 1 {
		tableHeight = 1
	}
	r.tc.SetTableHeight(tableHeight)

	rows := make([][]string, len(r.filtered))
	firstColContentWidth := lipgloss.Width(headers[1])
	for i, res := range r.filtered {
		rows[i] = r.renderer.RenderRow(dao.UnwrapResource(res), cols)

		if len(rows[i]) > 0 {
			firstColContentWidth = max(firstColContentWidth, lipgloss.Width(rows[i][0]))
		}
	}

	widths := r.calculateColumnWidths(cols, isMultiProfile, isMultiRegion, effectiveMetricsEnabled, numCols, firstColContentWidth)

	t := table.New().
		Headers(headers...).
		Width(r.width).
		Height(tableHeight).
		Wrap(false).
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderColumn(false).
		BorderHeader(true).
		BorderStyle(TableBorderStyle()).
		StyleFunc(NewTableStyleFunc(widths, cursor))

	for i, res := range r.filtered {
		row := rows[i]
		mark := " "
		if r.markedResource != nil && r.markedResource.GetID() == res.GetID() {
			mark = "◆"
		}

		fullRow := make([]string, numCols)
		fullRow[0] = mark
		copy(fullRow[1:], row)

		rowIdx := len(cols) + 1
		if isMultiProfile {
			profileID := dao.GetResourceProfile(res)
			fullRow[rowIdx] = config.ProfileSelectionFromID(profileID).DisplayName()
			rowIdx++
			fullRow[rowIdx] = dao.GetResourceAccountID(res)
			rowIdx++
			fullRow[rowIdx] = dao.GetResourceRegion(res)
			rowIdx++
		} else if isMultiRegion {
			fullRow[rowIdx] = dao.GetResourceRegion(res)
			rowIdx++
		}
		if effectiveMetricsEnabled && r.metricsData != nil {
			unit := ""
			if r.metricsData.Spec != nil {
				unit = r.metricsData.Spec.Unit
			}
			fullRow[rowIdx] = metrics.RenderSparkline(r.metricsData.Get(res.GetID()), unit)
		} else if effectiveMetricsEnabled {
			fullRow[rowIdx] = metrics.RenderSparkline(nil, "")
		}

		t = t.Row(fullRow...)
	}

	if r.tc.ScrollOffset() > 0 {
		t = t.YOffset(r.tc.ScrollOffset())
	}

	r.tableContent = t.String()
}

func (r *ResourceBrowser) calculateColumnWidths(cols []render.Column, isMultiProfile, isMultiRegion, hasMetrics bool, numCols, firstColContentWidth int) []int {
	metricsColWidth := metrics.ColumnWidth

	totalColWidth := markColWidth
	for _, col := range cols {
		totalColWidth += col.Width
	}
	if isMultiProfile {
		totalColWidth += profileColWidth + accountColWidth + regionColWidth
	} else if isMultiRegion {
		totalColWidth += regionColWidth
	}
	if hasMetrics {
		totalColWidth += metricsColWidth
	}

	extraWidth := r.width - totalColWidth
	if extraWidth < 0 {
		extraWidth = 0
	}

	firstColGrowth := min(extraWidth, max(0, firstColContentWidth+columnGap-cols[0].Width))
	extraWidth -= firstColGrowth

	hasTrailingCols := isMultiProfile || isMultiRegion || hasMetrics
	widths := make([]int, numCols)
	widths[0] = markColWidth

	colIdx := 1
	for i, col := range cols {
		w := col.Width
		if i == 0 {
			w += firstColGrowth
		}

		if i == len(cols)-1 && !hasTrailingCols {
			w += extraWidth
		}
		widths[colIdx] = w
		colIdx++
	}

	if isMultiProfile {
		widths[colIdx] = profileColWidth
		colIdx++
		widths[colIdx] = accountColWidth
		colIdx++
		w := regionColWidth
		if !hasMetrics {
			w += extraWidth
		}
		widths[colIdx] = w
		colIdx++
	} else if isMultiRegion {
		w := regionColWidth
		if !hasMetrics {
			w += extraWidth
		}
		widths[colIdx] = w
		colIdx++
	}

	if hasMetrics {
		widths[colIdx] = metricsColWidth + extraWidth
	}

	return widths
}
