
package termui

import (
	"fmt"
)

type MBarChart struct {
	Block
	BarColor   [NumberofColors]Attribute
	TextColor  Attribute
	NumColor   [NumberofColors]Attribute
	Data       [NumberofColors][]int
	DataLabels []string
	BarWidth   int
	BarGap     int
	labels     [][]rune
	dataNum    [NumberofColors][][]rune
	numBar     int
	scale      float64
	max        int
	minDataLen int
	numStack   int
	ShowScale  bool
	maxScale   []rune
}

func NewMBarChart() *MBarChart {
	bc := &MBarChart{Block: *NewBlock()}
	bc.BarColor[0] = ThemeAttr("mbarchart.bar.bg")
	bc.NumColor[0] = ThemeAttr("mbarchart.num.fg")
	bc.TextColor = ThemeAttr("mbarchart.text.fg")
	bc.BarGap = 1
	bc.BarWidth = 3
	return bc
}

func (bc *MBarChart) layout() {
	bc.numBar = bc.innerArea.Dx() / (bc.BarGap + bc.BarWidth)
	bc.labels = make([][]rune, bc.numBar)
	DataLen := 0
	LabelLen := len(bc.DataLabels)
	bc.minDataLen = 9999 

	for i := 0; i < len(bc.Data); i++ {
		if bc.Data[i] == nil {
			break
		}
		DataLen++
	}
	bc.numStack = DataLen

	for i := 0; i < DataLen; i++ {
		if bc.minDataLen > len(bc.Data[i]) {
			bc.minDataLen = len(bc.Data[i])
		}
	}

	if LabelLen > bc.minDataLen {
		LabelLen = bc.minDataLen
	}

	for i := 0; i < LabelLen && i < bc.numBar; i++ {
		bc.labels[i] = trimStr2Runes(bc.DataLabels[i], bc.BarWidth)
	}

	for i := 0; i < bc.numStack; i++ {
		bc.dataNum[i] = make([][]rune, len(bc.Data[i]))

		for j := 0; j < LabelLen && i < bc.numBar; j++ {
			n := bc.Data[i][j]
			s := fmt.Sprint(n)
			bc.dataNum[i][j] = trimStr2Runes(s, bc.BarWidth)
		}

		if bc.BarColor[i] == ColorDefault && bc.NumColor[i] == ColorDefault {
			if i == 0 {
				bc.BarColor[i] = ColorBlack
			} else {
				bc.BarColor[i] = bc.BarColor[i-1] + 1
				if bc.BarColor[i] > NumberofColors {
					bc.BarColor[i] = ColorBlack
				}
			}
			bc.NumColor[i] = (NumberofColors + 1) - bc.BarColor[i] 
		}
	}

	if bc.max == 0 {
		bc.max = -1
	}
	for i := 0; i < bc.minDataLen && i < LabelLen; i++ {
		var dsum int
		for j := 0; j < bc.numStack; j++ {
			dsum += bc.Data[j][i]
		}
		if dsum > bc.max {
			bc.max = dsum
		}
	}

	if bc.ShowScale {
		s := fmt.Sprintf("%d", bc.max)
		bc.maxScale = trimStr2Runes(s, len(s))
		bc.scale = float64(bc.max) / float64(bc.innerArea.Dy()-2)
	} else {
		bc.scale = float64(bc.max) / float64(bc.innerArea.Dy()-1)
	}

}

func (bc *MBarChart) SetMax(max int) {

	if max > 0 {
		bc.max = max
	}
}

func (bc *MBarChart) Buffer() Buffer {
	buf := bc.Block.Buffer()
	bc.layout()
	var oftX int

	for i := 0; i < bc.numBar && i < bc.minDataLen && i < len(bc.DataLabels); i++ {
		ph := 0 
		oftX = i * (bc.BarWidth + bc.BarGap)
		for i1 := 0; i1 < bc.numStack; i1++ {
			h := int(float64(bc.Data[i1][i]) / bc.scale)

			for j := 0; j < bc.BarWidth; j++ {
				for k := 0; k < h; k++ {
					c := Cell{
						Ch: ' ',
						Bg: bc.BarColor[i1],
					}
					if bc.BarColor[i1] == ColorDefault { 
						c.Bg |= AttrReverse
					}
					x := bc.innerArea.Min.X + i*(bc.BarWidth+bc.BarGap) + j
					y := bc.innerArea.Min.Y + bc.innerArea.Dy() - 2 - k - ph
					buf.Set(x, y, c)

				}
			}
			ph += h
		}

		for j, k := 0, 0; j < len(bc.labels[i]); j++ {
			w := charWidth(bc.labels[i][j])
			c := Cell{
				Ch: bc.labels[i][j],
				Bg: bc.Bg,
				Fg: bc.TextColor,
			}
			y := bc.innerArea.Min.Y + bc.innerArea.Dy() - 1
			x := bc.innerArea.Max.X + oftX + ((bc.BarWidth - len(bc.labels[i])) / 2) + k
			buf.Set(x, y, c)
			k += w
		}

		ph = 0 
		for i1 := 0; i1 < bc.numStack; i1++ {
			h := int(float64(bc.Data[i1][i]) / bc.scale)
			for j := 0; j < len(bc.dataNum[i1][i]) && h > 0; j++ {
				c := Cell{
					Ch: bc.dataNum[i1][i][j],
					Fg: bc.NumColor[i1],
					Bg: bc.BarColor[i1],
				}
				if bc.BarColor[i1] == ColorDefault { 
					c.Bg |= AttrReverse
				}
				if h == 0 {
					c.Bg = bc.Bg
				}
				x := bc.innerArea.Min.X + oftX + (bc.BarWidth-len(bc.dataNum[i1][i]))/2 + j
				y := bc.innerArea.Min.Y + bc.innerArea.Dy() - 2 - ph
				buf.Set(x, y, c)
			}
			ph += h
		}
	}

	if bc.ShowScale {

		c := Cell{
			Ch: '0',
			Bg: bc.Bg,
			Fg: bc.TextColor,
		}

		y := bc.innerArea.Min.Y + bc.innerArea.Dy() - 2
		x := bc.X
		buf.Set(x, y, c)

		for i := 0; i < len(bc.maxScale); i++ {
			c := Cell{
				Ch: bc.maxScale[i],
				Bg: bc.Bg,
				Fg: bc.TextColor,
			}

			y := bc.innerArea.Min.Y
			x := bc.X + i

			buf.Set(x, y, c)
		}

	}

	return buf
}
