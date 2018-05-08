
package termui

import (
	"regexp"
	"strings"

	"github.com/mitchellh/go-wordwrap"
)

type TextBuilder interface {
	Build(s string, fg, bg Attribute) []Cell
}

var DefaultTxBuilder = NewMarkdownTxBuilder()

type MarkdownTxBuilder struct {
	baseFg  Attribute
	baseBg  Attribute
	plainTx []rune
	markers []marker
}

type marker struct {
	st int
	ed int
	fg Attribute
	bg Attribute
}

var colorMap = map[string]Attribute{
	"red":     ColorRed,
	"blue":    ColorBlue,
	"black":   ColorBlack,
	"cyan":    ColorCyan,
	"yellow":  ColorYellow,
	"white":   ColorWhite,
	"default": ColorDefault,
	"green":   ColorGreen,
	"magenta": ColorMagenta,
}

var attrMap = map[string]Attribute{
	"bold":      AttrBold,
	"underline": AttrUnderline,
	"reverse":   AttrReverse,
}

func rmSpc(s string) string {
	reg := regexp.MustCompile(`\s+`)
	return reg.ReplaceAllString(s, "")
}

func (mtb MarkdownTxBuilder) readAttr(s string) (Attribute, Attribute) {
	fg := mtb.baseFg
	bg := mtb.baseBg

	updateAttr := func(a Attribute, attrs []string) Attribute {
		for _, s := range attrs {

			if c, ok := colorMap[s]; ok {
				a &= 0xFF00 
				a |= c      
			}

			if c, ok := attrMap[s]; ok {
				a |= c
			}
		}
		return a
	}

	ss := strings.Split(s, ",")
	fgs := []string{}
	bgs := []string{}
	for _, v := range ss {
		subs := strings.Split(v, "-")
		if len(subs) > 1 {
			if subs[0] == "fg" {
				fgs = append(fgs, subs[1])
			}
			if subs[0] == "bg" {
				bgs = append(bgs, subs[1])
			}
		}
	}

	fg = updateAttr(fg, fgs)
	bg = updateAttr(bg, bgs)
	return fg, bg
}

func (mtb *MarkdownTxBuilder) reset() {
	mtb.plainTx = []rune{}
	mtb.markers = []marker{}
}

func (mtb *MarkdownTxBuilder) parse(str string) {
	rs := str2runes(str)
	normTx := []rune{}
	square := []rune{}
	brackt := []rune{}
	accSquare := false
	accBrackt := false
	cntSquare := 0

	reset := func() {
		square = []rune{}
		brackt = []rune{}
		accSquare = false
		accBrackt = false
		cntSquare = 0
	}

	rollback := func() {
		normTx = append(normTx, square...)
		normTx = append(normTx, brackt...)
		reset()
	}

	chop := func(s []rune) []rune {
		return s[1 : len(s)-1]
	}

	for i, r := range rs {
		switch {

		case accBrackt:
			brackt = append(brackt, r)
			if ')' == r {
				fg, bg := mtb.readAttr(string(chop(brackt)))
				st := len(normTx)
				ed := len(normTx) + len(square) - 2
				mtb.markers = append(mtb.markers, marker{st, ed, fg, bg})
				normTx = append(normTx, chop(square)...)
				reset()
			} else if i+1 == len(rs) {
				rollback()
			}

		case accSquare:
			switch {

			case cntSquare == 0 && '(' == r:
				accBrackt = true
				brackt = append(brackt, '(')

			case cntSquare == 0:
				rollback()
				if '[' == r {
					accSquare = true
					cntSquare = 1
					brackt = append(brackt, '[')
				} else {
					normTx = append(normTx, r)
				}

			case i+1 == len(rs):
				square = append(square, r)
				rollback()
			case '[' == r:
				cntSquare++
				square = append(square, '[')
			case ']' == r:
				cntSquare--
				square = append(square, ']')

			default:
				square = append(square, r)
			}

		default:
			if '[' == r {
				accSquare = true
				cntSquare = 1
				square = append(square, '[')
			} else {
				normTx = append(normTx, r)
			}
		}
	}

	mtb.plainTx = normTx
}

func wrapTx(cs []Cell, wl int) []Cell {
	tmpCell := make([]Cell, len(cs))
	copy(tmpCell, cs)

	plain := CellsToStr(cs)

	plainWrapped := wordwrap.WrapString(plain, uint(wl))

	finalCell := tmpCell 

	plainRune := []rune(plain)
	plainWrappedRune := []rune(plainWrapped)
	trigger := "go"
	plainRuneNew := plainRune

	for trigger != "stop" {
		plainRune = plainRuneNew
		for i := range plainRune {
			if plainRune[i] == plainWrappedRune[i] {
				trigger = "stop"
			} else if plainRune[i] != plainWrappedRune[i] && plainWrappedRune[i] == 10 {
				trigger = "go"
				cell := Cell{10, 0, 0}
				j := i - 0

				tmpCell[i] = cell

				plainRuneNew = append(plainRune, 10)
				copy(plainRuneNew[j+1:], plainRuneNew[j:])
				plainRuneNew[j] = plainWrappedRune[j]

				break

			} else if plainRune[i] != plainWrappedRune[i] &&
				plainWrappedRune[i-1] == 10 && 
				plainRune[i] == 32 { 
				trigger = "go"

				plainRuneNew = append(plainRune[:i], plainRune[i+1:]...)
				break

			} else {
				trigger = "stop" 
			}
		}
	}

	finalCell = tmpCell

	return finalCell
}

func (mtb MarkdownTxBuilder) Build(s string, fg, bg Attribute) []Cell {
	mtb.baseFg = fg
	mtb.baseBg = bg
	mtb.reset()
	mtb.parse(s)
	cs := make([]Cell, len(mtb.plainTx))
	for i := range cs {
		cs[i] = Cell{Ch: mtb.plainTx[i], Fg: fg, Bg: bg}
	}
	for _, mrk := range mtb.markers {
		for i := mrk.st; i < mrk.ed; i++ {
			cs[i].Fg = mrk.fg
			cs[i].Bg = mrk.bg
		}
	}

	return cs
}

func NewMarkdownTxBuilder() TextBuilder {
	return MarkdownTxBuilder{}
}
