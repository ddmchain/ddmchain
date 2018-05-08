// +build !windows,!linux,!darwin,!openbsd,!freebsd,!netbsd

package liner

import (
	"bufio"
	"errors"
	"os"
)

type State struct {
	commonState
}

func (s *State) Prompt(p string) (string, error) {
	return s.promptUnsupported(p)
}

func (s *State) PasswordPrompt(p string) (string, error) {
	return "", errors.New("liner: function not supported in this terminal")
}

func NewLiner() *State {
	var s State
	s.r = bufio.NewReader(os.Stdin)
	return &s
}

func (s *State) Close() error {
	return nil
}

func TerminalSupported() bool {
	return false
}

type noopMode struct{}

func (n noopMode) ApplyMode() error {
	return nil
}

func TerminalMode() (ModeApplier, error) {
	return noopMode{}, nil
}
