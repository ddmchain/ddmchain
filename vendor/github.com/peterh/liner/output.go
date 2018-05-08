// +build linux darwin openbsd freebsd netbsd

package liner

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

func (s *State) cursorPos(x int) {
	if s.useCHA {

		fmt.Printf("\x1b[%dG", x+1)
	} else {

		fmt.Print("\r")
		if x > 0 {
			fmt.Printf("\x1b[%dC", x)
		}
	}
}

func (s *State) eraseLine() {
	fmt.Print("\x1b[0K")
}

func (s *State) eraseScreen() {
	fmt.Print("\x1b[H\x1b[2J")
}

func (s *State) moveUp(lines int) {
	fmt.Printf("\x1b[%dA", lines)
}

func (s *State) moveDown(lines int) {
	fmt.Printf("\x1b[%dB", lines)
}

func (s *State) emitNewLine() {
	fmt.Print("\n")
}

type winSize struct {
	row, col       uint16
	xpixel, ypixel uint16
}

func (s *State) getColumns() bool {
	var ws winSize
	ok, _, _ := syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdout),
		syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&ws)))
	if int(ok) < 0 {
		return false
	}
	s.columns = int(ws.col)
	if cursorColumn && s.columns > 1 {
		s.columns--
	}
	return true
}

func (s *State) checkOutput() {

	if strings.Contains(strings.ToLower(os.Getenv("TERM")), "xterm") {
		s.useCHA = true
		return
	}

	s.useCHA = false
}
