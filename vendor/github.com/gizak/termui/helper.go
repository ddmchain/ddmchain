
package termui

import (
	"regexp"
	"strings"

	tm "github.com/nsf/termbox-go"
)
import rw "github.com/mattn/go-runewidth"

type Attribute uint16

const (
	ColorDefault Attribute = iota
	ColorBlack
	ColorRed
	ColorGreen
	ColorYellow
	ColorBlue
	ColorMagenta
	ColorCyan
	ColorWhite
)

const NumberofColors = 8

const (
	AttrBold Attribute = 1 << (iota + 9)
	AttrUnderline
	AttrReverse
)

var (
	dot  = "…"
	dotw = rw.StringWidth(dot)
)

func toTmAttr(x Attribute) tm.Attribute {
	return tm.Attribute(x)
}

func str2runes(s string) []rune {
	return []rune(s)
}

func trimStr2Runes(s string, w int) []rune {
	return TrimStr2Runes(s, w)
}

func TrimStr2Runes(s string, w int) []rune {
	if w <= 0 {
		return []rune{}
	}

	sw := rw.StringWidth(s)
	if sw > w {
		return []rune(rw.Truncate(s, w, dot))
	}
	return str2runes(s)
}

func TrimStrIfAppropriate(s string, w int) string {
	if w <= 0 {
		return ""
	}

	sw := rw.StringWidth(s)
	if sw > w {
		return rw.Truncate(s, w, dot)
	}

	return s
}

func strWidth(s string) int {
	return rw.StringWidth(s)
}

func charWidth(ch rune) int {
	return rw.RuneWidth(ch)
}

var whiteSpaceRegex = regexp.MustCompile(`\s`)

func StringToAttribute(text string) Attribute {
	text = whiteSpaceRegex.ReplaceAllString(strings.ToLower(text), "")
	attributes := strings.Split(text, ",")
	result := Attribute(0)

	for _, theAttribute := range attributes {
		var match Attribute
		switch theAttribute {
		case "reset", "default":
			match = ColorDefault

		case "black":
			match = ColorBlack

		case "red":
			match = ColorRed

		case "green":
			match = ColorGreen

		case "yellow":
			match = ColorYellow

		case "blue":
			match = ColorBlue

		case "magenta":
			match = ColorMagenta

		case "cyan":
			match = ColorCyan

		case "white":
			match = ColorWhite

		case "bold":
			match = AttrBold

		case "underline":
			match = AttrUnderline

		case "reverse":
			match = AttrReverse
		}

		result |= match
	}

	return result
}

func TextCells(s string, fg, bg Attribute) []Cell {
	cs := make([]Cell, 0, len(s))

	runes := str2runes(s)

	for n := range runes {

		cs = append(cs, Cell{runes[n], fg, bg})
	}
	return cs
}

func (c Cell) Width() int {
	return charWidth(c.Ch)
}

func (c Cell) Copy() Cell {
	return c
}

func TrimTxCells(cs []Cell, w int) []Cell {
	if len(cs) <= w {
		return cs
	}
	return cs[:w]
}

func DTrimTxCls(cs []Cell, w int) []Cell {
	l := len(cs)
	if l <= 0 {
		return []Cell{}
	}

	rt := make([]Cell, 0, w)
	csw := 0
	for i := 0; i < l && csw <= w; i++ {
		c := cs[i]
		cw := c.Width()

		if cw+csw < w {
			rt = append(rt, c)
			csw += cw
		} else {
			rt = append(rt, Cell{'…', c.Fg, c.Bg})
			break
		}
	}

	return rt
}

func CellsToStr(cs []Cell) string {
	str := ""
	for _, c := range cs {
		str += string(c.Ch)
	}
	return str
}
