
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build amd64,!appengine,!gccgo

package sha3

// This function is implemented in keccakf_amd64.s.

//go:noescape

func keccakF1600(state *[25]uint64)
