
package termui

import (
	"image"
	"io"
	"sync"
	"time"

	"fmt"

	"os"

	"runtime/debug"

	"bytes"

	"github.com/maruel/panicparse/stack"
	tm "github.com/nsf/termbox-go"
)

type Bufferer interface {
	Buffer() Buffer
}

func Init() error {
	if err := tm.Init(); err != nil {
		return err
	}

	sysEvtChs = make([]chan Event, 0)
	go hookTermboxEvt()

	renderJobs = make(chan []Bufferer)

	Body = NewGrid()
	Body.X = 0
	Body.Y = 0
	Body.BgColor = ThemeAttr("bg")
	Body.Width = TermWidth()

	DefaultEvtStream.Init()
	DefaultEvtStream.Merge("termbox", NewSysEvtCh())
	DefaultEvtStream.Merge("timer", NewTimerCh(time.Second))
	DefaultEvtStream.Merge("custom", usrEvtCh)

	DefaultEvtStream.Handle("/", DefaultHandler)
	DefaultEvtStream.Handle("/sys/wnd/resize", func(e Event) {
		w := e.Data.(EvtWnd)
		Body.Width = w.Width
	})

	DefaultWgtMgr = NewWgtMgr()
	DefaultEvtStream.Hook(DefaultWgtMgr.WgtHandlersHook())

	go func() {
		for bs := range renderJobs {
			render(bs...)
		}
	}()

	return nil
}

func Close() {
	tm.Close()
}

var renderLock sync.Mutex

func termSync() {
	renderLock.Lock()
	tm.Sync()
	termWidth, termHeight = tm.Size()
	renderLock.Unlock()
}

func TermWidth() int {
	termSync()
	return termWidth
}

func TermHeight() int {
	termSync()
	return termHeight
}

func render(bs ...Bufferer) {
	defer func() {
		if e := recover(); e != nil {
			Close()
			fmt.Fprintf(os.Stderr, "Captured a panic(value=%v) when rendering Bufferer. Exit termui and clean terminal...\nPrint stack trace:\n\n", e)

			gs, err := stack.ParseDump(bytes.NewReader(debug.Stack()), os.Stderr)
			if err != nil {
				debug.PrintStack()
				os.Exit(1)
			}
			p := &stack.Palette{}
			buckets := stack.SortBuckets(stack.Bucketize(gs, stack.AnyValue))
			srcLen, pkgLen := stack.CalcLengths(buckets, false)
			for _, bucket := range buckets {
				io.WriteString(os.Stdout, p.BucketHeader(&bucket, false, len(buckets) > 1))
				io.WriteString(os.Stdout, p.StackLines(&bucket.Signature, srcLen, pkgLen, false))
			}
			os.Exit(1)
		}
	}()
	for _, b := range bs {

		buf := b.Buffer()

		for p, c := range buf.CellMap {
			if p.In(buf.Area) {

				tm.SetCell(p.X, p.Y, c.Ch, toTmAttr(c.Fg), toTmAttr(c.Bg))

			}
		}

	}

	renderLock.Lock()

	tm.Flush()
	renderLock.Unlock()
}

func Clear() {
	tm.Clear(tm.ColorDefault, toTmAttr(ThemeAttr("bg")))
}

func clearArea(r image.Rectangle, bg Attribute) {
	for i := r.Min.X; i < r.Max.X; i++ {
		for j := r.Min.Y; j < r.Max.Y; j++ {
			tm.SetCell(i, j, ' ', tm.ColorDefault, toTmAttr(bg))
		}
	}
}

func ClearArea(r image.Rectangle, bg Attribute) {
	clearArea(r, bg)
	tm.Flush()
}

var renderJobs chan []Bufferer

func Render(bs ...Bufferer) {

	renderJobs <- bs
}
