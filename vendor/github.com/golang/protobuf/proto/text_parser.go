
package proto

import (
	"encoding"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

const anyRepeatedlyUnpacked = "Any message unpacked multiple times, or %q already set"

type ParseError struct {
	Message string
	Line    int 
	Offset  int 
}

func (p *ParseError) Error() string {
	if p.Line == 1 {

		return fmt.Sprintf("line 1.%d: %v", p.Offset, p.Message)
	}
	return fmt.Sprintf("line %d: %v", p.Line, p.Message)
}

type token struct {
	value    string
	err      *ParseError
	line     int    
	offset   int    
	unquoted string 
}

func (t *token) String() string {
	if t.err == nil {
		return fmt.Sprintf("%q (line=%d, offset=%d)", t.value, t.line, t.offset)
	}
	return fmt.Sprintf("parse error: %v", t.err)
}

type textParser struct {
	s            string 
	done         bool   
	backed       bool   
	offset, line int
	cur          token
}

func newTextParser(s string) *textParser {
	p := new(textParser)
	p.s = s
	p.line = 1
	p.cur.line = 1
	return p
}

func (p *textParser) errorf(format string, a ...interface{}) *ParseError {
	pe := &ParseError{fmt.Sprintf(format, a...), p.cur.line, p.cur.offset}
	p.cur.err = pe
	p.done = true
	return pe
}

func isIdentOrNumberChar(c byte) bool {
	switch {
	case 'A' <= c && c <= 'Z', 'a' <= c && c <= 'z':
		return true
	case '0' <= c && c <= '9':
		return true
	}
	switch c {
	case '-', '+', '.', '_':
		return true
	}
	return false
}

func isWhitespace(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\r':
		return true
	}
	return false
}

func isQuote(c byte) bool {
	switch c {
	case '"', '\'':
		return true
	}
	return false
}

func (p *textParser) skipWhitespace() {
	i := 0
	for i < len(p.s) && (isWhitespace(p.s[i]) || p.s[i] == '#') {
		if p.s[i] == '#' {

			for i < len(p.s) && p.s[i] != '\n' {
				i++
			}
			if i == len(p.s) {
				break
			}
		}
		if p.s[i] == '\n' {
			p.line++
		}
		i++
	}
	p.offset += i
	p.s = p.s[i:len(p.s)]
	if len(p.s) == 0 {
		p.done = true
	}
}

func (p *textParser) advance() {

	p.skipWhitespace()
	if p.done {
		return
	}

	p.cur.err = nil
	p.cur.offset, p.cur.line = p.offset, p.line
	p.cur.unquoted = ""
	switch p.s[0] {
	case '<', '>', '{', '}', ':', '[', ']', ';', ',', '/':

		p.cur.value, p.s = p.s[0:1], p.s[1:len(p.s)]
	case '"', '\'':

		i := 1
		for i < len(p.s) && p.s[i] != p.s[0] && p.s[i] != '\n' {
			if p.s[i] == '\\' && i+1 < len(p.s) {

				i++
			}
			i++
		}
		if i >= len(p.s) || p.s[i] != p.s[0] {
			p.errorf("unmatched quote")
			return
		}
		unq, err := unquoteC(p.s[1:i], rune(p.s[0]))
		if err != nil {
			p.errorf("invalid quoted string %s: %v", p.s[0:i+1], err)
			return
		}
		p.cur.value, p.s = p.s[0:i+1], p.s[i+1:len(p.s)]
		p.cur.unquoted = unq
	default:
		i := 0
		for i < len(p.s) && isIdentOrNumberChar(p.s[i]) {
			i++
		}
		if i == 0 {
			p.errorf("unexpected byte %#x", p.s[0])
			return
		}
		p.cur.value, p.s = p.s[0:i], p.s[i:len(p.s)]
	}
	p.offset += len(p.cur.value)
}

var (
	errBadUTF8 = errors.New("proto: bad UTF-8")
	errBadHex  = errors.New("proto: bad hexadecimal")
)

