
// +build gensecp256k1

package btcec

import (
	"encoding/binary"
	"math/big"
)

var secp256k1BytePoints = ""

func (curve *KoblitzCurve) getDoublingPoints() [][3]fieldVal {
	doublingPoints := make([][3]fieldVal, curve.BitSize)

	px, py := curve.bigAffineToField(curve.Gx, curve.Gy)
	pz := new(fieldVal).SetInt(1)
	for i := 0; i < curve.BitSize; i++ {
		doublingPoints[i] = [3]fieldVal{*px, *py, *pz}

		curve.doubleJacobian(px, py, pz, px, py, pz)
	}
	return doublingPoints
}

func (curve *KoblitzCurve) SerializedBytePoints() []byte {
	doublingPoints := curve.getDoublingPoints()

	serialized := make([]byte, curve.byteSize*256*3*10*4)
	offset := 0
	for byteNum := 0; byteNum < curve.byteSize; byteNum++ {

		startingBit := 8 * (curve.byteSize - byteNum - 1)
		computingPoints := doublingPoints[startingBit : startingBit+8]

		for i := 0; i < 256; i++ {
			px, py, pz := new(fieldVal), new(fieldVal), new(fieldVal)
			for j := 0; j < 8; j++ {
				if i>>uint(j)&1 == 1 {
					curve.addJacobian(px, py, pz, &computingPoints[j][0],
						&computingPoints[j][1], &computingPoints[j][2], px, py, pz)
				}
			}
			for i := 0; i < 10; i++ {
				binary.LittleEndian.PutUint32(serialized[offset:], px.n[i])
				offset += 4
			}
			for i := 0; i < 10; i++ {
				binary.LittleEndian.PutUint32(serialized[offset:], py.n[i])
				offset += 4
			}
			for i := 0; i < 10; i++ {
				binary.LittleEndian.PutUint32(serialized[offset:], pz.n[i])
				offset += 4
			}
		}
	}

	return serialized
}

func sqrt(n *big.Int) *big.Int {

	guess := big.NewInt(2)
	guess.Exp(guess, big.NewInt(int64(n.BitLen()/2)), nil)

	big2 := big.NewInt(2)
	prevGuess := big.NewInt(0)
	for {
		prevGuess.Set(guess)
		guess.Add(guess, new(big.Int).Div(n, guess))
		guess.Div(guess, big2)
		if guess.Cmp(prevGuess) == 0 {
			break
		}
	}
	return guess
}

func (curve *KoblitzCurve) EndomorphismVectors() (a1, b1, a2, b2 *big.Int) {
	bigMinus1 := big.NewInt(-1)

	nSqrt := sqrt(curve.N)
	u, v := new(big.Int).Set(curve.N), new(big.Int).Set(curve.lambda)
	x1, y1 := big.NewInt(1), big.NewInt(0)
	x2, y2 := big.NewInt(0), big.NewInt(1)
	q, r := new(big.Int), new(big.Int)
	qu, qx1, qy1 := new(big.Int), new(big.Int), new(big.Int)
	s, t := new(big.Int), new(big.Int)
	ri, ti := new(big.Int), new(big.Int)
	a1, b1, a2, b2 = new(big.Int), new(big.Int), new(big.Int), new(big.Int)
	found, oneMore := false, false
	for u.Sign() != 0 {

		q.Div(v, u)

		qu.Mul(q, u)
		r.Sub(v, qu)

		qx1.Mul(q, x1)
		s.Sub(x2, qx1)

		qy1.Mul(q, y1)
		t.Sub(y2, qy1)

		v.Set(u)
		u.Set(r)
		x2.Set(x1)
		x1.Set(s)
		y2.Set(y1)
		y1.Set(t)

		if !found && r.Cmp(nSqrt) < 0 {

			a1.Set(r)
			b1.Mul(t, bigMinus1)
			found = true
			oneMore = true

			continue

		} else if oneMore {

			rSquared := new(big.Int).Mul(ri, ri)
			tSquared := new(big.Int).Mul(ti, ti)
			sum1 := new(big.Int).Add(rSquared, tSquared)

			r2Squared := new(big.Int).Mul(r, r)
			t2Squared := new(big.Int).Mul(t, t)
			sum2 := new(big.Int).Add(r2Squared, t2Squared)

			if sum1.Cmp(sum2) <= 0 {

				a2.Set(ri)
				b2.Mul(ti, bigMinus1)
			} else {

				a2.Set(r)
				b2.Mul(t, bigMinus1)
			}

			break
		}

		ri.Set(r)
		ti.Set(t)
	}

	return a1, b1, a2, b2
}
