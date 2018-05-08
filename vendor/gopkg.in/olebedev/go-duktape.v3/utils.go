package duktape

func (d *Context) Must() *Context {
	if d.duk_context == nil {
		panic("[duktape] Context does not exists!\nYou cannot call any contexts methods after `DestroyHeap()` was called.")
	}
	return d
}
