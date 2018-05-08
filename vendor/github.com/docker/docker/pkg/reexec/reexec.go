package reexec

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

var registeredInitializers = make(map[string]func())

func Register(name string, initializer func()) {
	if _, exists := registeredInitializers[name]; exists {
		panic(fmt.Sprintf("reexec func already registered under name %q", name))
	}

	registeredInitializers[name] = initializer
}

func Init() bool {
	initializer, exists := registeredInitializers[os.Args[0]]
	if exists {
		initializer()

		return true
	}
	return false
}

func naiveSelf() string {
	name := os.Args[0]
	if filepath.Base(name) == name {
		if lp, err := exec.LookPath(name); err == nil {
			return lp
		}
	}

	if absName, err := filepath.Abs(name); err == nil {
		return absName
	}

	return name
}
