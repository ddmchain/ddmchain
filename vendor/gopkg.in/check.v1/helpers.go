package check

import (
	"fmt"
	"strings"
	"time"
)

func (c *C) TestName() string {
	return c.testName
}

func (c *C) Failed() bool {
	return c.status() == failedSt
}

func (c *C) Fail() {
	c.setStatus(failedSt)
}

func (c *C) FailNow() {
	c.Fail()
	c.stopNow()
}

func (c *C) Succeed() {
	c.setStatus(succeededSt)
}

func (c *C) SucceedNow() {
	c.Succeed()
	c.stopNow()
}

func (c *C) ExpectFailure(reason string) {
	if reason == "" {
		panic("Missing reason why the test is expected to fail")
	}
	c.mustFail = true
	c.reason = reason
}

func (c *C) Skip(reason string) {
	if reason == "" {
		panic("Missing reason why the test is being skipped")
	}
	c.reason = reason
	c.setStatus(skippedSt)
	c.stopNow()
}

func (c *C) GetTestLog() string {
	return c.logb.String()
}

func (c *C) Log(args ...interface{}) {
	c.log(args...)
}

func (c *C) Logf(format string, args ...interface{}) {
	c.logf(format, args...)
}

func (c *C) Output(calldepth int, s string) error {
	d := time.Now().Sub(c.startTime)
	msec := d / time.Millisecond
	sec := d / time.Second
	min := d / time.Minute

	c.Logf("[LOG] %d:%02d.%03d %s", min, sec%60, msec%1000, s)
	return nil
}

func (c *C) Error(args ...interface{}) {
	c.logCaller(1)
	c.logString(fmt.Sprint("Error: ", fmt.Sprint(args...)))
	c.logNewLine()
	c.Fail()
}

func (c *C) Errorf(format string, args ...interface{}) {
	c.logCaller(1)
	c.logString(fmt.Sprintf("Error: "+format, args...))
	c.logNewLine()
	c.Fail()
}

func (c *C) Fatal(args ...interface{}) {
	c.logCaller(1)
	c.logString(fmt.Sprint("Error: ", fmt.Sprint(args...)))
	c.logNewLine()
	c.FailNow()
}

func (c *C) Fatalf(format string, args ...interface{}) {
	c.logCaller(1)
	c.logString(fmt.Sprint("Error: ", fmt.Sprintf(format, args...)))
	c.logNewLine()
	c.FailNow()
}

func (c *C) Check(obtained interface{}, checker Checker, args ...interface{}) bool {
	return c.internalCheck("Check", obtained, checker, args...)
}

func (c *C) Assert(obtained interface{}, checker Checker, args ...interface{}) {
	if !c.internalCheck("Assert", obtained, checker, args...) {
		c.stopNow()
	}
}

func (c *C) internalCheck(funcName string, obtained interface{}, checker Checker, args ...interface{}) bool {
	if checker == nil {
		c.logCaller(2)
		c.logString(fmt.Sprintf("%s(obtained, nil!?, ...):", funcName))
		c.logString("Oops.. you've provided a nil checker!")
		c.logNewLine()
		c.Fail()
		return false
	}

	var comment CommentInterface
	if len(args) > 0 {
		if c, ok := args[len(args)-1].(CommentInterface); ok {
			comment = c
			args = args[:len(args)-1]
		}
	}

	params := append([]interface{}{obtained}, args...)
	info := checker.Info()

	if len(params) != len(info.Params) {
		names := append([]string{info.Params[0], info.Name}, info.Params[1:]...)
		c.logCaller(2)
		c.logString(fmt.Sprintf("%s(%s):", funcName, strings.Join(names, ", ")))
		c.logString(fmt.Sprintf("Wrong number of parameters for %s: want %d, got %d", info.Name, len(names), len(params)+1))
		c.logNewLine()
		c.Fail()
		return false
	}

	names := append([]string{}, info.Params...)

	result, error := checker.Check(params, names)
	if !result || error != "" {
		c.logCaller(2)
		for i := 0; i != len(params); i++ {
			c.logValue(names[i], params[i])
		}
		if comment != nil {
			c.logString(comment.CheckCommentString())
		}
		if error != "" {
			c.logString(error)
		}
		c.logNewLine()
		c.Fail()
		return false
	}
	return true
}
