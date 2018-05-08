// +build !windows

package termbox

import "github.com/mattn/go-runewidth"
import "fmt"
import "os"
import "os/signal"
import "syscall"
import "runtime"

func Init() error {
	var err error

	out, err = os.OpenFile("/dev/tty", syscall.O_WRONLY, 0)
	if err != nil {
		return err
	}
	in, err = syscall.Open("/dev/tty", syscall.O_RDONLY, 0)
	if err != nil {
		return err
	}

	err = setup_term()
	if err != nil {
		return fmt.Errorf("termbox: error while reading terminfo data: %v", err)
	}

	signal.Notify(sigwinch, syscall.SIGWINCH)
	signal.Notify(sigio, syscall.SIGIO)

	_, err = fcntl(in, syscall.F_SETFL, syscall.O_ASYNC|syscall.O_NONBLOCK)
	if err != nil {
		return err
	}
	_, err = fcntl(in, syscall.F_SETOWN, syscall.Getpid())
	if runtime.GOOS != "darwin" && err != nil {
		return err
	}
	err = tcgetattr(out.Fd(), &orig_tios)
	if err != nil {
		return err
	}

	tios := orig_tios
	tios.Iflag &^= syscall_IGNBRK | syscall_BRKINT | syscall_PARMRK |
		syscall_ISTRIP | syscall_INLCR | syscall_IGNCR |
		syscall_ICRNL | syscall_IXON
	tios.Lflag &^= syscall_ECHO | syscall_ECHONL | syscall_ICANON |
		syscall_ISIG | syscall_IEXTEN
	tios.Cflag &^= syscall_CSIZE | syscall_PARENB
	tios.Cflag |= syscall_CS8
	tios.Cc[syscall_VMIN] = 1
	tios.Cc[syscall_VTIME] = 0

	err = tcsetattr(out.Fd(), &tios)
	if err != nil {
		return err
	}

	out.WriteString(funcs[t_enter_ca])
	out.WriteString(funcs[t_enter_keypad])
	out.WriteString(funcs[t_hide_cursor])
	out.WriteString(funcs[t_clear_screen])

	termw, termh = get_term_size(out.Fd())
	back_buffer.init(termw, termh)
	front_buffer.init(termw, termh)
	back_buffer.clear()
	front_buffer.clear()

	go func() {
		buf := make([]byte, 128)
		for {
			select {
			case <-sigio:
				for {
					n, err := syscall.Read(in, buf)
					if err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
						break
					}
					select {
					case input_comm <- input_event{buf[:n], err}:
						ie := <-input_comm
						buf = ie.data[:128]
					case <-quit:
						return
					}
				}
			case <-quit:
				return
			}
		}
	}()

	IsInit = true
	return nil
}

func Interrupt() {
	interrupt_comm <- struct{}{}
}

func Close() {
	quit <- 1
	out.WriteString(funcs[t_show_cursor])
	out.WriteString(funcs[t_sgr0])
	out.WriteString(funcs[t_clear_screen])
	out.WriteString(funcs[t_exit_ca])
	out.WriteString(funcs[t_exit_keypad])
	out.WriteString(funcs[t_exit_mouse])
	tcsetattr(out.Fd(), &orig_tios)

	out.Close()
	syscall.Close(in)

	termw = 0
	termh = 0
	input_mode = InputEsc
	out = nil
	in = 0
	lastfg = attr_invalid
	lastbg = attr_invalid
	lastx = coord_invalid
	lasty = coord_invalid
	cursor_x = cursor_hidden
	cursor_y = cursor_hidden
	foreground = ColorDefault
	background = ColorDefault
	IsInit = false
}

func Flush() error {

	lastx = coord_invalid
	lasty = coord_invalid

	update_size_maybe()

	for y := 0; y < front_buffer.height; y++ {
		line_offset := y * front_buffer.width
		for x := 0; x < front_buffer.width; {
			cell_offset := line_offset + x
			back := &back_buffer.cells[cell_offset]
			front := &front_buffer.cells[cell_offset]
			if back.Ch < ' ' {
				back.Ch = ' '
			}
			w := runewidth.RuneWidth(back.Ch)
			if w == 0 || w == 2 && runewidth.IsAmbiguousWidth(back.Ch) {
				w = 1
			}
			if *back == *front {
				x += w
				continue
			}
			*front = *back
			send_attr(back.Fg, back.Bg)

			if w == 2 && x == front_buffer.width-1 {

				send_char(x, y, ' ')
			} else {
				send_char(x, y, back.Ch)
				if w == 2 {
					next := cell_offset + 1
					front_buffer.cells[next] = Cell{
						Ch: 0,
						Fg: back.Fg,
						Bg: back.Bg,
					}
				}
			}
			x += w
		}
	}
	if !is_cursor_hidden(cursor_x, cursor_y) {
		write_cursor(cursor_x, cursor_y)
	}
	return flush()
}

