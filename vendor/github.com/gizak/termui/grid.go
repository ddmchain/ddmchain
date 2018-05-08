
package termui

type GridBufferer interface {
	Bufferer
	GetHeight() int
	SetWidth(int)
	SetX(int)
	SetY(int)
}

type Row struct {
	Cols   []*Row       
	Widget GridBufferer 
	X      int
	Y      int
	Width  int
	Height int
	Span   int
	Offset int
}

func (r *Row) calcLayout() {
	r.assignWidth(r.Width)
	r.Height = r.solveHeight()
	r.assignX(r.X)
	r.assignY(r.Y)
}

func (r *Row) isLeaf() bool {
	return r.Cols == nil || len(r.Cols) == 0
}

func (r *Row) isRenderableLeaf() bool {
	return r.isLeaf() && r.Widget != nil
}

func (r *Row) assignWidth(w int) {
	r.SetWidth(w)

	accW := 0                            
	calcW := make([]int, len(r.Cols))    
	calcOftX := make([]int, len(r.Cols)) 

	for i, c := range r.Cols {
		accW += c.Span + c.Offset
		cw := int(float64(c.Span*r.Width) / 12.0)

		if i >= 1 {
			calcOftX[i] = calcOftX[i-1] +
				calcW[i-1] +
				int(float64(r.Cols[i-1].Offset*r.Width)/12.0)
		}

		if i == len(r.Cols)-1 && accW == 12 {
			cw = r.Width - calcOftX[i]
		}
		calcW[i] = cw
		r.Cols[i].assignWidth(cw)
	}
}

func (r *Row) solveHeight() int {
	if r.isRenderableLeaf() {
		r.Height = r.Widget.GetHeight()
		return r.Widget.GetHeight()
	}

	maxh := 0
	if !r.isLeaf() {
		for _, c := range r.Cols {
			nh := c.solveHeight()

			if r.Widget != nil {
				nh += r.Widget.GetHeight()
			}
			if nh > maxh {
				maxh = nh
			}
		}
	}

	r.Height = maxh
	return maxh
}

func (r *Row) assignX(x int) {
	r.SetX(x)

	if !r.isLeaf() {
		acc := 0
		for i, c := range r.Cols {
			if c.Offset != 0 {
				acc += int(float64(c.Offset*r.Width) / 12.0)
			}
			r.Cols[i].assignX(x + acc)
			acc += c.Width
		}
	}
}

func (r *Row) assignY(y int) {
	r.SetY(y)

	if r.isLeaf() {
		return
	}

	for i := range r.Cols {
		acc := 0
		if r.Widget != nil {
			acc = r.Widget.GetHeight()
		}
		r.Cols[i].assignY(y + acc)
	}

}

func (r Row) GetHeight() int {
	return r.Height
}

func (r *Row) SetX(x int) {
	r.X = x
	if r.Widget != nil {
		r.Widget.SetX(x)
	}
}

func (r *Row) SetY(y int) {
	r.Y = y
	if r.Widget != nil {
		r.Widget.SetY(y)
	}
}

func (r *Row) SetWidth(w int) {
	r.Width = w
	if r.Widget != nil {
		r.Widget.SetWidth(w)
	}
}

func (r *Row) Buffer() Buffer {
	merged := NewBuffer()

	if r.isRenderableLeaf() {
		return r.Widget.Buffer()
	}

	if r.Widget != nil {
		merged.Merge(r.Widget.Buffer())
	}

	if !r.isLeaf() {
		for _, c := range r.Cols {
			merged.Merge(c.Buffer())
		}
	}

	return merged
}

type Grid struct {
	Rows    []*Row
	Width   int
	X       int
	Y       int
	BgColor Attribute
}

func NewGrid(rows ...*Row) *Grid {
	return &Grid{Rows: rows}
}

func (g *Grid) AddRows(rs ...*Row) {
	g.Rows = append(g.Rows, rs...)
}

func NewRow(cols ...*Row) *Row {
	rs := &Row{Span: 12, Cols: cols}
	return rs
}

func NewCol(span, offset int, widgets ...GridBufferer) *Row {
	r := &Row{Span: span, Offset: offset}

	if widgets != nil && len(widgets) == 1 {
		wgt := widgets[0]
		nw, isRow := wgt.(*Row)
		if isRow {
			r.Cols = nw.Cols
		} else {
			r.Widget = wgt
		}
		return r
	}

	r.Cols = []*Row{}
	ir := r
	for _, w := range widgets {
		nr := &Row{Span: 12, Widget: w}
		ir.Cols = []*Row{nr}
		ir = nr
	}

	return r
}

func (g *Grid) Align() {
	h := 0
	for _, r := range g.Rows {
		r.SetWidth(g.Width)
		r.SetX(g.X)
		r.SetY(g.Y + h)
		r.calcLayout()
		h += r.GetHeight()
	}
}

func (g Grid) Buffer() Buffer {
	buf := NewBuffer()

	for _, r := range g.Rows {
		buf.Merge(r.Buffer())
	}
	return buf
}

var Body *Grid
