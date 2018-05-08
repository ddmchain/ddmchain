package parser

import (
	"fmt"
	"sort"

	"github.com/robertkrimen/otto/file"
	"github.com/robertkrimen/otto/token"
)

const (
	err_UnexpectedToken      = "Unexpected token %v"
	err_UnexpectedEndOfInput = "Unexpected end of input"
	err_UnexpectedEscape     = "Unexpected escape"
)

type Error struct {
	Position file.Position
	Message  string
}

func (self Error) Error() string {
	filename := self.Position.Filename
	if filename == "" {
		filename = "(anonymous)"
	}
	return fmt.Sprintf("%s: Line %d:%d %s",
		filename,
		self.Position.Line,
		self.Position.Column,
		self.Message,
	)
}

func (self *_parser) error(place interface{}, msg string, msgValues ...interface{}) *Error {
	idx := file.Idx(0)
	switch place := place.(type) {
	case int:
		idx = self.idxOf(place)
	case file.Idx:
		if place == 0 {
			idx = self.idxOf(self.chrOffset)
		} else {
			idx = place
		}
	default:
		panic(fmt.Errorf("error(%T, ...)", place))
	}

	position := self.position(idx)
	msg = fmt.Sprintf(msg, msgValues...)
	self.errors.Add(position, msg)
	return self.errors[len(self.errors)-1]
}

func (self *_parser) errorUnexpected(idx file.Idx, chr rune) error {
	if chr == -1 {
		return self.error(idx, err_UnexpectedEndOfInput)
	}
	return self.error(idx, err_UnexpectedToken, token.ILLEGAL)
}

func (self *_parser) errorUnexpectedToken(tkn token.Token) error {
	switch tkn {
	case token.EOF:
		return self.error(file.Idx(0), err_UnexpectedEndOfInput)
	}
	value := tkn.String()
	switch tkn {
	case token.BOOLEAN, token.NULL:
		value = self.literal
	case token.IDENTIFIER:
		return self.error(self.idx, "Unexpected identifier")
	case token.KEYWORD:

		return self.error(self.idx, "Unexpected reserved word")
	case token.NUMBER:
		return self.error(self.idx, "Unexpected number")
	case token.STRING:
		return self.error(self.idx, "Unexpected string")
	}
	return self.error(self.idx, err_UnexpectedToken, value)
}

type ErrorList []*Error

func (self *ErrorList) Add(position file.Position, msg string) {
	*self = append(*self, &Error{position, msg})
}

func (self *ErrorList) Reset() { *self = (*self)[0:0] }

func (self ErrorList) Len() int      { return len(self) }
func (self ErrorList) Swap(i, j int) { self[i], self[j] = self[j], self[i] }
func (self ErrorList) Less(i, j int) bool {
	x := &self[i].Position
	y := &self[j].Position
	if x.Filename < y.Filename {
		return true
	}
	if x.Filename == y.Filename {
		if x.Line < y.Line {
			return true
		}
		if x.Line == y.Line {
			return x.Column < y.Column
		}
	}
	return false
}

func (self ErrorList) Sort() {
	sort.Sort(self)
}

func (self ErrorList) Error() string {
	switch len(self) {
	case 0:
		return "no errors"
	case 1:
		return self[0].Error()
	}
	return fmt.Sprintf("%s (and %d more errors)", self[0].Error(), len(self)-1)
}

func (self ErrorList) Err() error {
	if len(self) == 0 {
		return nil
	}
	return self
}
