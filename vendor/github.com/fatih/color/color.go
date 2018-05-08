package color

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
)

var (

	NoColor = !isatty.IsTerminal(os.Stdout.Fd()) || os.Getenv("TERM") == "dumb"

	Output = colorable.NewColorableStdout()

	colorsCache   = make(map[Attribute]*Color)
	colorsCacheMu sync.Mutex 
)

type Color struct {
	params  []Attribute
	noColor *bool
}

type Attribute int

const escape = "\x1b"

const (
	Reset Attribute = iota
	Bold
	Faint
	Italic
	Underline
	BlinkSlow
	BlinkRapid
	ReverseVideo
	Concealed
	CrossedOut
)

const (
	FgBlack Attribute = iota + 30
	FgRed
	FgGreen
	FgYellow
	FgBlue
	FgMagenta
	FgCyan
	FgWhite
)

const (
	FgHiBlack Attribute = iota + 90
	FgHiRed
	FgHiGreen
	FgHiYellow
	FgHiBlue
	FgHiMagenta
	FgHiCyan
	FgHiWhite
)

const (
	BgBlack Attribute = iota + 40
	BgRed
	BgGreen
	BgYellow
	BgBlue
	BgMagenta
	BgCyan
	BgWhite
)

const (
	BgHiBlack Attribute = iota + 100
	BgHiRed
	BgHiGreen
	BgHiYellow
	BgHiBlue
	BgHiMagenta
	BgHiCyan
	BgHiWhite
)

func New(value ...Attribute) *Color {
	c := &Color{params: make([]Attribute, 0)}
	c.Add(value...)
	return c
}

func Set(p ...Attribute) *Color {
	c := New(p...)
	c.Set()
	return c
}

func Unset() {
	if NoColor {
		return
	}

	fmt.Fprintf(Output, "%s[%dm", escape, Reset)
}

func (c *Color) Set() *Color {
	if c.isNoColorSet() {
		return c
	}

	fmt.Fprintf(Output, c.format())
	return c
}

func (c *Color) unset() {
	if c.isNoColorSet() {
		return
	}

	Unset()
}

func (c *Color) setWriter(w io.Writer) *Color {
	if c.isNoColorSet() {
		return c
	}

	fmt.Fprintf(w, c.format())
	return c
}

func (c *Color) unsetWriter(w io.Writer) {
	if c.isNoColorSet() {
		return
	}

	if NoColor {
		return
	}

	fmt.Fprintf(w, "%s[%dm", escape, Reset)
}

func (c *Color) Add(value ...Attribute) *Color {
	c.params = append(c.params, value...)
	return c
}

func (c *Color) prepend(value Attribute) {
	c.params = append(c.params, 0)
	copy(c.params[1:], c.params[0:])
	c.params[0] = value
}

func (c *Color) Fprint(w io.Writer, a ...interface{}) (n int, err error) {
	c.setWriter(w)
	defer c.unsetWriter(w)

	return fmt.Fprint(w, a...)
}

func (c *Color) Print(a ...interface{}) (n int, err error) {
	c.Set()
	defer c.unset()

	return fmt.Fprint(Output, a...)
}

func (c *Color) Fprintf(w io.Writer, format string, a ...interface{}) (n int, err error) {
	c.setWriter(w)
	defer c.unsetWriter(w)

	return fmt.Fprintf(w, format, a...)
}

func (c *Color) Printf(format string, a ...interface{}) (n int, err error) {
	c.Set()
	defer c.unset()

	return fmt.Fprintf(Output, format, a...)
}

func (c *Color) Fprintln(w io.Writer, a ...interface{}) (n int, err error) {
	c.setWriter(w)
	defer c.unsetWriter(w)

	return fmt.Fprintln(w, a...)
}

func (c *Color) Println(a ...interface{}) (n int, err error) {
	c.Set()
	defer c.unset()

	return fmt.Fprintln(Output, a...)
}

func (c *Color) Sprint(a ...interface{}) string {
	return c.wrap(fmt.Sprint(a...))
}

func (c *Color) Sprintln(a ...interface{}) string {
	return c.wrap(fmt.Sprintln(a...))
}

func (c *Color) Sprintf(format string, a ...interface{}) string {
	return c.wrap(fmt.Sprintf(format, a...))
}

func (c *Color) FprintFunc() func(w io.Writer, a ...interface{}) {
	return func(w io.Writer, a ...interface{}) {
		c.Fprint(w, a...)
	}
}

func (c *Color) PrintFunc() func(a ...interface{}) {
	return func(a ...interface{}) {
		c.Print(a...)
	}
}

