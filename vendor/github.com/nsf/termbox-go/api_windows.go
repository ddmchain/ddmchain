package termbox

import (
	"syscall"
)

func Init() error {
	var err error

	interrupt, err = create_event()
	if err != nil {
		return err
	}

	in, err = syscall.Open("CONIN$", syscall.O_RDWR, 0)
	if err != nil {
		return err
	}
	out, err = syscall.Open("CONOUT$", syscall.O_RDWR, 0)
	if err != nil {
		return err
	}

	err = get_console_mode(in, &orig_mode)
	if err != nil {
		return err
	}

	err = set_console_mode(in, enable_window_input)
	if err != nil {
		return err
	}

	orig_size = get_term_size(out)
	win_size := get_win_size(out)

	err = set_console_screen_buffer_size(out, win_size)
	if err != nil {
		return err
	}

	err = get_console_cursor_info(out, &orig_cursor_info)
	if err != nil {
		return err
	}

	show_cursor(false)
	term_size = get_term_size(out)
	back_buffer.init(int(term_size.x), int(term_size.y))
	front_buffer.init(int(term_size.x), int(term_size.y))
	back_buffer.clear()
	front_buffer.clear()
	clear()

	diffbuf = make([]diff_msg, 0, 32)

	go input_event_producer()
	IsInit = true
	return nil
}

func Close() {

	Clear(0, 0)
	Flush()

	cancel_comm <- true
	set_event(interrupt)
	select {
	case <-input_comm:
	default:
	}
	<-cancel_done_comm

	set_console_cursor_info(out, &orig_cursor_info)
	set_console_cursor_position(out, coord{})
	set_console_screen_buffer_size(out, orig_size)
	set_console_mode(in, orig_mode)
	syscall.Close(in)
	syscall.Close(out)
	syscall.Close(interrupt)
	IsInit = false
}

func Interrupt() {
	interrupt_comm <- struct{}{}
}

func Flush() error {
	update_size_maybe()
	prepare_diff_messages()
	for _, diff := range diffbuf {
		r := small_rect{
			left:   0,
			top:    diff.pos,
			right:  term_size.x - 1,
			bottom: diff.pos + diff.lines - 1,
		}
		write_console_output(out, diff.chars, r)
	}
	if !is_cursor_hidden(cursor_x, cursor_y) {
		move_cursor(cursor_x, cursor_y)
	}
	return nil
}

func SetCursor(x, y int) {
	if is_cursor_hidden(cursor_x, cursor_y) && !is_cursor_hidden(x, y) {
		show_cursor(true)
	}

	if !is_cursor_hidden(cursor_x, cursor_y) && is_cursor_hidden(x, y) {
		show_cursor(false)
	}

	cursor_x, cursor_y = x, y
	if !is_cursor_hidden(cursor_x, cursor_y) {
		move_cursor(cursor_x, cursor_y)
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

func PollEvent() Event {
	select {
	case ev := <-input_comm:
		return ev
	case <-interrupt_comm:
		return Event{Type: EventInterrupt}
	}
}

func Size() (int, int) {
	return int(term_size.x), int(term_size.y)
}

func Clear(fg, bg Attribute) error {
	foreground, background = fg, bg
	update_size_maybe()
	back_buffer.clear()
	return nil
}

func SetInputMode(mode InputMode) InputMode {
	if mode == InputCurrent {
		return input_mode
	}
	if mode&InputMouse != 0 {
		err := set_console_mode(in, enable_window_input|enable_mouse_input|enable_extended_flags)
		if err != nil {
			panic(err)
		}
	} else {
		err := set_console_mode(in, enable_window_input)
		if err != nil {
			panic(err)
		}
	}

	input_mode = mode
	return input_mode
}

func SetOutputMode(mode OutputMode) OutputMode {
	return OutputNormal
}

func Sync() error {
	return nil
}
