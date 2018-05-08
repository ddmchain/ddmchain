
package termui

import "image"

type Hline struct {
	X   int
	Y   int
	Len int
	Fg  Attribute
	Bg  Attribute
}

type Vline struct {
	X   int
	Y   int
	Len int
	Fg  Attribute
	Bg  Attribute
}

func (l Hline) Buffer() Buffer {
	if l.Len <= 0 {
		return NewBuffer()
	}
	return NewFilledBuffer(l.X, l.Y, l.X+l.Len, l.Y+1, HORIZONTAL_LINE, l.Fg, l.Bg)
}

func (l Vline) Buffer() Buffer {
	if l.Len <= 0 {
		return NewBuffer()
	}
	return NewFilledBuffer(l.X, l.Y, l.X+1, l.Y+l.Len, VERTICAL_LINE, l.Fg, l.Bg)
}

func (b Block) drawBorder(buf Buffer) {
	if !b.Border {
		return
	}

	min := b.area.Min
	max := b.area.Max

	x0 := min.X
	y0 := min.Y
	x1 := max.X - 1
	y1 := max.Y - 1

	if b.BorderTop {
		buf.Merge(Hline{x0, y0, x1 - x0, b.BorderFg, b.BorderBg}.Buffer())
	}
	if b.BorderBottom {
		buf.Merge(Hline{x0, y1, x1 - x0, b.BorderFg, b.BorderBg}.Buffer())
	}
	if b.BorderLeft {
		buf.Merge(Vline{x0, y0, y1 - y0, b.BorderFg, b.BorderBg}.Buffer())
	}
	if b.BorderRight {
		buf.Merge(Vline{x1, y0, y1 - y0, b.BorderFg, b.BorderBg}.Buffer())
	}

	if b.BorderTop && b.BorderLeft && b.area.Dx() > 0 && b.area.Dy() > 0 {
		buf.Set(x0, y0, Cell{TOP_LEFT, b.BorderFg, b.BorderBg})
	}
	if b.BorderTop && b.BorderRight && b.area.Dx() > 1 && b.area.Dy() > 0 {
		buf.Set(x1, y0, Cell{TOP_RIGHT, b.BorderFg, b.BorderBg})
	}
	if b.BorderBottom && b.BorderLeft && b.area.Dx() > 0 && b.area.Dy() > 1 {
		buf.Set(x0, y1, Cell{BOTTOM_LEFT, b.BorderFg, b.BorderBg})
	}
	if b.BorderBottom && b.BorderRight && b.area.Dx() > 1 && b.area.Dy() > 1 {
		buf.Set(x1, y1, Cell{BOTTOM_RIGHT, b.BorderFg, b.BorderBg})
	}
}

func (b Block) drawBorderLabel(buf Buffer) {
	maxTxtW := b.area.Dx() - 2
	tx := DTrimTxCls(DefaultTxBuilder.Build(b.BorderLabel, b.BorderLabelFg, b.BorderLabelBg), maxTxtW)

	for i, w := 0, 0; i < len(tx); i++ {
		buf.Set(b.area.Min.X+1+w, b.area.Min.Y, tx[i])
		w += tx[i].Width()
	}
}

type Block struct {
	area          image.Rectangle
	innerArea     image.Rectangle
	X             int
	Y             int
	Border        bool
	BorderFg      Attribute
	BorderBg      Attribute
	BorderLeft    bool
	BorderRight   bool
	BorderTop     bool
	BorderBottom  bool
	BorderLabel   string
	BorderLabelFg Attribute
	BorderLabelBg Attribute
	Display       bool
	Bg            Attribute
	Width         int
	Height        int
	PaddingTop    int
	PaddingBottom int
	PaddingLeft   int
	PaddingRight  int
	id            string
	Float         Align
}

func NewBlock() *Block {
	b := Block{}
	b.Display = true
	b.Border = true
	b.BorderLeft = true
	b.BorderRight = true
	b.BorderTop = true
	b.BorderBottom = true
	b.BorderBg = ThemeAttr("border.bg")
	b.BorderFg = ThemeAttr("border.fg")
	b.BorderLabelBg = ThemeAttr("label.bg")
	b.BorderLabelFg = ThemeAttr("label.fg")
	b.Bg = ThemeAttr("block.bg")
	b.Width = 2
	b.Height = 2
	b.id = GenId()
	b.Float = AlignNone
	return &b
}

func (b Block) Id() string {
	return b.id
}

func (b *Block) Align() {

	b.area.Min.X = 0
	b.area.Min.Y = 0
	b.area.Max.X = b.Width
	b.area.Max.Y = b.Height

	b.area = AlignArea(TermRect(), b.area, b.Float)
	b.area = MoveArea(b.area, b.X, b.Y)

	b.innerArea.Min.X = b.area.Min.X + b.PaddingLeft
	b.innerArea.Min.Y = b.area.Min.Y + b.PaddingTop
	b.innerArea.Max.X = b.area.Max.X - b.PaddingRight
	b.innerArea.Max.Y = b.area.Max.Y - b.PaddingBottom

	if b.Border {
		if b.BorderLeft {
			b.innerArea.Min.X++
		}
		if b.BorderRight {
			b.innerArea.Max.X--
		}
		if b.BorderTop {
			b.innerArea.Min.Y++
		}
		if b.BorderBottom {
			b.innerArea.Max.Y--
		}
	}
}

func (b *Block) InnerBounds() image.Rectangle {
	b.Align()
	return b.innerArea
}

func (b *Block) Buffer() Buffer {
	b.Align()

	buf := NewBuffer()
	buf.SetArea(b.area)
	buf.Fill(' ', ColorDefault, b.Bg)

	b.drawBorder(buf)
	b.drawBorderLabel(buf)

	return buf
}

func (b Block) GetHeight() int {
	return b.Height
}

func (b *Block) SetX(x int) {
	b.X = x
}

func (b *Block) SetY(y int) {
	b.Y = y
}

func (b *Block) SetWidth(w int) {
	b.Width = w
}

func (b Block) InnerWidth() int {
	return b.innerArea.Dx()
}

func (b Block) InnerHeight() int {
	return b.innerArea.Dy()
}

func (b Block) InnerX() int {
	return b.innerArea.Min.X
}

func (b Block) InnerY() int { return b.innerArea.Min.Y }
