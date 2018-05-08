package liner

import "unicode"

var zeroWidth = []*unicode.RangeTable{
	unicode.Mn,
	unicode.Me,
	unicode.Cc,
	unicode.Cf,
}

var doubleWidth = []*unicode.RangeTable{
	unicode.Han,
	unicode.Hangul,
	unicode.Hiragana,
	unicode.Katakana,
}

func countGlyphs(s []rune) int {
	n := 0
	for _, r := range s {

		if r < 127 {
			n++
			continue
		}

		switch {
		case unicode.IsOneOf(zeroWidth, r):
		case unicode.IsOneOf(doubleWidth, r):
			n += 2
		default:
			n++
		}
	}
	return n
}

func countMultiLineGlyphs(s []rune, columns int, start int) int {
	n := start
	for _, r := range s {
		if r < 127 {
			n++
			continue
		}
		switch {
		case unicode.IsOneOf(zeroWidth, r):
		case unicode.IsOneOf(doubleWidth, r):
			n += 2

			if n%columns == 1 {
				n++
			}
		default:
			n++
		}
	}
	return n
}

func getPrefixGlyphs(s []rune, num int) []rune {
	p := 0
	for n := 0; n < num && p < len(s); p++ {

		if s[p] < 127 {
			n++
			continue
		}
		if !unicode.IsOneOf(zeroWidth, s[p]) {
			n++
		}
	}
	for p < len(s) && unicode.IsOneOf(zeroWidth, s[p]) {
		p++
	}
	return s[:p]
}

func getSuffixGlyphs(s []rune, num int) []rune {
	p := len(s)
	for n := 0; n < num && p > 0; p-- {

		if s[p-1] < 127 {
			n++
			continue
		}
		if !unicode.IsOneOf(zeroWidth, s[p-1]) {
			n++
		}
	}
	return s[p:]
}
