
package termui

type Sparkline struct {
	Data          []int
	Height        int
	Title         string
	TitleColor    Attribute
	LineColor     Attribute
	displayHeight int
	scale         float32
	max           int
}

type Sparklines struct {
	Block
	Lines        []Sparkline
	displayLines int
	displayWidth int
}

var sparks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

func (s *Sparklines) Add(sl Sparkline) {
	s.Lines = append(s.Lines, sl)
}

func NewSparkline() Sparkline {
	return Sparkline{
		Height:     1,
		TitleColor: ThemeAttr("sparkline.title.fg"),
		LineColor:  ThemeAttr("sparkline.line.fg")}
}

func NewSparklines(ss ...Sparkline) *Sparklines {
	s := &Sparklines{Block: *NewBlock(), Lines: ss}
	return s
}

func (sl *Sparklines) update() {
	for i, v := range sl.Lines {
		if v.Title == "" {
			sl.Lines[i].displayHeight = v.Height
		} else {
			sl.Lines[i].displayHeight = v.Height + 1
		}
	}
	sl.displayWidth = sl.innerArea.Dx()

	h := 0
	sl.displayLines = 0
	for _, v := range sl.Lines {
		if h+v.displayHeight <= sl.innerArea.Dy() {
			sl.displayLines++
		} else {
			break
		}
		h += v.displayHeight
	}

	for i := 0; i < sl.displayLines; i++ {
		data := sl.Lines[i].Data

		max := 0
		for _, v := range data {
			if max < v {
				max = v
			}
		}
		sl.Lines[i].max = max
		if max != 0 {
			sl.Lines[i].scale = float32(8*sl.Lines[i].Height) / float32(max)
		} else { 
			sl.Lines[i].scale = 0
		}
	}
}

func (sl *Sparklines) Buffer() Buffer {
	buf := sl.Block.Buffer()
	sl.update()

	oftY := 0
	for i := 0; i < sl.displayLines; i++ {
		l := sl.Lines[i]
		data := l.Data

		if len(data) > sl.innerArea.Dx() {
			data = data[len(data)-sl.innerArea.Dx():]
		}

		if l.Title != "" {
			rs := trimStr2Runes(l.Title, sl.innerArea.Dx())
			oftX := 0
			for _, v := range rs {
				w := charWidth(v)
				c := Cell{
					Ch: v,
					Fg: l.TitleColor,
					Bg: sl.Bg,
				}
				x := sl.innerArea.Min.X + oftX
				y := sl.innerArea.Min.Y + oftY
				buf.Set(x, y, c)
				oftX += w
			}
		}

		for j, v := range data {

			h := int(float32(v)*l.scale + 0.5)
			if v < 0 {
				h = 0
			}

			barCnt := h / 8
			barMod := h % 8
			for jj := 0; jj < barCnt; jj++ {
				c := Cell{
					Ch: ' ', 
					Bg: l.LineColor,
				}
				x := sl.innerArea.Min.X + j
				y := sl.innerArea.Min.Y + oftY + l.Height - jj

				buf.Set(x, y, c)
			}
			if barMod != 0 {
				c := Cell{
					Ch: sparks[barMod-1],
					Fg: l.LineColor,
					Bg: sl.Bg,
				}
				x := sl.innerArea.Min.X + j
				y := sl.innerArea.Min.Y + oftY + l.Height - barCnt
				buf.Set(x, y, c)
			}
		}

		oftY += l.displayHeight
	}

	return buf
}
