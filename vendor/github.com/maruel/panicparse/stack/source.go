
package stack

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"math"
	"strings"
)

type cache struct {
	files  map[string][]byte
	parsed map[string]*parsedFile
}

func Augment(goroutines []Goroutine) {
	c := &cache{}
	for i := range goroutines {
		c.augmentGoroutine(&goroutines[i])
	}
}

func (c *cache) augmentGoroutine(goroutine *Goroutine) {
	if c.files == nil {
		c.files = map[string][]byte{}
	}
	if c.parsed == nil {
		c.parsed = map[string]*parsedFile{}
	}

	for i := range goroutine.Stack.Calls {
		c.load(goroutine.Stack.Calls[i].SourcePath)
	}

	for i := 1; i < len(goroutine.Stack.Calls); i++ {

		if f := c.getFuncAST(&goroutine.Stack.Calls[i]); f != nil {
			processCall(&goroutine.Stack.Calls[i], f)
		}
	}
}

func (c *cache) load(fileName string) {
	if _, ok := c.parsed[fileName]; ok {
		return
	}
	c.parsed[fileName] = nil
	if !strings.HasSuffix(fileName, ".go") {

		c.files[fileName] = nil
		return
	}
	log.Printf("load(%s)", fileName)
	if _, ok := c.files[fileName]; !ok {
		var err error
		if c.files[fileName], err = ioutil.ReadFile(fileName); err != nil {
			log.Printf("Failed to read %s: %s", fileName, err)
			c.files[fileName] = nil
			return
		}
	}
	fset := token.NewFileSet()
	src := c.files[fileName]
	parsed, err := parser.ParseFile(fset, fileName, src, 0)
	if err != nil {
		log.Printf("Failed to parse %s: %s", fileName, err)
		return
	}

	offsets := []int{0, 0}
	start := 0
	for l := 1; start < len(src); l++ {
		start += bytes.IndexByte(src[start:], '\n') + 1
		offsets = append(offsets, start)
	}
	c.parsed[fileName] = &parsedFile{offsets, parsed}
}

func (c *cache) getFuncAST(call *Call) *ast.FuncDecl {
	if p := c.parsed[call.SourcePath]; p != nil {
		return p.getFuncAST(call.Func.Name(), call.Line)
	}
	return nil
}

type parsedFile struct {
	lineToByteOffset []int
	parsed           *ast.File
}

func (p *parsedFile) getFuncAST(f string, l int) (d *ast.FuncDecl) {

	var lastFunc *ast.FuncDecl
	var found ast.Node

	ast.Inspect(p.parsed, func(n ast.Node) bool {
		if d != nil {
			return false
		}
		if n == nil {
			return true
		}
		if found != nil {

		}
		if int(n.Pos()) >= p.lineToByteOffset[l] {

			d = lastFunc

			return false
		} else if f, ok := n.(*ast.FuncDecl); ok {
			lastFunc = f
		}
		return true
	})
	return
}

func name(n ast.Node) string {
	if _, ok := n.(*ast.InterfaceType); ok {
		return "interface{}"
	}
	if i, ok := n.(*ast.Ident); ok {
		return i.Name
	}
	if _, ok := n.(*ast.FuncType); ok {
		return "func"
	}
	if s, ok := n.(*ast.SelectorExpr); ok {
		return s.Sel.Name
	}

	return "<unknown>"
}

func fieldToType(f *ast.Field) (string, bool) {
	switch arg := f.Type.(type) {
	case *ast.ArrayType:
		return "[]" + name(arg.Elt), false
	case *ast.Ellipsis:
		return name(arg.Elt), true
	case *ast.FuncType:

		return "func", false
	case *ast.Ident:
		return arg.Name, false
	case *ast.InterfaceType:
		return "interface{}", false
	case *ast.SelectorExpr:
		return arg.Sel.Name, false
	case *ast.StarExpr:
		return "*" + name(arg.X), false
	default:

		return "<unknown>", false
	}
}

func extractArgumentsType(f *ast.FuncDecl) ([]string, bool) {
	var fields []*ast.Field
	if f.Recv != nil {
		if len(f.Recv.List) != 1 {
			panic("Expect only one receiver; please fix panicparse's code")
		}

		if _, ok := f.Recv.List[0].Type.(*ast.StarExpr); ok {
			fields = append(fields, f.Recv.List[0])
		}
	}
	var types []string
	extra := false
	for _, arg := range append(fields, f.Type.Params.List...) {

		var t string
		t, extra = fieldToType(arg)
		mult := len(arg.Names)
		if mult == 0 {
			mult = 1
		}
		for i := 0; i < mult; i++ {
			types = append(types, t)
		}
	}
	return types, extra
}

func processCall(call *Call, f *ast.FuncDecl) {
	values := make([]uint64, len(call.Args.Values))
	for i := range call.Args.Values {
		values[i] = call.Args.Values[i].Value
	}
	index := 0
	pop := func() uint64 {
		if len(values) != 0 {
			x := values[0]
			values = values[1:]
			index++
			return x
		}
		return 0
	}
	popName := func() string {
		n := call.Args.Values[index].Name
		v := pop()
		if len(n) == 0 {
			return fmt.Sprintf("0x%x", v)
		}
		return n
	}

	types, extra := extractArgumentsType(f)
	for i := 0; len(values) != 0; i++ {
		var t string
		if i >= len(types) {
			if !extra {

				call.Args.Processed = append(call.Args.Processed, popName())
				continue
			}
			t = types[len(types)-1]
		} else {
			t = types[i]
		}
		switch t {
		case "float32":
			call.Args.Processed = append(call.Args.Processed, fmt.Sprintf("%g", math.Float32frombits(uint32(pop()))))
		case "float64":
			call.Args.Processed = append(call.Args.Processed, fmt.Sprintf("%g", math.Float64frombits(pop())))
		case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
			call.Args.Processed = append(call.Args.Processed, fmt.Sprintf("%d", pop()))
		case "string":
			call.Args.Processed = append(call.Args.Processed, fmt.Sprintf("%s(%s, len=%d)", t, popName(), pop()))
		default:
			if strings.HasPrefix(t, "*") {
				call.Args.Processed = append(call.Args.Processed, fmt.Sprintf("%s(%s)", t, popName()))
			} else if strings.HasPrefix(t, "[]") {
				call.Args.Processed = append(call.Args.Processed, fmt.Sprintf("%s(%s len=%d cap=%d)", t, popName(), pop(), pop()))
			} else {

				call.Args.Processed = append(call.Args.Processed, fmt.Sprintf("%s(%s)", t, popName()))
				pop()
			}
		}
		if len(values) == 0 && call.Args.Elided {
			return
		}
	}
}
