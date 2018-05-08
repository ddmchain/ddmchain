package otto

import (
	"fmt"
	"regexp"
	runtime_ "runtime"
	"strconv"
	"strings"
)

var isIdentifier_Regexp *regexp.Regexp = regexp.MustCompile(`^[a-zA-Z\$][a-zA-Z0-9\$]*$`)

func isIdentifier(string_ string) bool {
	return isIdentifier_Regexp.MatchString(string_)
}

func (self *_runtime) toValueArray(arguments ...interface{}) []Value {
	length := len(arguments)
	if length == 1 {
		if valueArray, ok := arguments[0].([]Value); ok {
			return valueArray
		}
		return []Value{self.toValue(arguments[0])}
	}

	valueArray := make([]Value, length)
	for index, value := range arguments {
		valueArray[index] = self.toValue(value)
	}

	return valueArray
}

func stringToArrayIndex(name string) int64 {
	index, err := strconv.ParseInt(name, 10, 64)
	if err != nil {
		return -1
	}
	if index < 0 {
		return -1
	}
	if index >= maxUint32 {

		return -1
	}
	return index
}

func isUint32(value int64) bool {
	return value >= 0 && value <= maxUint32
}

func arrayIndexToString(index int64) string {
	return strconv.FormatInt(index, 10)
}

func valueOfArrayIndex(array []Value, index int) Value {
	value, _ := getValueOfArrayIndex(array, index)
	return value
}

func getValueOfArrayIndex(array []Value, index int) (Value, bool) {
	if index >= 0 && index < len(array) {
		value := array[index]
		if !value.isEmpty() {
			return value, true
		}
	}
	return Value{}, false
}

func valueToRangeIndex(indexValue Value, length int64, negativeIsZero bool) int64 {
	index := indexValue.number().int64
	if negativeIsZero {
		if index < 0 {
			index = 0
		}

		if index >= length {
			index = length
		}
		return index
	}

	if index < 0 {
		index += length
		if index < 0 {
			index = 0
		}
	} else {
		if index > length {
			index = length
		}
	}
	return index
}

func rangeStartEnd(array []Value, size int64, negativeIsZero bool) (start, end int64) {
	start = valueToRangeIndex(valueOfArrayIndex(array, 0), size, negativeIsZero)
	if len(array) == 1 {

		end = size
		return
	}

	end = size
	endValue := valueOfArrayIndex(array, 1)
	if !endValue.IsUndefined() {

		end = valueToRangeIndex(endValue, size, negativeIsZero)
	}
	return
}

func rangeStartLength(source []Value, size int64) (start, length int64) {
	start = valueToRangeIndex(valueOfArrayIndex(source, 0), size, false)

	length = int64(size)
	if len(source) == 1 {

		return
	}

	lengthValue := valueOfArrayIndex(source, 1)
	if !lengthValue.IsUndefined() {

		length = lengthValue.number().int64
	}
	return
}

func boolFields(input string) (result map[string]bool) {
	result = map[string]bool{}
	for _, word := range strings.Fields(input) {
		result[word] = true
	}
	return result
}

func hereBeDragons(arguments ...interface{}) string {
	pc, _, _, _ := runtime_.Caller(1)
	name := runtime_.FuncForPC(pc).Name()
	message := fmt.Sprintf("Here be dragons -- %s", name)
	if len(arguments) > 0 {
		message += ": "
		argument0 := fmt.Sprintf("%s", arguments[0])
		if len(arguments) == 1 {
			message += argument0
		} else {
			message += fmt.Sprintf(argument0, arguments[1:]...)
		}
	} else {
		message += "."
	}
	return message
}

func throwHereBeDragons(arguments ...interface{}) {
	panic(hereBeDragons(arguments...))
}

func eachPair(list []interface{}, fn func(_0, _1 interface{})) {
	for len(list) > 0 {
		var _0, _1 interface{}
		_0 = list[0]
		list = list[1:] 
		if len(list) > 0 {
			_1 = list[0]
			list = list[1:] 
		}
		fn(_0, _1)
	}
}
