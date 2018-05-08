package assert

type Assertions struct {
	t TestingT
}

func New(t TestingT) *Assertions {
	return &Assertions{
		t: t,
	}
}

//go:generate go run ../_codegen/main.go -output-package=assert -template=assertion_forward.go.tmpl -include-format-funcs
