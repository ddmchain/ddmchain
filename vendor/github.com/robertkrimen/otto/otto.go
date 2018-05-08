
package otto

import (
	"fmt"
	"strings"

	"github.com/robertkrimen/otto/file"
	"github.com/robertkrimen/otto/registry"
)

type Otto struct {

	Interrupt chan func()
	runtime   *_runtime
}

func New() *Otto {
	self := &Otto{
		runtime: newContext(),
	}
	self.runtime.otto = self
	self.runtime.traceLimit = 10
	self.Set("console", self.runtime.newConsole())

	registry.Apply(func(entry registry.Entry) {
		self.Run(entry.Source())
	})

	return self
}

func (otto *Otto) clone() *Otto {
	self := &Otto{
		runtime: otto.runtime.clone(),
	}
	self.runtime.otto = self
	return self
}

func Run(src interface{}) (*Otto, Value, error) {
	otto := New()
	value, err := otto.Run(src) 
	return otto, value, err
}

func (self Otto) Run(src interface{}) (Value, error) {
	value, err := self.runtime.cmpl_run(src, nil)
	if !value.safe() {
		value = Value{}
	}
	return value, err
}

func (self Otto) Eval(src interface{}) (Value, error) {
	if self.runtime.scope == nil {
		self.runtime.enterGlobalScope()
		defer self.runtime.leaveScope()
	}

	value, err := self.runtime.cmpl_eval(src, nil)
	if !value.safe() {
		value = Value{}
	}
	return value, err
}

func (self Otto) Get(name string) (Value, error) {
	value := Value{}
	err := catchPanic(func() {
		value = self.getValue(name)
	})
	if !value.safe() {
		value = Value{}
	}
	return value, err
}

func (self Otto) getValue(name string) Value {
	return self.runtime.globalStash.getBinding(name, false)
}

func (self Otto) Set(name string, value interface{}) error {
	{
		value, err := self.ToValue(value)
		if err != nil {
			return err
		}
		err = catchPanic(func() {
			self.setValue(name, value)
		})
		return err
	}
}

func (self Otto) setValue(name string, value Value) {
	self.runtime.globalStash.setValue(name, value, false)
}

func (self Otto) SetDebuggerHandler(fn func(vm *Otto)) {
	self.runtime.debugger = fn
}

func (self Otto) SetRandomSource(fn func() float64) {
	self.runtime.random = fn
}

func (self Otto) SetStackDepthLimit(limit int) {
	self.runtime.stackLimit = limit
}

func (self Otto) SetStackTraceLimit(limit int) {
	self.runtime.traceLimit = limit
}

func (self Otto) MakeCustomError(name, message string) Value {
	return self.runtime.toValue(self.runtime.newError(name, self.runtime.toValue(message), 0))
}

func (self Otto) MakeRangeError(message string) Value {
	return self.runtime.toValue(self.runtime.newRangeError(self.runtime.toValue(message)))
}

func (self Otto) MakeSyntaxError(message string) Value {
	return self.runtime.toValue(self.runtime.newSyntaxError(self.runtime.toValue(message)))
}

func (self Otto) MakeTypeError(message string) Value {
	return self.runtime.toValue(self.runtime.newTypeError(self.runtime.toValue(message)))
}

type Context struct {
	Filename   string
	Line       int
	Column     int
	Callee     string
	Symbols    map[string]Value
	This       Value
	Stacktrace []string
}

func (self Otto) Context() Context {
	return self.ContextSkip(10, true)
}

func (self Otto) ContextLimit(limit int) Context {
	return self.ContextSkip(limit, true)
}

