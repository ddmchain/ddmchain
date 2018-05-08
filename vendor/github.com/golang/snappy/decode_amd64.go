
// +build !appengine
// +build gc
// +build !noasm

package snappy

//go:noescape
func decode(dst, src []byte) int