func unquoteC(s string, quote rune) (string, error) {

	simple := true
	for _, r := range s {
		if r == '\\' || r == quote {
			simple = false
			break
		}
	}
	if simple {
		return s, nil
	}

	buf := make([]byte, 0, 3*len(s)/2)
	for len(s) > 0 {
		r, n := utf8.DecodeRuneInString(s)
		if r == utf8.RuneError && n == 1 {
			return "", errBadUTF8
		}
		s = s[n:]
		if r != '\\' {
			if r < utf8.RuneSelf {
				buf = append(buf, byte(r))
			} else {
				buf = append(buf, string(r)...)
			}
			continue
		}

		ch, tail, err := unescape(s)
		if err != nil {
			return "", err
		}
		buf = append(buf, ch...)
		s = tail
	}
	return string(buf), nil
}

func unescape(s string) (ch string, tail string, err error) {
	r, n := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && n == 1 {
		return "", "", errBadUTF8
	}
	s = s[n:]
	switch r {
	case 'a':
		return "\a", s, nil
	case 'b':
		return "\b", s, nil
	case 'f':
		return "\f", s, nil
	case 'n':
		return "\n", s, nil
	case 'r':
		return "\r", s, nil
	case 't':
		return "\t", s, nil
	case 'v':
		return "\v", s, nil
	case '?':
		return "?", s, nil 
	case '\'', '"', '\\':
		return string(r), s, nil
	case '0', '1', '2', '3', '4', '5', '6', '7', 'x', 'X':
		if len(s) < 2 {
			return "", "", fmt.Errorf(`\%c requires 2 following digits`, r)
		}
		base := 8
		ss := s[:2]
		s = s[2:]
		if r == 'x' || r == 'X' {
			base = 16
		} else {
			ss = string(r) + ss
		}
		i, err := strconv.ParseUint(ss, base, 8)
		if err != nil {
			return "", "", err
		}
		return string([]byte{byte(i)}), s, nil
	case 'u', 'U':
		n := 4
		if r == 'U' {
			n = 8
		}
		if len(s) < n {
			return "", "", fmt.Errorf(`\%c requires %d digits`, r, n)
		}

		bs := make([]byte, n/2)
		for i := 0; i < n; i += 2 {
			a, ok1 := unhex(s[i])
			b, ok2 := unhex(s[i+1])
			if !ok1 || !ok2 {
				return "", "", errBadHex
			}
			bs[i/2] = a<<4 | b
		}
		s = s[n:]
		return string(bs), s, nil
	}
	return "", "", fmt.Errorf(`unknown escape \%c`, r)
}

func unhex(b byte) (v byte, ok bool) {
	switch {
	case '0' <= b && b <= '9':
		return b - '0', true
	case 'a' <= b && b <= 'f':
		return b - 'a' + 10, true
	case 'A' <= b && b <= 'F':
		return b - 'A' + 10, true
	}
	return 0, false
}

func (p *textParser) back() { p.backed = true }

func (p *textParser) next() *token {
	if p.backed || p.done {
		p.backed = false
		return &p.cur
	}
	p.advance()
	if p.done {
		p.cur.value = ""
	} else if len(p.cur.value) > 0 && isQuote(p.cur.value[0]) {

		cat := p.cur
		for {
			p.skipWhitespace()
			if p.done || !isQuote(p.s[0]) {
				break
			}
			p.advance()
			if p.cur.err != nil {
				return &p.cur
			}
			cat.value += " " + p.cur.value
			cat.unquoted += p.cur.unquoted
		}
		p.done = false 
		p.cur = cat
	}
	return &p.cur
}

func (p *textParser) consumeToken(s string) error {
	tok := p.next()
	if tok.err != nil {
		return tok.err
	}
	if tok.value != s {
		p.back()
		return p.errorf("expected %q, found %q", s, tok.value)
	}
	return nil
}

