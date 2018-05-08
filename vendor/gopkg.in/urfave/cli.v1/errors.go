package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
)

var OsExiter = os.Exit

var ErrWriter io.Writer = os.Stderr

type MultiError struct {
	Errors []error
}

func NewMultiError(err ...error) MultiError {
	return MultiError{Errors: err}
}

func (m MultiError) Error() string {
	errs := make([]string, len(m.Errors))
	for i, err := range m.Errors {
		errs[i] = err.Error()
	}

	return strings.Join(errs, "\n")
}

type ErrorFormatter interface {
	Format(s fmt.State, verb rune)
}

type ExitCoder interface {
	error
	ExitCode() int
}

type ExitError struct {
	exitCode int
	message  interface{}
}

func NewExitError(message interface{}, exitCode int) *ExitError {
	return &ExitError{
		exitCode: exitCode,
		message:  message,
	}
}

func (ee *ExitError) Error() string {
	return fmt.Sprintf("%v", ee.message)
}

func (ee *ExitError) ExitCode() int {
	return ee.exitCode
}

func HandleExitCoder(err error) {
	if err == nil {
		return
	}

	if exitErr, ok := err.(ExitCoder); ok {
		if err.Error() != "" {
			if _, ok := exitErr.(ErrorFormatter); ok {
				fmt.Fprintf(ErrWriter, "%+v\n", err)
			} else {
				fmt.Fprintln(ErrWriter, err)
			}
		}
		OsExiter(exitErr.ExitCode())
		return
	}

	if multiErr, ok := err.(MultiError); ok {
		code := handleMultiError(multiErr)
		OsExiter(code)
		return
	}
}

func handleMultiError(multiErr MultiError) int {
	code := 1
	for _, merr := range multiErr.Errors {
		if multiErr2, ok := merr.(MultiError); ok {
			code = handleMultiError(multiErr2)
		} else {
			fmt.Fprintln(ErrWriter, merr)
			if exitErr, ok := merr.(ExitCoder); ok {
				code = exitErr.ExitCode()
			}
		}
	}
	return code
}
