
// +build !appengine
// +build gc
// +build !noasm

package snappy

//go:noescape
func emitLiteral(dst, lit []byte) int

//go:noescape
func emitCopy(dst []byte, offset, length int) int

//go:noescape
func extendMatch(src []byte, i, j int) int

//go:noescape
func encodeBlock(dst, src []byte) (d int)