func (p *textParser) missingRequiredFieldError(sv reflect.Value) *RequiredNotSetError {
	st := sv.Type()
	sprops := GetProperties(st)
	for i := 0; i < st.NumField(); i++ {
		if !isNil(sv.Field(i)) {
			continue
		}

		props := sprops.Prop[i]
		if props.Required {
			return &RequiredNotSetError{fmt.Sprintf("%v.%v", st, props.OrigName)}
		}
	}
	return &RequiredNotSetError{fmt.Sprintf("%v.<unknown field name>", st)} 
}

func structFieldByName(sprops *StructProperties, name string) (int, *Properties, bool) {
	i, ok := sprops.decoderOrigNames[name]
	if ok {
		return i, sprops.Prop[i], true
	}
	return -1, nil, false
}

func (p *textParser) checkForColon(props *Properties, typ reflect.Type) *ParseError {
	tok := p.next()
	if tok.err != nil {
		return tok.err
	}
	if tok.value != ":" {

		needColon := true
		switch props.Wire {
		case "group":
			needColon = false
		case "bytes":

			if typ.Kind() == reflect.Ptr {

				if typ.Elem().Kind() == reflect.String {
					break
				}
			} else if typ.Kind() == reflect.Slice {

				if typ.Elem().Kind() != reflect.Ptr {
					break
				}
			} else if typ.Kind() == reflect.String {

				break
			}
			needColon = false
		}
		if needColon {
			return p.errorf("expected ':', found %q", tok.value)
		}
		p.back()
	}
	return nil
}

