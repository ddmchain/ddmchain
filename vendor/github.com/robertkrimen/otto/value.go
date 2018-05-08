package otto

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"unicode/utf16"
)

type _valueKind int

const (
	valueUndefined _valueKind = iota
	valueNull
	valueNumber
	valueString
	valueBoolean
	valueObject

	valueEmpty
	valueResult
	valueReference
)

type Value struct {
	kind  _valueKind
	value interface{}
}

func (value Value) safe() bool {
	return value.kind < valueEmpty
}

var (
	emptyValue = Value{kind: valueEmpty}
	nullValue  = Value{kind: valueNull}
	falseValue = Value{kind: valueBoolean, value: false}
	trueValue  = Value{kind: valueBoolean, value: true}
)

func ToValue(value interface{}) (Value, error) {
	result := Value{}
	err := catchPanic(func() {
		result = toValue(value)
	})
	return result, err
}

func (value Value) isEmpty() bool {
	return value.kind == valueEmpty
}

func UndefinedValue() Value {
	return Value{}
}

func (value Value) IsDefined() bool {
	return value.kind != valueUndefined
}

func (value Value) IsUndefined() bool {
	return value.kind == valueUndefined
}

func NullValue() Value {
	return Value{kind: valueNull}
}

func (value Value) IsNull() bool {
	return value.kind == valueNull
}

func (value Value) isCallable() bool {
	switch value := value.value.(type) {
	case *_object:
		return value.isCall()
	}
	return false
}

func (value Value) Call(this Value, argumentList ...interface{}) (Value, error) {
	result := Value{}
	err := catchPanic(func() {

		result = value.call(nil, this, argumentList...)
	})
	if !value.safe() {
		value = Value{}
	}
	return result, err
}

func (value Value) call(rt *_runtime, this Value, argumentList ...interface{}) Value {
	switch function := value.value.(type) {
	case *_object:
		return function.call(this, function.runtime.toValueArray(argumentList...), false, nativeFrame)
	}
	if rt == nil {
		panic("FIXME TypeError")
	}
	panic(rt.panicTypeError())
}

func (value Value) constructSafe(rt *_runtime, this Value, argumentList ...interface{}) (Value, error) {
	result := Value{}
	err := catchPanic(func() {
		result = value.construct(rt, this, argumentList...)
	})
	return result, err
}

func (value Value) construct(rt *_runtime, this Value, argumentList ...interface{}) Value {
	switch fn := value.value.(type) {
	case *_object:
		return fn.construct(fn.runtime.toValueArray(argumentList...))
	}
	if rt == nil {
		panic("FIXME TypeError")
	}
	panic(rt.panicTypeError())
}

func (value Value) IsPrimitive() bool {
	return !value.IsObject()
}

func (value Value) IsBoolean() bool {
	return value.kind == valueBoolean
}

func (value Value) IsNumber() bool {
	return value.kind == valueNumber
}

func (value Value) IsNaN() bool {
	switch value := value.value.(type) {
	case float64:
		return math.IsNaN(value)
	case float32:
		return math.IsNaN(float64(value))
	case int, int8, int32, int64:
		return false
	case uint, uint8, uint32, uint64:
		return false
	}

	return math.IsNaN(value.float64())
}

func (value Value) IsString() bool {
	return value.kind == valueString
}

func (value Value) IsObject() bool {
	return value.kind == valueObject
}

func (value Value) IsFunction() bool {
	if value.kind != valueObject {
		return false
	}
	return value.value.(*_object).class == "Function"
}

func (value Value) Class() string {
	if value.kind != valueObject {
		return ""
	}
	return value.value.(*_object).class
}

func (value Value) isArray() bool {
	if value.kind != valueObject {
		return false
	}
	return isArray(value.value.(*_object))
}

func (value Value) isStringObject() bool {
	if value.kind != valueObject {
		return false
	}
	return value.value.(*_object).class == "String"
}

func (value Value) isBooleanObject() bool {
	if value.kind != valueObject {
		return false
	}
	return value.value.(*_object).class == "Boolean"
}

func (value Value) isNumberObject() bool {
	if value.kind != valueObject {
		return false
	}
	return value.value.(*_object).class == "Number"
}

func (value Value) isDate() bool {
	if value.kind != valueObject {
		return false
	}
	return value.value.(*_object).class == "Date"
}

func (value Value) isRegExp() bool {
	if value.kind != valueObject {
		return false
	}
	return value.value.(*_object).class == "RegExp"
}

