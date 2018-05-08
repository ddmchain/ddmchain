package check

import (
	"fmt"
	"reflect"
	"regexp"
)

type comment struct {
	format string
	args   []interface{}
}

func Commentf(format string, args ...interface{}) CommentInterface {
	return &comment{format, args}
}

type CommentInterface interface {
	CheckCommentString() string
}

func (c *comment) CheckCommentString() string {
	return fmt.Sprintf(c.format, c.args...)
}

type Checker interface {
	Info() *CheckerInfo
	Check(params []interface{}, names []string) (result bool, error string)
}

type CheckerInfo struct {
	Name   string
	Params []string
}

func (info *CheckerInfo) Info() *CheckerInfo {
	return info
}

func Not(checker Checker) Checker {
	return &notChecker{checker}
}

type notChecker struct {
	sub Checker
}

func (checker *notChecker) Info() *CheckerInfo {
	info := *checker.sub.Info()
	info.Name = "Not(" + info.Name + ")"
	return &info
}

func (checker *notChecker) Check(params []interface{}, names []string) (result bool, error string) {
	result, error = checker.sub.Check(params, names)
	result = !result
	return
}

type isNilChecker struct {
	*CheckerInfo
}

var IsNil Checker = &isNilChecker{
	&CheckerInfo{Name: "IsNil", Params: []string{"value"}},
}

func (checker *isNilChecker) Check(params []interface{}, names []string) (result bool, error string) {
	return isNil(params[0]), ""
}

func isNil(obtained interface{}) (result bool) {
	if obtained == nil {
		result = true
	} else {
		switch v := reflect.ValueOf(obtained); v.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
			return v.IsNil()
		}
	}
	return
}

type notNilChecker struct {
	*CheckerInfo
}

var NotNil Checker = &notNilChecker{
	&CheckerInfo{Name: "NotNil", Params: []string{"value"}},
}

func (checker *notNilChecker) Check(params []interface{}, names []string) (result bool, error string) {
	return !isNil(params[0]), ""
}

type equalsChecker struct {
	*CheckerInfo
}

var Equals Checker = &equalsChecker{
	&CheckerInfo{Name: "Equals", Params: []string{"obtained", "expected"}},
}

func (checker *equalsChecker) Check(params []interface{}, names []string) (result bool, error string) {
	defer func() {
		if v := recover(); v != nil {
			result = false
			error = fmt.Sprint(v)
		}
	}()
	return params[0] == params[1], ""
}

type deepEqualsChecker struct {
	*CheckerInfo
}

var DeepEquals Checker = &deepEqualsChecker{
	&CheckerInfo{Name: "DeepEquals", Params: []string{"obtained", "expected"}},
}

func (checker *deepEqualsChecker) Check(params []interface{}, names []string) (result bool, error string) {
	return reflect.DeepEqual(params[0], params[1]), ""
}

type hasLenChecker struct {
	*CheckerInfo
}

var HasLen Checker = &hasLenChecker{
	&CheckerInfo{Name: "HasLen", Params: []string{"obtained", "n"}},
}

func (checker *hasLenChecker) Check(params []interface{}, names []string) (result bool, error string) {
	n, ok := params[1].(int)
	if !ok {
		return false, "n must be an int"
	}
	value := reflect.ValueOf(params[0])
	switch value.Kind() {
	case reflect.Map, reflect.Array, reflect.Slice, reflect.Chan, reflect.String:
	default:
		return false, "obtained value type has no length"
	}
	return value.Len() == n, ""
}

type errorMatchesChecker struct {
	*CheckerInfo
}

var ErrorMatches Checker = errorMatchesChecker{
	&CheckerInfo{Name: "ErrorMatches", Params: []string{"value", "regex"}},
}

func (checker errorMatchesChecker) Check(params []interface{}, names []string) (result bool, errStr string) {
	if params[0] == nil {
		return false, "Error value is nil"
	}
	err, ok := params[0].(error)
	if !ok {
		return false, "Value is not an error"
	}
	params[0] = err.Error()
	names[0] = "error"
	return matches(params[0], params[1])
}