func (p *textParser) readStruct(sv reflect.Value, terminator string) error {
	st := sv.Type()
	sprops := GetProperties(st)
	reqCount := sprops.reqCount
	var reqFieldErr error
	fieldSet := make(map[string]bool)

	for {
		tok := p.next()
		if tok.err != nil {
			return tok.err
		}
		if tok.value == terminator {
			break
		}
		if tok.value == "[" {

			extName, err := p.consumeExtName()
			if err != nil {
				return err
			}

			if s := strings.LastIndex(extName, "/"); s >= 0 {

				messageName := extName[s+1:]
				mt := MessageType(messageName)
				if mt == nil {
					return p.errorf("unrecognized message %q in google.protobuf.Any", messageName)
				}
				tok = p.next()
				if tok.err != nil {
					return tok.err
				}

				if tok.value == ":" {
					tok = p.next()
					if tok.err != nil {
						return tok.err
					}
				}
				var terminator string
				switch tok.value {
				case "<":
					terminator = ">"
				case "{":
					terminator = "}"
				default:
					return p.errorf("expected '{' or '<', found %q", tok.value)
				}
				v := reflect.New(mt.Elem())
				if pe := p.readStruct(v.Elem(), terminator); pe != nil {
					return pe
				}
				b, err := Marshal(v.Interface().(Message))
				if err != nil {
					return p.errorf("failed to marshal message of type %q: %v", messageName, err)
				}
				if fieldSet["type_url"] {
					return p.errorf(anyRepeatedlyUnpacked, "type_url")
				}
				if fieldSet["value"] {
					return p.errorf(anyRepeatedlyUnpacked, "value")
				}
				sv.FieldByName("TypeUrl").SetString(extName)
				sv.FieldByName("Value").SetBytes(b)
				fieldSet["type_url"] = true
				fieldSet["value"] = true
				continue
			}

			var desc *ExtensionDesc

			for _, d := range RegisteredExtensions(reflect.New(st).Interface().(Message)) {
				if d.Name == extName {
					desc = d
					break
				}
			}
			if desc == nil {
				return p.errorf("unrecognized extension %q", extName)
			}

			props := &Properties{}
			props.Parse(desc.Tag)

			typ := reflect.TypeOf(desc.ExtensionType)
			if err := p.checkForColon(props, typ); err != nil {
				return err
			}

			rep := desc.repeated()

			var ext reflect.Value
			if !rep {
				ext = reflect.New(typ).Elem()
			} else {
				ext = reflect.New(typ.Elem()).Elem()
			}
			if err := p.readAny(ext, props); err != nil {
				if _, ok := err.(*RequiredNotSetError); !ok {
					return err
				}
				reqFieldErr = err
			}
			ep := sv.Addr().Interface().(Message)
			if !rep {
				SetExtension(ep, desc, ext.Interface())
			} else {
				old, err := GetExtension(ep, desc)
				var sl reflect.Value
				if err == nil {
					sl = reflect.ValueOf(old) 
				} else {
					sl = reflect.MakeSlice(typ, 0, 1)
				}
				sl = reflect.Append(sl, ext)
				SetExtension(ep, desc, sl.Interface())
			}
			if err := p.consumeOptionalSeparator(); err != nil {
				return err
			}
			continue
		}

		name := tok.value
		var dst reflect.Value
		fi, props, ok := structFieldByName(sprops, name)
		if ok {
			dst = sv.Field(fi)
		} else if oop, ok := sprops.OneofTypes[name]; ok {

			props = oop.Prop
			nv := reflect.New(oop.Type.Elem())
			dst = nv.Elem().Field(0)
			field := sv.Field(oop.Field)
			if !field.IsNil() {
				return p.errorf("field '%s' would overwrite already parsed oneof '%s'", name, sv.Type().Field(oop.Field).Name)
			}
			field.Set(nv)
		}
		if !dst.IsValid() {
			return p.errorf("unknown field name %q in %v", name, st)
		}

		if dst.Kind() == reflect.Map {

			if err := p.checkForColon(props, dst.Type()); err != nil {
				return err
			}

			if dst.IsNil() {
				dst.Set(reflect.MakeMap(dst.Type()))
			}
			key := reflect.New(dst.Type().Key()).Elem()
			val := reflect.New(dst.Type().Elem()).Elem()

			tok := p.next()
			var terminator string
			switch tok.value {
			case "<":
				terminator = ">"
			case "{":
				terminator = "}"
			default:
				return p.errorf("expected '{' or '<', found %q", tok.value)
			}
			for {
				tok := p.next()
				if tok.err != nil {
					return tok.err
				}
				if tok.value == terminator {
					break
				}
				switch tok.value {
				case "key":
					if err := p.consumeToken(":"); err != nil {
						return err
					}
					if err := p.readAny(key, props.mkeyprop); err != nil {
						return err
					}
					if err := p.consumeOptionalSeparator(); err != nil {
						return err
					}
				case "value":
					if err := p.checkForColon(props.mvalprop, dst.Type().Elem()); err != nil {
						return err
					}
					if err := p.readAny(val, props.mvalprop); err != nil {
						return err
					}
					if err := p.consumeOptionalSeparator(); err != nil {
						return err
					}
				default:
					p.back()
					return p.errorf(`expected "key", "value", or %q, found %q`, terminator, tok.value)
				}
			}

			dst.SetMapIndex(key, val)
			continue
		}

		if !props.Repeated && fieldSet[name] {
			return p.errorf("non-repeated field %q was repeated", name)
		}

		if err := p.checkForColon(props, dst.Type()); err != nil {
			return err
		}

		fieldSet[name] = true
		if err := p.readAny(dst, props); err != nil {
			if _, ok := err.(*RequiredNotSetError); !ok {
				return err
			}
			reqFieldErr = err
		}
		if props.Required {
			reqCount--
		}

		if err := p.consumeOptionalSeparator(); err != nil {
			return err
		}

	}

	if reqCount > 0 {
		return p.missingRequiredFieldError(sv)
	}
	return reqFieldErr
}

func (p *textParser) consumeExtName() (string, error) {
	tok := p.next()
	if tok.err != nil {
		return "", tok.err
	}

	if len(tok.value) > 2 && isQuote(tok.value[0]) && tok.value[len(tok.value)-1] == tok.value[0] {
		name, err := unquoteC(tok.value[1:len(tok.value)-1], rune(tok.value[0]))
		if err != nil {
			return "", err
		}
		return name, p.consumeToken("]")
	}

	var parts []string
	for tok.value != "]" {
		parts = append(parts, tok.value)
		tok = p.next()
		if tok.err != nil {
			return "", p.errorf("unrecognized type_url or extension name: %s", tok.err)
		}
	}
	return strings.Join(parts, ""), nil
}

