package otto

type _scope struct {
	lexical  _stash
	variable _stash
	this     *_object
	eval     bool 
	outer    *_scope
	depth    int

	frame _frame
}

func newScope(lexical _stash, variable _stash, this *_object) *_scope {
	return &_scope{
		lexical:  lexical,
		variable: variable,
		this:     this,
	}
}