func SetCursor(x, y int) {
	if is_cursor_hidden(cursor_x, cursor_y) && !is_cursor_hidden(x, y) {
		outbuf.WriteString(funcs[t_show_cursor])
	}

	if !is_cursor_hidden(cursor_x, cursor_y) && is_cursor_hidden(x, y) {
		outbuf.WriteString(funcs[t_hide_cursor])
	}

	cursor_x, cursor_y = x, y
	if !is_cursor_hidden(cursor_x, cursor_y) {
		write_cursor(cursor_x, cursor_y)
	}
}

func HideCursor() {
	SetCursor(cursor_hidden, cursor_hidden)
}

func SetCell(x, y int, ch rune, fg, bg Attribute) {
	if x < 0 || x >= back_buffer.width {
		return
	}
	if y < 0 || y >= back_buffer.height {
		return
	}

	back_buffer.cells[y*back_buffer.width+x] = Cell{ch, fg, bg}
}

func CellBuffer() []Cell {
	return back_buffer.cells
}

func ParseEvent(data []byte) Event {
	event := Event{Type: EventKey}
	ok := extract_event(data, &event)
	if !ok {
		return Event{Type: EventNone, N: event.N}
	}
	return event
}

func PollRawEvent(data []byte) Event {
	if len(data) == 0 {
		panic("len(data) >= 1 is a requirement")
	}

	var event Event
	if extract_raw_event(data, &event) {
		return event
	}

	for {
		select {
		case ev := <-input_comm:
			if ev.err != nil {
				return Event{Type: EventError, Err: ev.err}
			}

			inbuf = append(inbuf, ev.data...)
			input_comm <- ev
			if extract_raw_event(data, &event) {
				return event
			}
		case <-interrupt_comm:
			event.Type = EventInterrupt
			return event

		case <-sigwinch:
			event.Type = EventResize
			event.Width, event.Height = get_term_size(out.Fd())
			return event
		}
	}
}

func PollEvent() Event {
	var event Event

	event.Type = EventKey
	ok := extract_event(inbuf, &event)
	if event.N != 0 {
		copy(inbuf, inbuf[event.N:])
		inbuf = inbuf[:len(inbuf)-event.N]
	}
	if ok {
		return event
	}

	for {
		select {
		case ev := <-input_comm:
			if ev.err != nil {
				return Event{Type: EventError, Err: ev.err}
			}

			inbuf = append(inbuf, ev.data...)
			input_comm <- ev
			ok := extract_event(inbuf, &event)
			if event.N != 0 {
				copy(inbuf, inbuf[event.N:])
				inbuf = inbuf[:len(inbuf)-event.N]
			}
			if ok {
				return event
			}
		case <-interrupt_comm:
			event.Type = EventInterrupt
			return event

		case <-sigwinch:
			event.Type = EventResize
			event.Width, event.Height = get_term_size(out.Fd())
			return event
		}
	}
}

func Size() (width int, height int) {
	return termw, termh
}

func Clear(fg, bg Attribute) error {
	foreground, background = fg, bg
	err := update_size_maybe()
	back_buffer.clear()
	return err
}

func SetInputMode(mode InputMode) InputMode {
	if mode == InputCurrent {
		return input_mode
	}
	if mode&(InputEsc|InputAlt) == 0 {
		mode |= InputEsc
	}
	if mode&(InputEsc|InputAlt) == InputEsc|InputAlt {
		mode &^= InputAlt
	}
	if mode&InputMouse != 0 {
		out.WriteString(funcs[t_enter_mouse])
	} else {
		out.WriteString(funcs[t_exit_mouse])
	}

	input_mode = mode
	return input_mode
}

func SetOutputMode(mode OutputMode) OutputMode {
	if mode == OutputCurrent {
		return output_mode
	}

	output_mode = mode
	return output_mode
}

func Sync() error {
	front_buffer.clear()
	err := send_clear()
	if err != nil {
		return err
	}

	return Flush()
}
