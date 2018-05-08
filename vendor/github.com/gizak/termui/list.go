
package termui

import "strings"

type List struct {
	Block
	Items       []string
	Overflow    string
	ItemFgColor Attribute
	ItemBgColor Attribute
}

func NewList() *List {
	l := &List{Block: *NewBlock()}
	l.Overflow = "hidden"
	l.ItemFgColor = ThemeAttr("list.item.fg")
	l.ItemBgColor = ThemeAttr("list.item.bg")
	return l
}

func (l *List) Buffer() Buffer {
	buf := l.Block.Buffer()

	switch l.Overflow {
	case "wrap":
		cs := DefaultTxBuilder.Build(strings.Join(l.Items, "\n"), l.ItemFgColor, l.ItemBgColor)
		i, j, k := 0, 0, 0
		for i < l.innerArea.Dy() && k < len(cs) {
			w := cs[k].Width()
			if cs[k].Ch == '\n' || j+w > l.innerArea.Dx() {
				i++
				j = 0
				if cs[k].Ch == '\n' {
					k++
				}
				continue
			}
			buf.Set(l.innerArea.Min.X+j, l.innerArea.Min.Y+i, cs[k])

			k++
			j++
		}

	case "hidden":
		trimItems := l.Items
		if len(trimItems) > l.innerArea.Dy() {
			trimItems = trimItems[:l.innerArea.Dy()]
		}
		for i, v := range trimItems {
			cs := DTrimTxCls(DefaultTxBuilder.Build(v, l.ItemFgColor, l.ItemBgColor), l.innerArea.Dx())
			j := 0
			for _, vv := range cs {
				w := vv.Width()
				buf.Set(l.innerArea.Min.X+j, l.innerArea.Min.Y+i, vv)
				j += w
			}
		}
	}
	return buf
}
