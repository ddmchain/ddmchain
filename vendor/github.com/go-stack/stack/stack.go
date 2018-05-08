
package stack

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strconv"
	"strings"
)

type Call struct {
	fn *runtime.Func
	pc uintptr
}

func Caller(skip int) Call {
	var pcs [2]uintptr
	n := runtime.Callers(skip+1, pcs[:])

	var c Call

	if n < 2 {
		return c
	}

	c.pc = pcs[1]
	if runtime.FuncForPC(pcs[0]).Name() != "runtime.sigpanic" {
		c.pc--
	}
	c.fn = runtime.FuncForPC(c.pc)
	return c
}

func (c Call) String() string {
	return fmt.Sprint(c)
}

func (c Call) MarshalText() ([]byte, error) {
	if c.fn == nil {
		return nil, ErrNoFunc
	}
	buf := bytes.Buffer{}
	fmt.Fprint(&buf, c)
	return buf.Bytes(), nil
}

var ErrNoFunc = errors.New("no call stack information")

func (c Call) Format(s fmt.State, verb rune) {
	if c.fn == nil {
		fmt.Fprintf(s, "%%!%c(NOFUNC)", verb)
		return
	}

	switch verb {
	case 's', 'v':
		file, line := c.fn.FileLine(c.pc)
		switch {
		case s.Flag('#'):

		case s.Flag('+'):
			file = file[pkgIndex(file, c.fn.Name()):]
		default:
			const sep = "/"
			if i := strings.LastIndex(file, sep); i != -1 {
				file = file[i+len(sep):]
			}
		}
		io.WriteString(s, file)
		if verb == 'v' {
			buf := [7]byte{':'}
			s.Write(strconv.AppendInt(buf[:1], int64(line), 10))
		}

	case 'd':
		_, line := c.fn.FileLine(c.pc)
		buf := [6]byte{}
		s.Write(strconv.AppendInt(buf[:0], int64(line), 10))

	case 'n':
		name := c.fn.Name()
		if !s.Flag('+') {
			const pathSep = "/"
			if i := strings.LastIndex(name, pathSep); i != -1 {
				name = name[i+len(pathSep):]
			}
			const pkgSep = "."
			if i := strings.Index(name, pkgSep); i != -1 {
				name = name[i+len(pkgSep):]
			}
		}
		io.WriteString(s, name)
	}
}

func (c Call) PC() uintptr {
	return c.pc
}

func (c Call) name() string {
	if c.fn == nil {
		return "???"
	}
	return c.fn.Name()
}

func (c Call) file() string {
	if c.fn == nil {
		return "???"
	}
	file, _ := c.fn.FileLine(c.pc)
	return file
}

func (c Call) line() int {
	if c.fn == nil {
		return 0
	}
	_, line := c.fn.FileLine(c.pc)
	return line
}

type CallStack []Call

func (cs CallStack) String() string {
	return fmt.Sprint(cs)
}

var (
	openBracketBytes  = []byte("[")
	closeBracketBytes = []byte("]")
	spaceBytes        = []byte(" ")
)

func (cs CallStack) MarshalText() ([]byte, error) {
	buf := bytes.Buffer{}
	buf.Write(openBracketBytes)
	for i, pc := range cs {
		if pc.fn == nil {
			return nil, ErrNoFunc
		}
		if i > 0 {
			buf.Write(spaceBytes)
		}
		fmt.Fprint(&buf, pc)
	}
	buf.Write(closeBracketBytes)
	return buf.Bytes(), nil
}

func (cs CallStack) Format(s fmt.State, verb rune) {
	s.Write(openBracketBytes)
	for i, pc := range cs {
		if i > 0 {
			s.Write(spaceBytes)
		}
		pc.Format(s, verb)
	}
	s.Write(closeBracketBytes)
}

func Trace() CallStack {
	var pcs [512]uintptr
	n := runtime.Callers(2, pcs[:])
	cs := make([]Call, n)

	for i, pc := range pcs[:n] {
		pcFix := pc
		if i > 0 && cs[i-1].fn.Name() != "runtime.sigpanic" {
			pcFix--
		}
		cs[i] = Call{
			fn: runtime.FuncForPC(pcFix),
			pc: pcFix,
		}
	}

	return cs
}

func (cs CallStack) TrimBelow(c Call) CallStack {
	for len(cs) > 0 && cs[0].pc != c.pc {
		cs = cs[1:]
	}
	return cs
}

func (cs CallStack) TrimAbove(c Call) CallStack {
	for len(cs) > 0 && cs[len(cs)-1].pc != c.pc {
		cs = cs[:len(cs)-1]
	}
	return cs
}

func pkgIndex(file, funcName string) int {

	const sep = "/"
	i := len(file)
	for n := strings.Count(funcName, sep) + 2; n > 0; n-- {
		i = strings.LastIndex(file[:i], sep)
		if i == -1 {
			i = -len(sep)
			break
		}
	}

	return i + len(sep)
}

var runtimePath string

func init() {
	var pcs [1]uintptr
	runtime.Callers(0, pcs[:])
	fn := runtime.FuncForPC(pcs[0])
	file, _ := fn.FileLine(pcs[0])

	idx := pkgIndex(file, fn.Name())

	runtimePath = file[:idx]
	if runtime.GOOS == "windows" {
		runtimePath = strings.ToLower(runtimePath)
	}
}

func inGoroot(c Call) bool {
	file := c.file()
	if len(file) == 0 || file[0] == '?' {
		return true
	}
	if runtime.GOOS == "windows" {
		file = strings.ToLower(file)
	}
	return strings.HasPrefix(file, runtimePath) || strings.HasSuffix(file, "/_testmain.go")
}

func (cs CallStack) TrimRuntime() CallStack {
	for len(cs) > 0 && inGoroot(cs[len(cs)-1]) {
		cs = cs[:len(cs)-1]
	}
	return cs
}
