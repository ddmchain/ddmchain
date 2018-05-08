package parser

import (
	"bytes"
	"fmt"
	"strconv"
)

type _RegExp_parser struct {
	str    string
	length int

	chr       rune 
	chrOffset int  
	offset    int  

	errors  []error
	invalid bool 

	goRegexp *bytes.Buffer
}

func TransformRegExp(pattern string) (string, error) {

	if pattern == "" {
		return "", nil
	}

	parser := _RegExp_parser{
		str:      pattern,
		length:   len(pattern),
		goRegexp: bytes.NewBuffer(make([]byte, 0, 3*len(pattern)/2)),
	}
	parser.read() 
	parser.scan()
	var err error
	if len(parser.errors) > 0 {
		err = parser.errors[0]
	}
	if parser.invalid {
		return "", err
	}

	return parser.goRegexp.String(), err
}

func (self *_RegExp_parser) scan() {
	for self.chr != -1 {
		switch self.chr {
		case '\\':
			self.read()
			self.scanEscape(false)
		case '(':
			self.pass()
			self.scanGroup()
		case '[':
			self.pass()
			self.scanBracket()
		case ')':
			self.error(-1, "Unmatched ')'")
			self.invalid = true
			self.pass()
		default:
			self.pass()
		}
	}
}

func (self *_RegExp_parser) scanGroup() {
	str := self.str[self.chrOffset:]
	if len(str) > 1 { 
		if str[0] == '?' {
			if str[1] == '=' || str[1] == '!' {
				self.error(-1, "re2: Invalid (%s) <lookahead>", self.str[self.chrOffset:self.chrOffset+2])
			}
		}
	}
	for self.chr != -1 && self.chr != ')' {
		switch self.chr {
		case '\\':
			self.read()
			self.scanEscape(false)
		case '(':
			self.pass()
			self.scanGroup()
		case '[':
			self.pass()
			self.scanBracket()
		default:
			self.pass()
			continue
		}
	}
	if self.chr != ')' {
		self.error(-1, "Unterminated group")
		self.invalid = true
		return
	}
	self.pass()
}

func (self *_RegExp_parser) scanBracket() {
	for self.chr != -1 {
		if self.chr == ']' {
			break
		} else if self.chr == '\\' {
			self.read()
			self.scanEscape(true)
			continue
		}
		self.pass()
	}
	if self.chr != ']' {
		self.error(-1, "Unterminated character class")
		self.invalid = true
		return
	}
	self.pass()
}

func (self *_RegExp_parser) scanEscape(inClass bool) {
	offset := self.chrOffset

	var length, base uint32
	switch self.chr {

	case '0', '1', '2', '3', '4', '5', '6', '7':
		var value int64
		size := 0
		for {
			digit := int64(digitValue(self.chr))
			if digit >= 8 {

				break
			}
			value = value*8 + digit
			self.read()
			size += 1
		}
		if size == 1 { 
			_, err := self.goRegexp.Write([]byte{'\\', byte(value) + '0'})
			if err != nil {
				self.errors = append(self.errors, err)
			}
			if value != 0 {

				self.error(-1, "re2: Invalid \\%d <backreference>", value)
			}
			return
		}
		tmp := []byte{'\\', 'x', '0', 0}
		if value >= 16 {
			tmp = tmp[0:2]
		} else {
			tmp = tmp[0:3]
		}
		tmp = strconv.AppendInt(tmp, value, 16)
		_, err := self.goRegexp.Write(tmp)
		if err != nil {
			self.errors = append(self.errors, err)
		}
		return

	case '8', '9':
		size := 0
		for {
			digit := digitValue(self.chr)
			if digit >= 10 {

				break
			}
			self.read()
			size += 1
		}
		err := self.goRegexp.WriteByte('\\')
		if err != nil {
			self.errors = append(self.errors, err)
		}
		_, err = self.goRegexp.WriteString(self.str[offset:self.chrOffset])
		if err != nil {
			self.errors = append(self.errors, err)
		}
		self.error(-1, "re2: Invalid \\%s <backreference>", self.str[offset:self.chrOffset])
		return

	case 'x':
		self.read()
		length, base = 2, 16

	case 'u':
		self.read()
		length, base = 4, 16

	case 'b':
		if inClass {
			_, err := self.goRegexp.Write([]byte{'\\', 'x', '0', '8'})
			if err != nil {
				self.errors = append(self.errors, err)
			}
			self.read()
			return
		}
		fallthrough

	case 'B':
		fallthrough

	case 'd', 'D', 's', 'S', 'w', 'W':

		fallthrough

	case '\\':
		fallthrough

	case 'f', 'n', 'r', 't', 'v':
		err := self.goRegexp.WriteByte('\\')
		if err != nil {
			self.errors = append(self.errors, err)
		}
		self.pass()
		return

	case 'c':
		self.read()
		var value int64
		if 'a' <= self.chr && self.chr <= 'z' {
			value = int64(self.chr) - 'a' + 1
		} else if 'A' <= self.chr && self.chr <= 'Z' {
			value = int64(self.chr) - 'A' + 1
		} else {
			err := self.goRegexp.WriteByte('c')
			if err != nil {
				self.errors = append(self.errors, err)
			}
			return
		}
		tmp := []byte{'\\', 'x', '0', 0}
		if value >= 16 {
			tmp = tmp[0:2]
		} else {
			tmp = tmp[0:3]
		}
		tmp = strconv.AppendInt(tmp, value, 16)
		_, err := self.goRegexp.Write(tmp)
		if err != nil {
			self.errors = append(self.errors, err)
		}
		self.read()
		return

	default:

		if self.chr == '$' || !isIdentifierPart(self.chr) {

			err := self.goRegexp.WriteByte('\\')
			if err != nil {
				self.errors = append(self.errors, err)
			}
		} else {

		}
		self.pass()
		return
	}

	valueOffset := self.chrOffset

	var value uint32
	{
		length := length
		for ; length > 0; length-- {
			digit := uint32(digitValue(self.chr))
			if digit >= base {

				goto skip
			}
			value = value*base + digit
			self.read()
		}
	}

	if length == 4 {
		_, err := self.goRegexp.Write([]byte{
			'\\',
			'x',
			'{',
			self.str[valueOffset+0],
			self.str[valueOffset+1],
			self.str[valueOffset+2],
			self.str[valueOffset+3],
			'}',
		})
		if err != nil {
			self.errors = append(self.errors, err)
		}
	} else if length == 2 {
		_, err := self.goRegexp.Write([]byte{
			'\\',
			'x',
			self.str[valueOffset+0],
			self.str[valueOffset+1],
		})
		if err != nil {
			self.errors = append(self.errors, err)
		}
	} else {

		self.error(-1, "re2: Illegal branch in scanEscape")
		goto skip
	}

	return

skip:
	_, err := self.goRegexp.WriteString(self.str[offset:self.chrOffset])
	if err != nil {
		self.errors = append(self.errors, err)
	}
}

func (self *_RegExp_parser) pass() {
	if self.chr != -1 {
		_, err := self.goRegexp.WriteRune(self.chr)
		if err != nil {
			self.errors = append(self.errors, err)
		}
	}
	self.read()
}

func (self *_RegExp_parser) error(offset int, msg string, msgValues ...interface{}) error {
	err := fmt.Errorf(msg, msgValues...)
	self.errors = append(self.errors, err)
	return err
}
