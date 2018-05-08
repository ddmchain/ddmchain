
package tablewriter

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
)

const (
	MAX_ROW_WIDTH = 30
)

const (
	CENTER  = "+"
	ROW     = "-"
	COLUMN  = "|"
	SPACE   = " "
	NEWLINE = "\n"
)

const (
	ALIGN_DEFAULT = iota
	ALIGN_CENTER
	ALIGN_RIGHT
	ALIGN_LEFT
)

var (
	decimal = regexp.MustCompile(`^-*\d*\.?\d*$`)
	percent = regexp.MustCompile(`^-*\d*\.?\d*$%$`)
)

type Border struct {
	Left   bool
	Right  bool
	Top    bool
	Bottom bool
}

type Table struct {
	out            io.Writer
	rows           [][]string
	lines          [][][]string
	cs             map[int]int
	rs             map[int]int
	headers        []string
	footers        []string
	autoFmt        bool
	autoWrap       bool
	mW             int
	pCenter        string
	pRow           string
	pColumn        string
	tColumn        int
	tRow           int
	hAlign         int
	fAlign         int
	align          int
	newLine        string
	rowLine        bool
	autoMergeCells bool
	hdrLine        bool
	borders        Border
	colSize        int
}

func NewWriter(writer io.Writer) *Table {
	t := &Table{
		out:      writer,
		rows:     [][]string{},
		lines:    [][][]string{},
		cs:       make(map[int]int),
		rs:       make(map[int]int),
		headers:  []string{},
		footers:  []string{},
		autoFmt:  true,
		autoWrap: true,
		mW:       MAX_ROW_WIDTH,
		pCenter:  CENTER,
		pRow:     ROW,
		pColumn:  COLUMN,
		tColumn:  -1,
		tRow:     -1,
		hAlign:   ALIGN_DEFAULT,
		fAlign:   ALIGN_DEFAULT,
		align:    ALIGN_DEFAULT,
		newLine:  NEWLINE,
		rowLine:  false,
		hdrLine:  true,
		borders:  Border{Left: true, Right: true, Bottom: true, Top: true},
		colSize:  -1}
	return t
}

func (t Table) Render() {
	if t.borders.Top {
		t.printLine(true)
	}
	t.printHeading()
	if t.autoMergeCells {
		t.printRowsMergeCells()
	} else {
		t.printRows()
	}

	if !t.rowLine && t.borders.Bottom {
		t.printLine(true)
	}
	t.printFooter()

}

func (t *Table) SetHeader(keys []string) {
	t.colSize = len(keys)
	for i, v := range keys {
		t.parseDimension(v, i, -1)
		t.headers = append(t.headers, v)
	}
}

func (t *Table) SetFooter(keys []string) {

	for i, v := range keys {
		t.parseDimension(v, i, -1)
		t.footers = append(t.footers, v)
	}
}

func (t *Table) SetAutoFormatHeaders(auto bool) {
	t.autoFmt = auto
}

func (t *Table) SetAutoWrapText(auto bool) {
	t.autoWrap = auto
}

func (t *Table) SetColWidth(width int) {
	t.mW = width
}

func (t *Table) SetColumnSeparator(sep string) {
	t.pColumn = sep
}

func (t *Table) SetRowSeparator(sep string) {
	t.pRow = sep
}

func (t *Table) SetCenterSeparator(sep string) {
	t.pCenter = sep
}

func (t *Table) SetHeaderAlignment(hAlign int) {
	t.hAlign = hAlign
}

func (t *Table) SetFooterAlignment(fAlign int) {
	t.fAlign = fAlign
}

func (t *Table) SetAlignment(align int) {
	t.align = align
}

func (t *Table) SetNewLine(nl string) {
	t.newLine = nl
}

func (t *Table) SetHeaderLine(line bool) {
	t.hdrLine = line
}

func (t *Table) SetRowLine(line bool) {
	t.rowLine = line
}

func (t *Table) SetAutoMergeCells(auto bool) {
	t.autoMergeCells = auto
}

func (t *Table) SetBorder(border bool) {
	t.SetBorders(Border{border, border, border, border})
}

func (t *Table) SetBorders(border Border) {
	t.borders = border
}

func (t *Table) Append(row []string) {
	rowSize := len(t.headers)
	if rowSize > t.colSize {
		t.colSize = rowSize
	}

	n := len(t.lines)
	line := [][]string{}
	for i, v := range row {

		out := t.parseDimension(v, i, n)

		line = append(line, out)
	}
	t.lines = append(t.lines, line)
}

func (t *Table) AppendBulk(rows [][]string) {
	for _, row := range rows {
		t.Append(row)
	}
}

