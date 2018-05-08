
package liner

import (
	"bufio"
	"container/ring"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"unicode/utf8"
)

type commonState struct {
	terminalSupported bool
	outputRedirected  bool
	inputRedirected   bool
	history           []string
	historyMutex      sync.RWMutex
	completer         WordCompleter
	columns           int
	killRing          *ring.Ring
	ctrlCAborts       bool
	r                 *bufio.Reader
	tabStyle          TabStyle
	multiLineMode     bool
	cursorRows        int
	maxRows           int
	shouldRestart     ShouldRestart
	needRefresh       bool
}

type TabStyle int

const (
	TabCircular TabStyle = iota
	TabPrints
)

var ErrPromptAborted = errors.New("prompt aborted")

var ErrNotTerminalOutput = errors.New("standard output is not a terminal")

var ErrInvalidPrompt = errors.New("invalid prompt")

var ErrInternal = errors.New("liner: internal error")

const KillRingMax = 60

const HistoryLimit = 1000

func (s *State) ReadHistory(r io.Reader) (num int, err error) {
	s.historyMutex.Lock()
	defer s.historyMutex.Unlock()

	in := bufio.NewReader(r)
	num = 0
	for {
		line, part, err := in.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return num, err
		}
		if part {
			return num, fmt.Errorf("line %d is too long", num+1)
		}
		if !utf8.Valid(line) {
			return num, fmt.Errorf("invalid string at line %d", num+1)
		}
		num++
		s.history = append(s.history, string(line))
		if len(s.history) > HistoryLimit {
			s.history = s.history[1:]
		}
	}
	return num, nil
}

func (s *State) WriteHistory(w io.Writer) (num int, err error) {
	s.historyMutex.RLock()
	defer s.historyMutex.RUnlock()

	for _, item := range s.history {
		_, err := fmt.Fprintln(w, item)
		if err != nil {
			return num, err
		}
		num++
	}
	return num, nil
}

func (s *State) AppendHistory(item string) {
	s.historyMutex.Lock()
	defer s.historyMutex.Unlock()

	if len(s.history) > 0 {
		if item == s.history[len(s.history)-1] {
			return
		}
	}
	s.history = append(s.history, item)
	if len(s.history) > HistoryLimit {
		s.history = s.history[1:]
	}
}

func (s *State) ClearHistory() {
	s.historyMutex.Lock()
	defer s.historyMutex.Unlock()
	s.history = nil
}

func (s *State) getHistoryByPrefix(prefix string) (ph []string) {
	for _, h := range s.history {
		if strings.HasPrefix(h, prefix) {
			ph = append(ph, h)
		}
	}
	return
}

func (s *State) getHistoryByPattern(pattern string) (ph []string, pos []int) {
	if pattern == "" {
		return
	}
	for _, h := range s.history {
		if i := strings.Index(h, pattern); i >= 0 {
			ph = append(ph, h)
			pos = append(pos, i)
		}
	}
	return
}

type Completer func(line string) []string

type WordCompleter func(line string, pos int) (head string, completions []string, tail string)

func (s *State) SetCompleter(f Completer) {
	if f == nil {
		s.completer = nil
		return
	}
	s.completer = func(line string, pos int) (string, []string, string) {
		return "", f(string([]rune(line)[:pos])), string([]rune(line)[pos:])
	}
}

func (s *State) SetWordCompleter(f WordCompleter) {
	s.completer = f
}

func (s *State) SetTabCompletionStyle(tabStyle TabStyle) {
	s.tabStyle = tabStyle
}

type ModeApplier interface {
	ApplyMode() error
}

func (s *State) SetCtrlCAborts(aborts bool) {
	s.ctrlCAborts = aborts
}

func (s *State) SetMultiLineMode(mlmode bool) {
	s.multiLineMode = mlmode
}

type ShouldRestart func(err error) bool

func (s *State) SetShouldRestart(f ShouldRestart) {
	s.shouldRestart = f
}

func (s *State) promptUnsupported(p string) (string, error) {
	if !s.inputRedirected || !s.terminalSupported {
		fmt.Print(p)
	}
	linebuf, _, err := s.r.ReadLine()
	if err != nil {
		return "", err
	}
	return string(linebuf), nil
}