func (value Value) isError() bool {
	if value.kind != valueObject {
		return false
	}
	return value.value.(*_object).class == "Error"
}

func toValue_reflectValuePanic(value interface{}, kind reflect.Kind) {

	switch kind {
	case reflect.Struct:
		panic(newError(nil, "TypeError", 0, "invalid value (struct): missing runtime: %v (%T)", value, value))
	case reflect.Map:
		panic(newError(nil, "TypeError", 0, "invalid value (map): missing runtime: %v (%T)", value, value))
	case reflect.Slice:
		panic(newError(nil, "TypeError", 0, "invalid value (slice): missing runtime: %v (%T)", value, value))
	}
}

func toValue(value interface{}) Value {
	switch value := value.(type) {
	case Value:
		return value
	case bool:
		return Value{valueBoolean, value}
	case int:
		return Value{valueNumber, value}
	case int8:
		return Value{valueNumber, value}
	case int16:
		return Value{valueNumber, value}
	case int32:
		return Value{valueNumber, value}
	case int64:
		return Value{valueNumber, value}
	case uint:
		return Value{valueNumber, value}
	case uint8:
		return Value{valueNumber, value}
	case uint16:
		return Value{valueNumber, value}
	case uint32:
		return Value{valueNumber, value}
	case uint64:
		return Value{valueNumber, value}
	case float32:
		return Value{valueNumber, float64(value)}
	case float64:
		return Value{valueNumber, value}
	case []uint16:
		return Value{valueString, value}
	case string:
		return Value{valueString, value}

	case *_object:
		return Value{valueObject, value}
	case *Object:
		return Value{valueObject, value.object}
	case Object:
		return Value{valueObject, value.object}
	case _reference: 
		return Value{valueReference, value}
	case _result:
		return Value{valueResult, value}
	case nil:

		return Value{}
	case reflect.Value:
		for value.Kind() == reflect.Ptr {

			if value.IsNil() {
				return Value{}
			}
			value = value.Elem()
		}
		switch value.Kind() {
		case reflect.Bool:
			return Value{valueBoolean, bool(value.Bool())}
		case reflect.Int:
			return Value{valueNumber, int(value.Int())}
		case reflect.Int8:
			return Value{valueNumber, int8(value.Int())}
		case reflect.Int16:
			return Value{valueNumber, int16(value.Int())}
		case reflect.Int32:
			return Value{valueNumber, int32(value.Int())}
		case reflect.Int64:
			return Value{valueNumber, int64(value.Int())}
		case reflect.Uint:
			return Value{valueNumber, uint(value.Uint())}
		case reflect.Uint8:
			return Value{valueNumber, uint8(value.Uint())}
		case reflect.Uint16:
			return Value{valueNumber, uint16(value.Uint())}
		case reflect.Uint32:
			return Value{valueNumber, uint32(value.Uint())}
		case reflect.Uint64:
			return Value{valueNumber, uint64(value.Uint())}
		case reflect.Float32:
			return Value{valueNumber, float32(value.Float())}
		case reflect.Float64:
			return Value{valueNumber, float64(value.Float())}
		case reflect.String:
			return Value{valueString, string(value.String())}
		default:
			toValue_reflectValuePanic(value.Interface(), value.Kind())
		}
	default:
		return toValue(reflect.ValueOf(value))
	}

	panic(newError(nil, "TypeError", 0, "invalid value: %v (%T)", value, value))
}

func (value Value) String() string {
	result := ""
	catchPanic(func() {
		result = value.string()
	})
	return result
}

func (value Value) ToBoolean() (bool, error) {
	result := false
	err := catchPanic(func() {
		result = value.bool()
	})
	return result, err
}

func (value Value) numberValue() Value {
	if value.kind == valueNumber {
		return value
	}
	return Value{valueNumber, value.float64()}
}

func (value Value) ToFloat() (float64, error) {
	result := float64(0)
	err := catchPanic(func() {
		result = value.float64()
	})
	return result, err
}

func (value Value) ToInteger() (int64, error) {
	result := int64(0)
	err := catchPanic(func() {
		result = value.number().int64
	})
	return result, err
}

func (value Value) ToString() (string, error) {
	result := ""
	err := catchPanic(func() {
		result = value.string()
	})
	return result, err
}

func (value Value) _object() *_object {
	switch value := value.value.(type) {
	case *_object:
		return value
	}
	return nil
}