func (p *textParser) consumeOptionalSeparator() error {
	tok := p.next()
	if tok.err != nil {
		return tok.err
	}
	if tok.value != ";" && tok.value != "," {
		p.back()
	}
	return nil
}

func (p *textParser) readAny(v reflect.Value, props *Properties) error {
	tok := p.next()
	if tok.err != nil {
		return tok.err
	}
	if tok.value == "" {
		return p.errorf("unexpected EOF")
	}

	switch fv := v; fv.Kind() {
	case reflect.Slice:
		at := v.Type()
		if at.Elem().Kind() == reflect.Uint8 {

			if tok.value[0] != '"' && tok.value[0] != '\'' {

				return p.errorf("invalid string: %v", tok.value)
			}
			bytes := []byte(tok.unquoted)
			fv.Set(reflect.ValueOf(bytes))
			return nil
		}

		if tok.value == "[" {

			for {
				fv.Set(reflect.Append(fv, reflect.New(at.Elem()).Elem()))
				err := p.readAny(fv.Index(fv.Len()-1), props)
				if err != nil {
					return err
				}
				tok := p.next()
				if tok.err != nil {
					return tok.err
				}
				if tok.value == "]" {
					break
				}
				if tok.value != "," {
					return p.errorf("Expected ']' or ',' found %q", tok.value)
				}
			}
			return nil
		}

		p.back()
		fv.Set(reflect.Append(fv, reflect.New(at.Elem()).Elem()))
		return p.readAny(fv.Index(fv.Len()-1), props)
	case reflect.Bool:

		switch tok.value {
		case "true", "1", "t", "True":
			fv.SetBool(true)
			return nil
		case "false", "0", "f", "False":
			fv.SetBool(false)
			return nil
		}
	case reflect.Float32, reflect.Float64:
		v := tok.value

		if strings.HasSuffix(v, "f") && tok.value != "-inf" && tok.value != "inf" {
			v = v[:len(v)-1]
		}
		if f, err := strconv.ParseFloat(v, fv.Type().Bits()); err == nil {
			fv.SetFloat(f)
			return nil
		}
	case reflect.Int32:
		if x, err := strconv.ParseInt(tok.value, 0, 32); err == nil {
			fv.SetInt(x)
			return nil
		}

		if len(props.Enum) == 0 {
			break
		}
		m, ok := enumValueMaps[props.Enum]
		if !ok {
			break
		}
		x, ok := m[tok.value]
		if !ok {
			break
		}
		fv.SetInt(int64(x))
		return nil
	case reflect.Int64:
		if x, err := strconv.ParseInt(tok.value, 0, 64); err == nil {
			fv.SetInt(x)
			return nil
		}

	case reflect.Ptr:

		p.back()
		fv.Set(reflect.New(fv.Type().Elem()))
		return p.readAny(fv.Elem(), props)
	case reflect.String:
		if tok.value[0] == '"' || tok.value[0] == '\'' {
			fv.SetString(tok.unquoted)
			return nil
		}
	case reflect.Struct:
		var terminator string
		switch tok.value {
		case "{":
			terminator = "}"
		case "<":
			terminator = ">"
		default:
			return p.errorf("expected '{' or '<', found %q", tok.value)
		}

		return p.readStruct(fv, terminator)
	case reflect.Uint32:
		if x, err := strconv.ParseUint(tok.value, 0, 32); err == nil {
			fv.SetUint(x)
			return nil
		}
	case reflect.Uint64:
		if x, err := strconv.ParseUint(tok.value, 0, 64); err == nil {
			fv.SetUint(x)
			return nil
		}
	}
	return p.errorf("invalid %v: %v", v.Type(), tok.value)
}

func UnmarshalText(s string, pb Message) error {
	if um, ok := pb.(encoding.TextUnmarshaler); ok {
		err := um.UnmarshalText([]byte(s))
		return err
	}
	pb.Reset()
	v := reflect.ValueOf(pb)
	if pe := newTextParser(s).readStruct(v.Elem(), ""); pe != nil {
		return pe
	}
	return nil
}
