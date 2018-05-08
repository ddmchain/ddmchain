
package termui

var brailleBase = '\u2800'

var brailleOftMap = [4][2]rune{
	{'\u0001', '\u0008'},
	{'\u0002', '\u0010'},
	{'\u0004', '\u0020'},
	{'\u0040', '\u0080'}}

type Canvas map[[2]int]rune

func NewCanvas() Canvas {
	return make(map[[2]int]rune)
}

func chOft(x, y int) rune {
	return brailleOftMap[y%4][x%2]
}

func (c Canvas) rawCh(x, y int) rune {
	if ch, ok := c[[2]int{x, y}]; ok {
		return ch
	}
	return '\u0000' 
}

func chPos(x, y int) (int, int) {
	return y / 4, x / 2
}

func (c Canvas) Set(x, y int) {
	i, j := chPos(x, y)
	ch := c.rawCh(i, j)
	ch |= chOft(x, y)
	c[[2]int{i, j}] = ch
}

func (c Canvas) Unset(x, y int) {
	i, j := chPos(x, y)
	ch := c.rawCh(i, j)
	ch &= ^chOft(x, y)
	c[[2]int{i, j}] = ch
}

func (c Canvas) Buffer() Buffer {
	buf := NewBuffer()
	for k, v := range c {
		buf.Set(k[0], k[1], Cell{Ch: v + brailleBase})
	}
	return buf
}