func (value Value) Object() *Object {
	switch object := value.value.(type) {
	case *_object:
		return _newObject(object, value)
	}
	return nil
}

func (value Value) reference() _reference {
	switch value := value.value.(type) {
	case _reference:
		return value
	}
	return nil
}

func (value Value) resolve() Value {
	switch value := value.value.(type) {
	case _reference:
		return value.getValue()
	}
	return value
}

var (
	__NaN__              float64 = math.NaN()
	__PositiveInfinity__ float64 = math.Inf(+1)
	__NegativeInfinity__ float64 = math.Inf(-1)
	__PositiveZero__     float64 = 0
	__NegativeZero__     float64 = math.Float64frombits(0 | (1 << 63))
)

func positiveInfinity() float64 {
	return __PositiveInfinity__
}

func negativeInfinity() float64 {
	return __NegativeInfinity__
}

func positiveZero() float64 {
	return __PositiveZero__
}

func negativeZero() float64 {
	return __NegativeZero__
}

func NaNValue() Value {
	return Value{valueNumber, __NaN__}
}

func positiveInfinityValue() Value {
	return Value{valueNumber, __PositiveInfinity__}
}

func negativeInfinityValue() Value {
	return Value{valueNumber, __NegativeInfinity__}
}

func positiveZeroValue() Value {
	return Value{valueNumber, __PositiveZero__}
}

func negativeZeroValue() Value {
	return Value{valueNumber, __NegativeZero__}
}

func TrueValue() Value {
	return Value{valueBoolean, true}
}

func FalseValue() Value {
	return Value{valueBoolean, false}
}

func sameValue(x Value, y Value) bool {
	if x.kind != y.kind {
		return false
	}
	result := false
	switch x.kind {
	case valueUndefined, valueNull:
		result = true
	case valueNumber:
		x := x.float64()
		y := y.float64()
		if math.IsNaN(x) && math.IsNaN(y) {
			result = true
		} else {
			result = x == y
			if result && x == 0 {

				result = math.Signbit(x) == math.Signbit(y)
			}
		}
	case valueString:
		result = x.string() == y.string()
	case valueBoolean:
		result = x.bool() == y.bool()
	case valueObject:
		result = x._object() == y._object()
	default:
		panic(hereBeDragons())
	}

	return result
}

func strictEqualityComparison(x Value, y Value) bool {
	if x.kind != y.kind {
		return false
	}
	result := false
	switch x.kind {
	case valueUndefined, valueNull:
		result = true
	case valueNumber:
		x := x.float64()
		y := y.float64()
		if math.IsNaN(x) && math.IsNaN(y) {
			result = false
		} else {
			result = x == y
		}
	case valueString:
		result = x.string() == y.string()
	case valueBoolean:
		result = x.bool() == y.bool()
	case valueObject:
		result = x._object() == y._object()
	default:
		panic(hereBeDragons())
	}

	return result
}

func (self Value) Export() (interface{}, error) {
	return self.export(), nil
}

func (self Value) export() interface{} {

	switch self.kind {
	case valueUndefined:
		return nil
	case valueNull:
		return nil
	case valueNumber, valueBoolean:
		return self.value
	case valueString:
		switch value := self.value.(type) {
		case string:
			return value
		case []uint16:
			return string(utf16.Decode(value))
		}
	case valueObject:
		object := self._object()
		switch value := object.value.(type) {
		case *_goStructObject:
			return value.value.Interface()
		case *_goMapObject:
			return value.value.Interface()
		case *_goArrayObject:
			return value.value.Interface()
		case *_goSliceObject:
			return value.value.Interface()
		}
		if object.class == "Array" {
			result := make([]interface{}, 0)
			lengthValue := object.get("length")
			length := lengthValue.value.(uint32)
			kind := reflect.Invalid
			state := 0
			var t reflect.Type
			for index := uint32(0); index < length; index += 1 {
				name := strconv.FormatInt(int64(index), 10)
				if !object.hasProperty(name) {
					continue
				}
				value := object.get(name).export()

				t = reflect.TypeOf(value)

				var k reflect.Kind
				if t != nil {
					k = t.Kind()
				}

				if state == 0 {
					kind = k
					state = 1
				} else if state == 1 && kind != k {
					state = 2
				}

				result = append(result, value)
			}

			if state != 1 || kind == reflect.Interface || t == nil {

				return result
			}

			val := reflect.MakeSlice(reflect.SliceOf(t), len(result), len(result))
			for i, v := range result {
				val.Index(i).Set(reflect.ValueOf(v))
			}
			return val.Interface()
		} else {
			result := make(map[string]interface{})

			object.enumerate(false, func(name string) bool {
				value := object.get(name)
				if value.IsDefined() {
					result[name] = value.export()
				}
				return true
			})
			return result
		}
	}

	if self.safe() {
		return self
	}

	return Value{}
}