func (t Table) printLine(nl bool) {
	fmt.Fprint(t.out, t.pCenter)
	for i := 0; i < len(t.cs); i++ {
		v := t.cs[i]
		fmt.Fprintf(t.out, "%s%s%s%s",
			t.pRow,
			strings.Repeat(string(t.pRow), v),
			t.pRow,
			t.pCenter)
	}
	if nl {
		fmt.Fprint(t.out, t.newLine)
	}
}

func (t Table) printLineOptionalCellSeparators(nl bool, displayCellSeparator []bool) {
	fmt.Fprint(t.out, t.pCenter)
	for i := 0; i < len(t.cs); i++ {
		v := t.cs[i]
		if i > len(displayCellSeparator) || displayCellSeparator[i] {

			fmt.Fprintf(t.out, "%s%s%s%s",
				t.pRow,
				strings.Repeat(string(t.pRow), v),
				t.pRow,
				t.pCenter)
		} else {

			fmt.Fprintf(t.out, "%s%s",
				strings.Repeat(" ", v+2),
				t.pCenter)
		}
	}
	if nl {
		fmt.Fprint(t.out, t.newLine)
	}
}

func pad(align int) func(string, string, int) string {
	padFunc := Pad
	switch align {
	case ALIGN_LEFT:
		padFunc = PadRight
	case ALIGN_RIGHT:
		padFunc = PadLeft
	}
	return padFunc
}

func (t Table) printHeading() {

	if len(t.headers) < 1 {
		return
	}

	fmt.Fprint(t.out, ConditionString(t.borders.Left, t.pColumn, SPACE))

	end := len(t.cs) - 1

	padFunc := pad(t.hAlign)

	for i := 0; i <= end; i++ {
		v := t.cs[i]
		h := t.headers[i]
		if t.autoFmt {
			h = Title(h)
		}
		pad := ConditionString((i == end && !t.borders.Left), SPACE, t.pColumn)
		fmt.Fprintf(t.out, " %s %s",
			padFunc(h, SPACE, v),
			pad)
	}

	fmt.Fprint(t.out, t.newLine)
	if t.hdrLine {
		t.printLine(true)
	}
}

func (t Table) printFooter() {

	if len(t.footers) < 1 {
		return
	}

	if !t.borders.Bottom {
		t.printLine(true)
	}

	fmt.Fprint(t.out, ConditionString(t.borders.Bottom, t.pColumn, SPACE))

	end := len(t.cs) - 1

	padFunc := pad(t.fAlign)

	for i := 0; i <= end; i++ {
		v := t.cs[i]
		f := t.footers[i]
		if t.autoFmt {
			f = Title(f)
		}
		pad := ConditionString((i == end && !t.borders.Top), SPACE, t.pColumn)

		if len(t.footers[i]) == 0 {
			pad = SPACE
		}
		fmt.Fprintf(t.out, " %s %s",
			padFunc(f, SPACE, v),
			pad)
	}

	fmt.Fprint(t.out, t.newLine)

	hasPrinted := false

	for i := 0; i <= end; i++ {
		v := t.cs[i]
		pad := t.pRow
		center := t.pCenter
		length := len(t.footers[i])

		if length > 0 {
			hasPrinted = true
		}

		if length == 0 && !t.borders.Right {
			center = SPACE
		}

		if i == 0 {
			fmt.Fprint(t.out, center)
		}

		if length == 0 {
			pad = SPACE
		}

		if hasPrinted || t.borders.Left {
			pad = t.pRow
			center = t.pCenter
		}

		if center == SPACE {
			if i < end && len(t.footers[i+1]) != 0 {
				center = t.pCenter
			}
		}

		fmt.Fprintf(t.out, "%s%s%s%s",
			pad,
			strings.Repeat(string(pad), v),
			pad,
			center)

	}

	fmt.Fprint(t.out, t.newLine)

}

func (t Table) printRows() {
	for i, lines := range t.lines {
		t.printRow(lines, i)
	}

}

