
package curve25519 

var basePoint = [32]byte{9, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

func ScalarMult(dst, in, base *[32]byte) {
	scalarMult(dst, in, base)
}

func ScalarBaseMult(dst, in *[32]byte) {
	ScalarMult(dst, in, &basePoint)
}