func (self Value) evaluateBreakContinue(labels []string) _resultKind {
	result := self.value.(_result)
	if result.kind == resultBreak || result.kind == resultContinue {
		for _, label := range labels {
			if label == result.target {
				return result.kind
			}
		}
	}
	return resultReturn
}

func (self Value) evaluateBreak(labels []string) _resultKind {
	result := self.value.(_result)
	if result.kind == resultBreak {
		for _, label := range labels {
			if label == result.target {
				return result.kind
			}
		}
	}
	return resultReturn
}

func (self Value) exportNative() interface{} {

	switch self.kind {
	case valueUndefined:
		return self
	case valueNull:
		return nil
	case valueNumber, valueBoolean:
		return self.value
	case valueString:
		switch value := self.value.(type) {
		case string:
			return value
		case []uint16:
			return string(utf16.Decode(value))
		}
	case valueObject:
		object := self._object()
		switch value := object.value.(type) {
		case *_goStructObject:
			return value.value.Interface()
		case *_goMapObject:
			return value.value.Interface()
		case *_goArrayObject:
			return value.value.Interface()
		case *_goSliceObject:
			return value.value.Interface()
		}
	}

	return self
}

func (value Value) toReflectValue(kind reflect.Kind) (reflect.Value, error) {
	if kind != reflect.Float32 && kind != reflect.Float64 && kind != reflect.Interface {
		switch value := value.value.(type) {
		case float32:
			_, frac := math.Modf(float64(value))
			if frac > 0 {
				return reflect.Value{}, fmt.Errorf("RangeError: %v to reflect.Kind: %v", value, kind)
			}
		case float64:
			_, frac := math.Modf(value)
			if frac > 0 {
				return reflect.Value{}, fmt.Errorf("RangeError: %v to reflect.Kind: %v", value, kind)
			}
		}
	}

	switch kind {
	case reflect.Bool: 
		return reflect.ValueOf(value.bool()), nil
	case reflect.Int: 

		tmp := toIntegerFloat(value)
		if tmp < float_minInt || tmp > float_maxInt {
			return reflect.Value{}, fmt.Errorf("RangeError: %f (%v) to int", tmp, value)
		} else {
			return reflect.ValueOf(int(tmp)), nil
		}
	case reflect.Int8: 
		tmp := value.number().int64
		if tmp < int64_minInt8 || tmp > int64_maxInt8 {
			return reflect.Value{}, fmt.Errorf("RangeError: %d (%v) to int8", tmp, value)
		} else {
			return reflect.ValueOf(int8(tmp)), nil
		}
	case reflect.Int16: 
		tmp := value.number().int64
		if tmp < int64_minInt16 || tmp > int64_maxInt16 {
			return reflect.Value{}, fmt.Errorf("RangeError: %d (%v) to int16", tmp, value)
		} else {
			return reflect.ValueOf(int16(tmp)), nil
		}
	case reflect.Int32: 
		tmp := value.number().int64
		if tmp < int64_minInt32 || tmp > int64_maxInt32 {
			return reflect.Value{}, fmt.Errorf("RangeError: %d (%v) to int32", tmp, value)
		} else {
			return reflect.ValueOf(int32(tmp)), nil
		}
	case reflect.Int64: 

		tmp := toIntegerFloat(value)
		if tmp < float_minInt64 || tmp > float_maxInt64 {
			return reflect.Value{}, fmt.Errorf("RangeError: %f (%v) to int", tmp, value)
		} else {
			return reflect.ValueOf(int64(tmp)), nil
		}
	case reflect.Uint: 

		tmp := toIntegerFloat(value)
		if tmp < 0 || tmp > float_maxUint {
			return reflect.Value{}, fmt.Errorf("RangeError: %f (%v) to uint", tmp, value)
		} else {
			return reflect.ValueOf(uint(tmp)), nil
		}
	case reflect.Uint8: 
		tmp := value.number().int64
		if tmp < 0 || tmp > int64_maxUint8 {
			return reflect.Value{}, fmt.Errorf("RangeError: %d (%v) to uint8", tmp, value)
		} else {
			return reflect.ValueOf(uint8(tmp)), nil
		}
	case reflect.Uint16: 
		tmp := value.number().int64
		if tmp < 0 || tmp > int64_maxUint16 {
			return reflect.Value{}, fmt.Errorf("RangeError: %d (%v) to uint16", tmp, value)
		} else {
			return reflect.ValueOf(uint16(tmp)), nil
		}
	case reflect.Uint32: 
		tmp := value.number().int64
		if tmp < 0 || tmp > int64_maxUint32 {
			return reflect.Value{}, fmt.Errorf("RangeError: %d (%v) to uint32", tmp, value)
		} else {
			return reflect.ValueOf(uint32(tmp)), nil
		}
	case reflect.Uint64: 

		tmp := toIntegerFloat(value)
		if tmp < 0 || tmp > float_maxUint64 {
			return reflect.Value{}, fmt.Errorf("RangeError: %f (%v) to uint64", tmp, value)
		} else {
			return reflect.ValueOf(uint64(tmp)), nil
		}
	case reflect.Float32: 
		tmp := value.float64()
		tmp1 := tmp
		if 0 > tmp1 {
			tmp1 = -tmp1
		}
		if tmp1 > 0 && (tmp1 < math.SmallestNonzeroFloat32 || tmp1 > math.MaxFloat32) {
			return reflect.Value{}, fmt.Errorf("RangeError: %f (%v) to float32", tmp, value)
		} else {
			return reflect.ValueOf(float32(tmp)), nil
		}
	case reflect.Float64: 
		value := value.float64()
		return reflect.ValueOf(float64(value)), nil
	case reflect.String: 
		return reflect.ValueOf(value.string()), nil
	case reflect.Invalid: 
	case reflect.Complex64: 
	case reflect.Complex128: 
	case reflect.Chan: 
	case reflect.Func: 
	case reflect.Ptr: 
	case reflect.UnsafePointer: 
	default:
		switch value.kind {
		case valueObject:
			object := value._object()
			switch vl := object.value.(type) {
			case *_goStructObject: 
				return reflect.ValueOf(vl.value.Interface()), nil
			case *_goMapObject: 
				return reflect.ValueOf(vl.value.Interface()), nil
			case *_goArrayObject: 
				return reflect.ValueOf(vl.value.Interface()), nil
			case *_goSliceObject: 
				return reflect.ValueOf(vl.value.Interface()), nil
			}
			return reflect.ValueOf(value.exportNative()), nil
		case valueEmpty, valueResult, valueReference:

		default:
			return reflect.ValueOf(value.value), nil
		}
	}

	panic(fmt.Errorf("invalid conversion of %v (%v) to reflect.Kind: %v", value.kind, value, kind))
}

