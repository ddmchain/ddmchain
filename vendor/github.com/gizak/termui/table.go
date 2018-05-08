
package termui

import "strings"

type Table struct {
	Block
	Rows      [][]string
	CellWidth []int
	FgColor   Attribute
	BgColor   Attribute
	FgColors  []Attribute
	BgColors  []Attribute
	Separator bool
	TextAlign Align
}

func NewTable() *Table {
	table := &Table{Block: *NewBlock()}
	table.FgColor = ColorWhite
	table.BgColor = ColorDefault
	table.Separator = true
	return table
}

func cellsWidth(cells []Cell) int {
	width := 0
	for _, c := range cells {
		width += c.Width()
	}
	return width
}

func (table *Table) Analysis() [][]Cell {
	var rowCells [][]Cell
	length := len(table.Rows)
	if length < 1 {
		return rowCells
	}

	if len(table.FgColors) == 0 {
		table.FgColors = make([]Attribute, len(table.Rows))
	}
	if len(table.BgColors) == 0 {
		table.BgColors = make([]Attribute, len(table.Rows))
	}

	cellWidths := make([]int, len(table.Rows[0]))

	for y, row := range table.Rows {
		if table.FgColors[y] == 0 {
			table.FgColors[y] = table.FgColor
		}
		if table.BgColors[y] == 0 {
			table.BgColors[y] = table.BgColor
		}
		for x, str := range row {
			cells := DefaultTxBuilder.Build(str, table.FgColors[y], table.BgColors[y])
			cw := cellsWidth(cells)
			if cellWidths[x] < cw {
				cellWidths[x] = cw
			}
			rowCells = append(rowCells, cells)
		}
	}
	table.CellWidth = cellWidths
	return rowCells
}

func (table *Table) SetSize() {
	length := len(table.Rows)
	if table.Separator {
		table.Height = length*2 + 1
	} else {
		table.Height = length + 2
	}
	table.Width = 2
	if length != 0 {
		for _, cellWidth := range table.CellWidth {
			table.Width += cellWidth + 3
		}
	}
}

func (table *Table) CalculatePosition(x int, y int, coordinateX *int, coordinateY *int, cellStart *int) {
	if table.Separator {
		*coordinateY = table.innerArea.Min.Y + y*2
	} else {
		*coordinateY = table.innerArea.Min.Y + y
	}
	if x == 0 {
		*cellStart = table.innerArea.Min.X
	} else {
		*cellStart += table.CellWidth[x-1] + 3
	}

	switch table.TextAlign {
	case AlignRight:
		*coordinateX = *cellStart + (table.CellWidth[x] - len(table.Rows[y][x])) + 2
	case AlignCenter:
		*coordinateX = *cellStart + (table.CellWidth[x]-len(table.Rows[y][x]))/2 + 2
	default:
		*coordinateX = *cellStart + 2
	}
}

func (table *Table) Buffer() Buffer {
	buffer := table.Block.Buffer()
	rowCells := table.Analysis()
	pointerX := table.innerArea.Min.X + 2
	pointerY := table.innerArea.Min.Y
	borderPointerX := table.innerArea.Min.X
	for y, row := range table.Rows {
		for x := range row {
			table.CalculatePosition(x, y, &pointerX, &pointerY, &borderPointerX)
			background := DefaultTxBuilder.Build(strings.Repeat(" ", table.CellWidth[x]+3), table.BgColors[y], table.BgColors[y])
			cells := rowCells[y*len(row)+x]
			for i, back := range background {
				buffer.Set(borderPointerX+i, pointerY, back)
			}

			coordinateX := pointerX
			for _, printer := range cells {
				buffer.Set(coordinateX, pointerY, printer)
				coordinateX += printer.Width()
			}

			if x != 0 {
				dividors := DefaultTxBuilder.Build("|", table.FgColors[y], table.BgColors[y])
				for _, dividor := range dividors {
					buffer.Set(borderPointerX, pointerY, dividor)
				}
			}
		}

		if table.Separator {
			border := DefaultTxBuilder.Build(strings.Repeat("─", table.Width-2), table.FgColor, table.BgColor)
			for i, cell := range border {
				buffer.Set(i+1, pointerY+1, cell)
			}
		}
	}

	return buffer
}
