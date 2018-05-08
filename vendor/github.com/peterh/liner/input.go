// +build linux darwin openbsd freebsd netbsd

package liner

import (
	"bufio"
	"errors"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type nexter struct {
	r   rune
	err error
}

type State struct {
	commonState
	origMode    termios
	defaultMode termios
	next        <-chan nexter
	winch       chan os.Signal
	pending     []rune
	useCHA      bool
}

func NewLiner() *State {
	var s State
	s.r = bufio.NewReader(os.Stdin)

	s.terminalSupported = TerminalSupported()
	if m, err := TerminalMode(); err == nil {
		s.origMode = *m.(*termios)
	} else {
		s.inputRedirected = true
	}
	if _, err := getMode(syscall.Stdout); err != 0 {
		s.outputRedirected = true
	}
	if s.inputRedirected && s.outputRedirected {
		s.terminalSupported = false
	}
	if s.terminalSupported && !s.inputRedirected && !s.outputRedirected {
		mode := s.origMode
		mode.Iflag &^= icrnl | inpck | istrip | ixon
		mode.Cflag |= cs8
		mode.Lflag &^= syscall.ECHO | icanon | iexten
		mode.ApplyMode()

		winch := make(chan os.Signal, 1)
		signal.Notify(winch, syscall.SIGWINCH)
		s.winch = winch

		s.checkOutput()
	}

	if !s.outputRedirected {
		s.outputRedirected = !s.getColumns()
	}

	return &s
}

var errTimedOut = errors.New("timeout")

func (s *State) startPrompt() {
	if s.terminalSupported {
		if m, err := TerminalMode(); err == nil {
			s.defaultMode = *m.(*termios)
			mode := s.defaultMode
			mode.Lflag &^= isig
			mode.ApplyMode()
		}
	}
	s.restartPrompt()
}

func (s *State) inputWaiting() bool {
	return len(s.next) > 0
}

func (s *State) restartPrompt() {
	next := make(chan nexter, 200)
	go func() {
		for {
			var n nexter
			n.r, _, n.err = s.r.ReadRune()
			next <- n

			if n.err != nil || n.r == '\n' || n.r == '\r' || n.r == ctrlC || n.r == ctrlD {
				close(next)
				return
			}
		}
	}()
	s.next = next
}

func (s *State) stopPrompt() {
	if s.terminalSupported {
		s.defaultMode.ApplyMode()
	}
}

func (s *State) nextPending(timeout <-chan time.Time) (rune, error) {
	select {
	case thing, ok := <-s.next:
		if !ok {
			return 0, ErrInternal
		}
		if thing.err != nil {
			return 0, thing.err
		}
		s.pending = append(s.pending, thing.r)
		return thing.r, nil
	case <-timeout:
		rv := s.pending[0]
		s.pending = s.pending[1:]
		return rv, errTimedOut
	}
}

func (s *State) readNext() (interface{}, error) {
	if len(s.pending) > 0 {
		rv := s.pending[0]
		s.pending = s.pending[1:]
		return rv, nil
	}
	var r rune
	select {
	case thing, ok := <-s.next:
		if !ok {
			return 0, ErrInternal
		}
		if thing.err != nil {
			return nil, thing.err
		}
		r = thing.r
	case <-s.winch:
		s.getColumns()
		return winch, nil
	}
	if r != esc {
		return r, nil
	}
	s.pending = append(s.pending, r)

	timeout := time.After(50 * time.Millisecond)
	flag, err := s.nextPending(timeout)
	if err != nil {
		if err == errTimedOut {
			return flag, nil
		}
		return unknown, err
	}

	switch flag {
	case '[':
		code, err := s.nextPending(timeout)
		if err != nil {
			if err == errTimedOut {
				return code, nil
			}
			return unknown, err
		}
		switch code {
		case 'A':
			s.pending = s.pending[:0] 
			return up, nil
		case 'B':
			s.pending = s.pending[:0] 
			return down, nil
		case 'C':
			s.pending = s.pending[:0] 
			return right, nil
		case 'D':
			s.pending = s.pending[:0] 
			return left, nil
		case 'F':
			s.pending = s.pending[:0] 
			return end, nil
		case 'H':
			s.pending = s.pending[:0] 
			return home, nil
		case 'Z':
			s.pending = s.pending[:0] 
			return shiftTab, nil
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			num := []rune{code}
			for {
				code, err := s.nextPending(timeout)
				if err != nil {
					if err == errTimedOut {
						return code, nil
					}
					return nil, err
				}
				switch code {
				case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
					num = append(num, code)
				case ';':

					x, _ := strconv.ParseInt(string(num), 10, 32)
					if x != 1 {

						rv := s.pending[0]
						s.pending = s.pending[1:]
						return rv, nil
					}
					num = num[:0]
					for {
						code, err = s.nextPending(timeout)
						if err != nil {
							if err == errTimedOut {
								rv := s.pending[0]
								s.pending = s.pending[1:]
								return rv, nil
							}
							return nil, err
						}
						switch code {
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							num = append(num, code)
						case 'C', 'D':

							mod, _ := strconv.ParseInt(string(num), 10, 32)
							if mod != 5 {

								rv := s.pending[0]
								s.pending = s.pending[1:]
								return rv, nil
							}
							s.pending = s.pending[:0] 
							if code == 'C' {
								return wordRight, nil
							}
							return wordLeft, nil
						default:

							rv := s.pending[0]
							s.pending = s.pending[1:]
							return rv, nil
						}
					}
				case '~':
					s.pending = s.pending[:0] 
					x, _ := strconv.ParseInt(string(num), 10, 32)
					switch x {
					case 2:
						return insert, nil
					case 3:
						return del, nil
					case 5:
						return pageUp, nil
					case 6:
						return pageDown, nil
					case 7:
						return home, nil
					case 8:
						return end, nil
					case 15:
						return f5, nil
					case 17:
						return f6, nil
					case 18:
						return f7, nil
					case 19:
						return f8, nil
					case 20:
						return f9, nil
					case 21:
						return f10, nil
					case 23:
						return f11, nil
					case 24:
						return f12, nil
					default:
						return unknown, nil
					}
				default:

					rv := s.pending[0]
					s.pending = s.pending[1:]
					return rv, nil
				}
			}
		}

	case 'O':
		code, err := s.nextPending(timeout)
		if err != nil {
			if err == errTimedOut {
				return code, nil
			}
			return nil, err
		}
		s.pending = s.pending[:0] 
		switch code {
		case 'c':
			return wordRight, nil
		case 'd':
			return wordLeft, nil
		case 'H':
			return home, nil
		case 'F':
			return end, nil
		case 'P':
			return f1, nil
		case 'Q':
			return f2, nil
		case 'R':
			return f3, nil
		case 'S':
			return f4, nil
		default:
			return unknown, nil
		}
	case 'b':
		s.pending = s.pending[:0] 
		return altB, nil
	case 'f':
		s.pending = s.pending[:0] 
		return altF, nil
	case 'y':
		s.pending = s.pending[:0] 
		return altY, nil
	default:
		rv := s.pending[0]
		s.pending = s.pending[1:]
		return rv, nil
	}

	return r, nil
}

func (s *State) Close() error {
	signal.Stop(s.winch)
	if !s.inputRedirected {
		s.origMode.ApplyMode()
	}
	return nil
}

func TerminalSupported() bool {
	bad := map[string]bool{"": true, "dumb": true, "cons25": true}
	return !bad[strings.ToLower(os.Getenv("TERM"))]
}