func (self Otto) ContextSkip(limit int, skipNative bool) (ctx Context) {

	if self.runtime.scope == nil {
		self.runtime.enterGlobalScope()
		defer self.runtime.leaveScope()
	}

	scope := self.runtime.scope
	frame := scope.frame

	for skipNative && frame.native && scope.outer != nil {
		scope = scope.outer
		frame = scope.frame
	}

	ctx.Filename = "<unknown>"
	ctx.Callee = frame.callee

	switch {
	case frame.native:
		ctx.Filename = frame.nativeFile
		ctx.Line = frame.nativeLine
		ctx.Column = 0
	case frame.file != nil:
		ctx.Filename = "<anonymous>"

		if p := frame.file.Position(file.Idx(frame.offset)); p != nil {
			ctx.Line = p.Line
			ctx.Column = p.Column

			if p.Filename != "" {
				ctx.Filename = p.Filename
			}
		}
	}

	ctx.This = toValue_object(scope.this)

	ctx.Symbols = make(map[string]Value)
	ctx.Stacktrace = append(ctx.Stacktrace, frame.location())
	for limit != 0 {

		stash := scope.lexical
		for {
			for _, name := range getStashProperties(stash) {
				if _, ok := ctx.Symbols[name]; !ok {
					ctx.Symbols[name] = stash.getBinding(name, true)
				}
			}
			stash = stash.outer()
			if stash == nil || stash.outer() == nil {
				break
			}
		}

		scope = scope.outer
		if scope == nil {
			break
		}
		if scope.frame.offset >= 0 {
			ctx.Stacktrace = append(ctx.Stacktrace, scope.frame.location())
		}
		limit--
	}

	return
}

func (self Otto) Call(source string, this interface{}, argumentList ...interface{}) (Value, error) {

	thisValue := Value{}

	construct := false
	if strings.HasPrefix(source, "new ") {
		source = source[4:]
		construct = true
	}

	self.runtime.enterGlobalScope()
	defer func() {
		self.runtime.leaveScope()
	}()

	if !construct && this == nil {
		program, err := self.runtime.cmpl_parse("", source+"()", nil)
		if err == nil {
			if node, ok := program.body[0].(*_nodeExpressionStatement); ok {
				if node, ok := node.expression.(*_nodeCallExpression); ok {
					var value Value
					err := catchPanic(func() {
						value = self.runtime.cmpl_evaluate_nodeCallExpression(node, argumentList)
					})
					if err != nil {
						return Value{}, err
					}
					return value, nil
				}
			}
		}
	} else {
		value, err := self.ToValue(this)
		if err != nil {
			return Value{}, err
		}
		thisValue = value
	}

	{
		this := thisValue

		fn, err := self.Run(source)
		if err != nil {
			return Value{}, err
		}

		if construct {
			result, err := fn.constructSafe(self.runtime, this, argumentList...)
			if err != nil {
				return Value{}, err
			}
			return result, nil
		}

		result, err := fn.Call(this, argumentList...)
		if err != nil {
			return Value{}, err
		}
		return result, nil
	}
}

func (self Otto) Object(source string) (*Object, error) {
	value, err := self.runtime.cmpl_run(source, nil)
	if err != nil {
		return nil, err
	}
	if value.IsObject() {
		return value.Object(), nil
	}
	return nil, fmt.Errorf("value is not an object")
}

func (self Otto) ToValue(value interface{}) (Value, error) {
	return self.runtime.safeToValue(value)
}

func (in *Otto) Copy() *Otto {
	out := &Otto{
		runtime: in.runtime.clone(),
	}
	out.runtime.otto = out
	return out
}

type Object struct {
	object *_object
	value  Value
}

func _newObject(object *_object, value Value) *Object {

	return &Object{
		object: object,
		value:  value,
	}
}

func (self Object) Call(name string, argumentList ...interface{}) (Value, error) {

	function, err := self.Get(name)
	if err != nil {
		return Value{}, err
	}
	return function.Call(self.Value(), argumentList...)
}

func (self Object) Value() Value {
	return self.value
}

func (self Object) Get(name string) (Value, error) {
	value := Value{}
	err := catchPanic(func() {
		value = self.object.get(name)
	})
	if !value.safe() {
		value = Value{}
	}
	return value, err
}

func (self Object) Set(name string, value interface{}) error {
	{
		value, err := self.object.runtime.safeToValue(value)
		if err != nil {
			return err
		}
		err = catchPanic(func() {
			self.object.put(name, value, true)
		})
		return err
	}
}

func (self Object) Keys() []string {
	var keys []string
	self.object.enumerate(false, func(name string) bool {
		keys = append(keys, name)
		return true
	})
	return keys
}

func (self Object) KeysByParent() [][]string {
	var a [][]string

	for o := self.object; o != nil; o = o.prototype {
		var l []string

		o.enumerate(false, func(name string) bool {
			l = append(l, name)
			return true
		})

		a = append(a, l)
	}

	return a
}

func (self Object) Class() string {
	return self.object.class
}