func stringToReflectValue(value string, kind reflect.Kind) (reflect.Value, error) {
	switch kind {
	case reflect.Bool:
		value, err := strconv.ParseBool(value)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(value), nil
	case reflect.Int:
		value, err := strconv.ParseInt(value, 0, 0)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(int(value)), nil
	case reflect.Int8:
		value, err := strconv.ParseInt(value, 0, 8)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(int8(value)), nil
	case reflect.Int16:
		value, err := strconv.ParseInt(value, 0, 16)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(int16(value)), nil
	case reflect.Int32:
		value, err := strconv.ParseInt(value, 0, 32)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(int32(value)), nil
	case reflect.Int64:
		value, err := strconv.ParseInt(value, 0, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(int64(value)), nil
	case reflect.Uint:
		value, err := strconv.ParseUint(value, 0, 0)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(uint(value)), nil
	case reflect.Uint8:
		value, err := strconv.ParseUint(value, 0, 8)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(uint8(value)), nil
	case reflect.Uint16:
		value, err := strconv.ParseUint(value, 0, 16)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(uint16(value)), nil
	case reflect.Uint32:
		value, err := strconv.ParseUint(value, 0, 32)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(uint32(value)), nil
	case reflect.Uint64:
		value, err := strconv.ParseUint(value, 0, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(uint64(value)), nil
	case reflect.Float32:
		value, err := strconv.ParseFloat(value, 32)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(float32(value)), nil
	case reflect.Float64:
		value, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(float64(value)), nil
	case reflect.String:
		return reflect.ValueOf(value), nil
	}

	panic(fmt.Errorf("invalid conversion of %q to reflect.Kind: %v", value, kind))
}