func (c *Color) FprintfFunc() func(w io.Writer, format string, a ...interface{}) {
	return func(w io.Writer, format string, a ...interface{}) {
		c.Fprintf(w, format, a...)
	}
}

func (c *Color) PrintfFunc() func(format string, a ...interface{}) {
	return func(format string, a ...interface{}) {
		c.Printf(format, a...)
	}
}

func (c *Color) FprintlnFunc() func(w io.Writer, a ...interface{}) {
	return func(w io.Writer, a ...interface{}) {
		c.Fprintln(w, a...)
	}
}

func (c *Color) PrintlnFunc() func(a ...interface{}) {
	return func(a ...interface{}) {
		c.Println(a...)
	}
}

func (c *Color) SprintFunc() func(a ...interface{}) string {
	return func(a ...interface{}) string {
		return c.wrap(fmt.Sprint(a...))
	}
}

func (c *Color) SprintfFunc() func(format string, a ...interface{}) string {
	return func(format string, a ...interface{}) string {
		return c.wrap(fmt.Sprintf(format, a...))
	}
}

func (c *Color) SprintlnFunc() func(a ...interface{}) string {
	return func(a ...interface{}) string {
		return c.wrap(fmt.Sprintln(a...))
	}
}

func (c *Color) sequence() string {
	format := make([]string, len(c.params))
	for i, v := range c.params {
		format[i] = strconv.Itoa(int(v))
	}

	return strings.Join(format, ";")
}

func (c *Color) wrap(s string) string {
	if c.isNoColorSet() {
		return s
	}

	return c.format() + s + c.unformat()
}

func (c *Color) format() string {
	return fmt.Sprintf("%s[%sm", escape, c.sequence())
}

func (c *Color) unformat() string {
	return fmt.Sprintf("%s[%dm", escape, Reset)
}

func (c *Color) DisableColor() {
	c.noColor = boolPtr(true)
}

func (c *Color) EnableColor() {
	c.noColor = boolPtr(false)
}

func (c *Color) isNoColorSet() bool {

	if c.noColor != nil {
		return *c.noColor
	}

	return NoColor
}

func (c *Color) Equals(c2 *Color) bool {
	if len(c.params) != len(c2.params) {
		return false
	}

	for _, attr := range c.params {
		if !c2.attrExists(attr) {
			return false
		}
	}

	return true
}

func (c *Color) attrExists(a Attribute) bool {
	for _, attr := range c.params {
		if attr == a {
			return true
		}
	}

	return false
}

func boolPtr(v bool) *bool {
	return &v
}

func getCachedColor(p Attribute) *Color {
	colorsCacheMu.Lock()
	defer colorsCacheMu.Unlock()

	c, ok := colorsCache[p]
	if !ok {
		c = New(p)
		colorsCache[p] = c
	}

	return c
}

func colorPrint(format string, p Attribute, a ...interface{}) {
	c := getCachedColor(p)

	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}

	if len(a) == 0 {
		c.Print(format)
	} else {
		c.Printf(format, a...)
	}
}

func colorString(format string, p Attribute, a ...interface{}) string {
	c := getCachedColor(p)

	if len(a) == 0 {
		return c.SprintFunc()(format)
	}

	return c.SprintfFunc()(format, a...)
}

func Black(format string, a ...interface{}) { colorPrint(format, FgBlack, a...) }

func Red(format string, a ...interface{}) { colorPrint(format, FgRed, a...) }

func Green(format string, a ...interface{}) { colorPrint(format, FgGreen, a...) }

func Yellow(format string, a ...interface{}) { colorPrint(format, FgYellow, a...) }

func Blue(format string, a ...interface{}) { colorPrint(format, FgBlue, a...) }

func Magenta(format string, a ...interface{}) { colorPrint(format, FgMagenta, a...) }

func Cyan(format string, a ...interface{}) { colorPrint(format, FgCyan, a...) }

func White(format string, a ...interface{}) { colorPrint(format, FgWhite, a...) }

func BlackString(format string, a ...interface{}) string { return colorString(format, FgBlack, a...) }

func RedString(format string, a ...interface{}) string { return colorString(format, FgRed, a...) }

func GreenString(format string, a ...interface{}) string { return colorString(format, FgGreen, a...) }

func YellowString(format string, a ...interface{}) string { return colorString(format, FgYellow, a...) }

func BlueString(format string, a ...interface{}) string { return colorString(format, FgBlue, a...) }

func MagentaString(format string, a ...interface{}) string {
	return colorString(format, FgMagenta, a...)
}

func CyanString(format string, a ...interface{}) string { return colorString(format, FgCyan, a...) }

func WhiteString(format string, a ...interface{}) string { return colorString(format, FgWhite, a...) }
