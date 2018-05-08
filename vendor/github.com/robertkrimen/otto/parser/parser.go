
package parser

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"io/ioutil"

	"github.com/robertkrimen/otto/ast"
	"github.com/robertkrimen/otto/file"
	"github.com/robertkrimen/otto/token"
	"gopkg.in/sourcemap.v1"
)

type Mode uint

const (
	IgnoreRegExpErrors Mode = 1 << iota 
	StoreComments                       
)

type _parser struct {
	str    string
	length int
	base   int

	chr       rune 
	chrOffset int  
	offset    int  

	idx     file.Idx    
	token   token.Token 
	literal string      

	scope             *_scope
	insertSemicolon   bool 
	implicitSemicolon bool 

	errors ErrorList

	recover struct {

		idx   file.Idx
		count int
	}

	mode Mode

	file *file.File

	comments *ast.Comments
}

type Parser interface {
	Scan() (tkn token.Token, literal string, idx file.Idx)
}

func _newParser(filename, src string, base int, sm *sourcemap.Consumer) *_parser {
	return &_parser{
		chr:      ' ', 
		str:      src,
		length:   len(src),
		base:     base,
		file:     file.NewFile(filename, src, base).WithSourceMap(sm),
		comments: ast.NewComments(),
	}
}

func NewParser(filename, src string) Parser {
	return _newParser(filename, src, 1, nil)
}

func ReadSource(filename string, src interface{}) ([]byte, error) {
	if src != nil {
		switch src := src.(type) {
		case string:
			return []byte(src), nil
		case []byte:
			return src, nil
		case *bytes.Buffer:
			if src != nil {
				return src.Bytes(), nil
			}
		case io.Reader:
			var bfr bytes.Buffer
			if _, err := io.Copy(&bfr, src); err != nil {
				return nil, err
			}
			return bfr.Bytes(), nil
		}
		return nil, errors.New("invalid source")
	}
	return ioutil.ReadFile(filename)
}

func ReadSourceMap(filename string, src interface{}) (*sourcemap.Consumer, error) {
	if src == nil {
		return nil, nil
	}

	switch src := src.(type) {
	case string:
		return sourcemap.Parse(filename, []byte(src))
	case []byte:
		return sourcemap.Parse(filename, src)
	case *bytes.Buffer:
		if src != nil {
			return sourcemap.Parse(filename, src.Bytes())
		}
	case io.Reader:
		var bfr bytes.Buffer
		if _, err := io.Copy(&bfr, src); err != nil {
			return nil, err
		}
		return sourcemap.Parse(filename, bfr.Bytes())
	case *sourcemap.Consumer:
		return src, nil
	}

	return nil, errors.New("invalid sourcemap type")
}

func ParseFileWithSourceMap(fileSet *file.FileSet, filename string, javascriptSource, sourcemapSource interface{}, mode Mode) (*ast.Program, error) {
	src, err := ReadSource(filename, javascriptSource)
	if err != nil {
		return nil, err
	}

	if sourcemapSource == nil {
		lines := bytes.Split(src, []byte("\n"))
		lastLine := lines[len(lines)-1]
		if bytes.HasPrefix(lastLine, []byte("//# sourceMappingURL=data:application/json")) {
			bits := bytes.SplitN(lastLine, []byte(","), 2)
			if len(bits) == 2 {
				if d, err := base64.StdEncoding.DecodeString(string(bits[1])); err == nil {
					sourcemapSource = d
				}
			}
		}
	}

	sm, err := ReadSourceMap(filename, sourcemapSource)
	if err != nil {
		return nil, err
	}

	base := 1
	if fileSet != nil {
		base = fileSet.AddFile(filename, string(src))
	}

	parser := _newParser(filename, string(src), base, sm)
	parser.mode = mode
	program, err := parser.parse()
	program.Comments = parser.comments.CommentMap

	return program, err
}

func ParseFile(fileSet *file.FileSet, filename string, src interface{}, mode Mode) (*ast.Program, error) {
	return ParseFileWithSourceMap(fileSet, filename, src, nil, mode)
}

func ParseFunction(parameterList, body string) (*ast.FunctionLiteral, error) {

	src := "(function(" + parameterList + ") {\n" + body + "\n})"

	parser := _newParser("", src, 1, nil)
	program, err := parser.parse()
	if err != nil {
		return nil, err
	}

	return program.Body[0].(*ast.ExpressionStatement).Expression.(*ast.FunctionLiteral), nil
}

func (self *_parser) Scan() (tkn token.Token, literal string, idx file.Idx) {
	return self.scan()
}

func (self *_parser) slice(idx0, idx1 file.Idx) string {
	from := int(idx0) - self.base
	to := int(idx1) - self.base
	if from >= 0 && to <= len(self.str) {
		return self.str[from:to]
	}

	return ""
}

func (self *_parser) parse() (*ast.Program, error) {
	self.next()
	program := self.parseProgram()
	if false {
		self.errors.Sort()
	}

	if self.mode&StoreComments != 0 {
		self.comments.CommentMap.AddComments(program, self.comments.FetchAll(), ast.TRAILING)
	}

	return program, self.errors.Err()
}

func (self *_parser) next() {
	self.token, self.literal, self.idx = self.scan()
}

func (self *_parser) optionalSemicolon() {
	if self.token == token.SEMICOLON {
		self.next()
		return
	}

	if self.implicitSemicolon {
		self.implicitSemicolon = false
		return
	}

	if self.token != token.EOF && self.token != token.RIGHT_BRACE {
		self.expect(token.SEMICOLON)
	}
}

func (self *_parser) semicolon() {
	if self.token != token.RIGHT_PARENTHESIS && self.token != token.RIGHT_BRACE {
		if self.implicitSemicolon {
			self.implicitSemicolon = false
			return
		}

		self.expect(token.SEMICOLON)
	}
}

func (self *_parser) idxOf(offset int) file.Idx {
	return file.Idx(self.base + offset)
}

func (self *_parser) expect(value token.Token) file.Idx {
	idx := self.idx
	if self.token != value {
		self.errorUnexpectedToken(self.token)
	}
	self.next()
	return idx
}

func lineCount(str string) (int, int) {
	line, last := 0, -1
	pair := false
	for index, chr := range str {
		switch chr {
		case '\r':
			line += 1
			last = index
			pair = true
			continue
		case '\n':
			if !pair {
				line += 1
			}
			last = index
		case '\u2028', '\u2029':
			line += 1
			last = index + 2
		}
		pair = false
	}
	return line, last
}

func (self *_parser) position(idx file.Idx) file.Position {
	position := file.Position{}
	offset := int(idx) - self.base
	str := self.str[:offset]
	position.Filename = self.file.Name()
	line, last := lineCount(str)
	position.Line = 1 + line
	if last >= 0 {
		position.Column = offset - last
	} else {
		position.Column = 1 + len(str)
	}

	return position
}
