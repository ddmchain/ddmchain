
// +build solaris

package terminal 

import (
	"golang.org/x/sys/unix"
	"io"
	"syscall"
)

type State struct {
	state *unix.Termios
}

func IsTerminal(fd int) bool {
	_, err := unix.IoctlGetTermio(fd, unix.TCGETA)
	return err == nil
}

func ReadPassword(fd int) ([]byte, error) {

	val, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return nil, err
	}
	oldState := *val

	newState := oldState
	newState.Lflag &^= syscall.ECHO
	newState.Lflag |= syscall.ICANON | syscall.ISIG
	newState.Iflag |= syscall.ICRNL
	err = unix.IoctlSetTermios(fd, unix.TCSETS, &newState)
	if err != nil {
		return nil, err
	}

	defer unix.IoctlSetTermios(fd, unix.TCSETS, &oldState)

	var buf [16]byte
	var ret []byte
	for {
		n, err := syscall.Read(fd, buf[:])
		if err != nil {
			return nil, err
		}
		if n == 0 {
			if len(ret) == 0 {
				return nil, io.EOF
			}
			break
		}
		if buf[n-1] == '\n' {
			n--
		}
		ret = append(ret, buf[:n]...)
		if n < len(buf) {
			break
		}
	}

	return ret, nil
}

func MakeRaw(fd int) (*State, error) {
	oldTermiosPtr, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return nil, err
	}
	oldTermios := *oldTermiosPtr

	newTermios := oldTermios
	newTermios.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP | syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	newTermios.Oflag &^= syscall.OPOST
	newTermios.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	newTermios.Cflag &^= syscall.CSIZE | syscall.PARENB
	newTermios.Cflag |= syscall.CS8
	newTermios.Cc[unix.VMIN] = 1
	newTermios.Cc[unix.VTIME] = 0

	if err := unix.IoctlSetTermios(fd, unix.TCSETS, &newTermios); err != nil {
		return nil, err
	}

	return &State{
		state: oldTermiosPtr,
	}, nil
}

func Restore(fd int, oldState *State) error {
	return unix.IoctlSetTermios(fd, unix.TCSETS, oldState.state)
}

func GetState(fd int) (*State, error) {
	oldTermiosPtr, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return nil, err
	}

	return &State{
		state: oldTermiosPtr,
	}, nil
}

func GetSize(fd int) (width, height int, err error) {
	ws, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ)
	if err != nil {
		return 0, 0, err
	}
	return int(ws.Col), int(ws.Row), nil
}