type matchesChecker struct {
	*CheckerInfo
}

var Matches Checker = &matchesChecker{
	&CheckerInfo{Name: "Matches", Params: []string{"value", "regex"}},
}

func (checker *matchesChecker) Check(params []interface{}, names []string) (result bool, error string) {
	return matches(params[0], params[1])
}

func matches(value, regex interface{}) (result bool, error string) {
	reStr, ok := regex.(string)
	if !ok {
		return false, "Regex must be a string"
	}
	valueStr, valueIsStr := value.(string)
	if !valueIsStr {
		if valueWithStr, valueHasStr := value.(fmt.Stringer); valueHasStr {
			valueStr, valueIsStr = valueWithStr.String(), true
		}
	}
	if valueIsStr {
		matches, err := regexp.MatchString("^"+reStr+"$", valueStr)
		if err != nil {
			return false, "Can't compile regex: " + err.Error()
		}
		return matches, ""
	}
	return false, "Obtained value is not a string and has no .String()"
}

type panicsChecker struct {
	*CheckerInfo
}

var Panics Checker = &panicsChecker{
	&CheckerInfo{Name: "Panics", Params: []string{"function", "expected"}},
}

func (checker *panicsChecker) Check(params []interface{}, names []string) (result bool, error string) {
	f := reflect.ValueOf(params[0])
	if f.Kind() != reflect.Func || f.Type().NumIn() != 0 {
		return false, "Function must take zero arguments"
	}
	defer func() {

		if error != "" {
			return
		}
		params[0] = recover()
		names[0] = "panic"
		result = reflect.DeepEqual(params[0], params[1])
	}()
	f.Call(nil)
	return false, "Function has not panicked"
}

type panicMatchesChecker struct {
	*CheckerInfo
}

var PanicMatches Checker = &panicMatchesChecker{
	&CheckerInfo{Name: "PanicMatches", Params: []string{"function", "expected"}},
}

func (checker *panicMatchesChecker) Check(params []interface{}, names []string) (result bool, errmsg string) {
	f := reflect.ValueOf(params[0])
	if f.Kind() != reflect.Func || f.Type().NumIn() != 0 {
		return false, "Function must take zero arguments"
	}
	defer func() {

		if errmsg != "" {
			return
		}
		obtained := recover()
		names[0] = "panic"
		if e, ok := obtained.(error); ok {
			params[0] = e.Error()
		} else if _, ok := obtained.(string); ok {
			params[0] = obtained
		} else {
			errmsg = "Panic value is not a string or an error"
			return
		}
		result, errmsg = matches(params[0], params[1])
	}()
	f.Call(nil)
	return false, "Function has not panicked"
}

type fitsTypeChecker struct {
	*CheckerInfo
}

var FitsTypeOf Checker = &fitsTypeChecker{
	&CheckerInfo{Name: "FitsTypeOf", Params: []string{"obtained", "sample"}},
}

func (checker *fitsTypeChecker) Check(params []interface{}, names []string) (result bool, error string) {
	obtained := reflect.ValueOf(params[0])
	sample := reflect.ValueOf(params[1])
	if !obtained.IsValid() {
		return false, ""
	}
	if !sample.IsValid() {
		return false, "Invalid sample value"
	}
	return obtained.Type().AssignableTo(sample.Type()), ""
}

type implementsChecker struct {
	*CheckerInfo
}

var Implements Checker = &implementsChecker{
	&CheckerInfo{Name: "Implements", Params: []string{"obtained", "ifaceptr"}},
}

func (checker *implementsChecker) Check(params []interface{}, names []string) (result bool, error string) {
	obtained := reflect.ValueOf(params[0])
	ifaceptr := reflect.ValueOf(params[1])
	if !obtained.IsValid() {
		return false, ""
	}
	if !ifaceptr.IsValid() || ifaceptr.Kind() != reflect.Ptr || ifaceptr.Elem().Kind() != reflect.Interface {
		return false, "ifaceptr should be a pointer to an interface variable"
	}
	return obtained.Type().Implements(ifaceptr.Elem().Type()), ""
}