func (t Table) printRow(columns [][]string, colKey int) {

	max := t.rs[colKey]
	total := len(columns)

	pads := []int{}

	for i, line := range columns {
		length := len(line)
		pad := max - length
		pads = append(pads, pad)
		for n := 0; n < pad; n++ {
			columns[i] = append(columns[i], "  ")
		}
	}

	for x := 0; x < max; x++ {
		for y := 0; y < total; y++ {

			fmt.Fprint(t.out, ConditionString((!t.borders.Left && y == 0), SPACE, t.pColumn))

			fmt.Fprintf(t.out, SPACE)
			str := columns[y][x]

			switch t.align {
			case ALIGN_CENTER: 
				fmt.Fprintf(t.out, "%s", Pad(str, SPACE, t.cs[y]))
			case ALIGN_RIGHT:
				fmt.Fprintf(t.out, "%s", PadLeft(str, SPACE, t.cs[y]))
			case ALIGN_LEFT:
				fmt.Fprintf(t.out, "%s", PadRight(str, SPACE, t.cs[y]))
			default:
				if decimal.MatchString(strings.TrimSpace(str)) || percent.MatchString(strings.TrimSpace(str)) {
					fmt.Fprintf(t.out, "%s", PadLeft(str, SPACE, t.cs[y]))
				} else {
					fmt.Fprintf(t.out, "%s", PadRight(str, SPACE, t.cs[y]))

				}
			}
			fmt.Fprintf(t.out, SPACE)
		}

		fmt.Fprint(t.out, ConditionString(t.borders.Left, t.pColumn, SPACE))
		fmt.Fprint(t.out, t.newLine)
	}

	if t.rowLine {
		t.printLine(true)
	}
}

func (t Table) printRowsMergeCells() {
	var previousLine []string
	var displayCellBorder []bool
	var tmpWriter bytes.Buffer
	for i, lines := range t.lines {

		previousLine, displayCellBorder = t.printRowMergeCells(&tmpWriter, lines, i, previousLine)
		if i > 0 { 
			if t.rowLine {
				t.printLineOptionalCellSeparators(true, displayCellBorder)
			}
		}
		tmpWriter.WriteTo(t.out)
	}

	if t.rowLine {
		t.printLine(true)
	}
}

func (t Table) printRowMergeCells(writer io.Writer, columns [][]string, colKey int, previousLine []string) ([]string, []bool) {

	max := t.rs[colKey]
	total := len(columns)

	pads := []int{}

	for i, line := range columns {
		length := len(line)
		pad := max - length
		pads = append(pads, pad)
		for n := 0; n < pad; n++ {
			columns[i] = append(columns[i], "  ")
		}
	}

	var displayCellBorder []bool
	for x := 0; x < max; x++ {
		for y := 0; y < total; y++ {

			fmt.Fprint(writer, ConditionString((!t.borders.Left && y == 0), SPACE, t.pColumn))

			fmt.Fprintf(writer, SPACE)

			str := columns[y][x]

			if t.autoMergeCells {

				fullLine := strings.Join(columns[y], " ")
				if len(previousLine) > y && fullLine == previousLine[y] && fullLine != "" {

					displayCellBorder = append(displayCellBorder, false)
					str = ""
				} else {

					displayCellBorder = append(displayCellBorder, true)
				}
			}

			switch t.align {
			case ALIGN_CENTER: 
				fmt.Fprintf(writer, "%s", Pad(str, SPACE, t.cs[y]))
			case ALIGN_RIGHT:
				fmt.Fprintf(writer, "%s", PadLeft(str, SPACE, t.cs[y]))
			case ALIGN_LEFT:
				fmt.Fprintf(writer, "%s", PadRight(str, SPACE, t.cs[y]))
			default:
				if decimal.MatchString(strings.TrimSpace(str)) || percent.MatchString(strings.TrimSpace(str)) {
					fmt.Fprintf(writer, "%s", PadLeft(str, SPACE, t.cs[y]))
				} else {
					fmt.Fprintf(writer, "%s", PadRight(str, SPACE, t.cs[y]))
				}
			}
			fmt.Fprintf(writer, SPACE)
		}

		fmt.Fprint(writer, ConditionString(t.borders.Left, t.pColumn, SPACE))
		fmt.Fprint(writer, t.newLine)
	}

	previousLine = make([]string, total)
	for y := 0; y < total; y++ {
		previousLine[y] = strings.Join(columns[y], " ") 
	}

	return previousLine, displayCellBorder
}

func (t *Table) parseDimension(str string, colKey, rowKey int) []string {
	var (
		raw []string
		max int
	)
	w := DisplayWidth(str)

	if w > t.mW {
		w = t.mW
	}

	v, ok := t.cs[colKey]
	if !ok || v < w || v == 0 {
		t.cs[colKey] = w
	}

	if rowKey == -1 {
		return raw
	}

	if t.autoWrap {
		raw, _ = WrapString(str, t.cs[colKey])
	} else {
		raw = getLines(str)
	}

	for _, line := range raw {
		if w := DisplayWidth(line); w > max {
			max = w
		}
	}

	if max > t.cs[colKey] {
		t.cs[colKey] = max
	}

	h := len(raw)
	v, ok = t.rs[rowKey]

	if !ok || v < h || v == 0 {
		t.rs[rowKey] = h
	}

	return raw
}
