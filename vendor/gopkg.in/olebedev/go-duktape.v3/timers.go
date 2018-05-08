package duktape

import (
	"errors"
	"fmt"
	"time"
)

func (d *Context) PushTimers() error {
	d.PushGlobalStash()

	if !d.HasPropString(-1, "timers") {
		d.PushObject()
		d.PutPropString(-2, "timers") 
		d.Pop()

		d.PushGlobalGoFunction("setTimeout", setTimeout)
		d.PushGlobalGoFunction("setInterval", setInterval)
		d.PushGlobalGoFunction("clearTimeout", clearTimeout)
		d.PushGlobalGoFunction("clearInterval", clearTimeout)
		return nil
	} else {
		d.Pop()
		return errors.New("Timers are already defined")
	}
}

func (d *Context) FlushTimers() {
	d.PushGlobalStash()
	d.PushObject()
	d.PutPropString(-2, "timers") 
	d.Pop()
}

func setTimeout(c *Context) int {
	id := c.pushTimer(0)
	timeout := c.ToNumber(1)
	if timeout < 1 {
		timeout = 1
	}
	go func(id float64) {
		<-time.After(time.Duration(timeout) * time.Millisecond)
		c.Lock()
		defer c.Unlock()
		if c.duk_context == nil {
			fmt.Println("[duktape] Warning!\nsetTimeout invokes callback after the context was destroyed.")
			return
		}

		c.putTimer(id)
		if c.GetType(-1).IsObject() {
			c.Pcall(0 )
		}
		c.dropTimer(id)
	}(id)
	c.PushNumber(id)
	return 1
}

func clearTimeout(c *Context) int {
	if c.GetType(0).IsNumber() {
		c.dropTimer(c.GetNumber(0))
		c.Pop()
	}
	return 0
}

func setInterval(c *Context) int {
	id := c.pushTimer(0)
	timeout := c.ToNumber(1)
	if timeout < 1 {
		timeout = 1
	}
	go func(id float64) {
		ticker := time.NewTicker(time.Duration(timeout) * time.Millisecond)
		for _ = range ticker.C {
			c.Lock()

			if c.duk_context == nil {
				c.dropTimer(id)
				c.Pop()
				ticker.Stop()
				fmt.Println("[duktape] Warning!\nsetInterval invokes callback after the context was destroyed.")
				c.Unlock()
				continue
			}

			c.putTimer(id)
			if c.GetType(-1).IsObject() {
				c.Pcall(0 )
				c.Pop()
			} else {
				c.dropTimer(id)
				c.Pop()
				ticker.Stop()
			}
			c.Unlock()
		}
	}(id)
	c.PushNumber(id)
	return 1
}

func (d *Context) pushTimer(index int) float64 {
	id := d.timerIndex.get()

	d.PushGlobalStash()
	d.GetPropString(-1, "timers")
	d.PushNumber(id)
	d.Dup(index)
	d.PutProp(-3)
	d.Pop2()

	return id
}

func (d *Context) dropTimer(id float64) {
	d.PushGlobalStash()
	d.GetPropString(-1, "timers")
	d.PushNumber(id)
	d.DelProp(-2)
	d.Pop2()
}

func (d *Context) putTimer(id float64) {
	d.PushGlobalStash()           
	d.GetPropString(-1, "timers") 
	d.PushNumber(id)
	d.GetProp(-2) 
	d.Replace(-3)
	d.Pop()
}
