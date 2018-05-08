
package btcec

import (
	"crypto/elliptic"
	"math/big"
	"sync"
)

var (

	fieldOne = new(fieldVal).SetInt(1)
)

type KoblitzCurve struct {
	*elliptic.CurveParams
	q         *big.Int
	H         int      
	halfOrder *big.Int 

	byteSize int

	bytePoints *[32][256][3]fieldVal

	lambda *big.Int

	beta *fieldVal

	a1 *big.Int
	b1 *big.Int
	a2 *big.Int
	b2 *big.Int
}

func (curve *KoblitzCurve) Params() *elliptic.CurveParams {
	return curve.CurveParams
}

func (curve *KoblitzCurve) bigAffineToField(x, y *big.Int) (*fieldVal, *fieldVal) {
	x3, y3 := new(fieldVal), new(fieldVal)
	x3.SetByteSlice(x.Bytes())
	y3.SetByteSlice(y.Bytes())

	return x3, y3
}

func (curve *KoblitzCurve) fieldJacobianToBigAffine(x, y, z *fieldVal) (*big.Int, *big.Int) {

	var zInv, tempZ fieldVal
	zInv.Set(z).Inverse()   
	tempZ.SquareVal(&zInv)  
	x.Mul(&tempZ)           
	y.Mul(tempZ.Mul(&zInv)) 
	z.SetInt(1)             

	x.Normalize()
	y.Normalize()

	x3, y3 := new(big.Int), new(big.Int)
	x3.SetBytes(x.Bytes()[:])
	y3.SetBytes(y.Bytes()[:])
	return x3, y3
}

func (curve *KoblitzCurve) IsOnCurve(x, y *big.Int) bool {

	fx, fy := curve.bigAffineToField(x, y)

	y2 := new(fieldVal).SquareVal(fy).Normalize()
	result := new(fieldVal).SquareVal(fx).Mul(fx).AddInt(7).Normalize()
	return y2.Equals(result)
}

func (curve *KoblitzCurve) addZ1AndZ2EqualsOne(x1, y1, z1, x2, y2, x3, y3, z3 *fieldVal) {

	x1.Normalize()
	y1.Normalize()
	x2.Normalize()
	y2.Normalize()
	if x1.Equals(x2) {
		if y1.Equals(y2) {

			curve.doubleJacobian(x1, y1, z1, x3, y3, z3)
			return
		}

		x3.SetInt(0)
		y3.SetInt(0)
		z3.SetInt(0)
		return
	}

	var h, i, j, r, v fieldVal
	var negJ, neg2V, negX3 fieldVal
	h.Set(x1).Negate(1).Add(x2)                
	i.SquareVal(&h).MulInt(4)                  
	j.Mul2(&h, &i)                             
	r.Set(y1).Negate(1).Add(y2).MulInt(2)      
	v.Mul2(x1, &i)                             
	negJ.Set(&j).Negate(1)                     
	neg2V.Set(&v).MulInt(2).Negate(2)          
	x3.Set(&r).Square().Add(&negJ).Add(&neg2V) 
	negX3.Set(x3).Negate(6)                    
	j.Mul(y1).MulInt(2).Negate(2)              
	y3.Set(&v).Add(&negX3).Mul(&r).Add(&j)     
	z3.Set(&h).MulInt(2)                       

	x3.Normalize()
	y3.Normalize()
	z3.Normalize()
}

func (curve *KoblitzCurve) addZ1EqualsZ2(x1, y1, z1, x2, y2, x3, y3, z3 *fieldVal) {

	x1.Normalize()
	y1.Normalize()
	x2.Normalize()
	y2.Normalize()
	if x1.Equals(x2) {
		if y1.Equals(y2) {

			curve.doubleJacobian(x1, y1, z1, x3, y3, z3)
			return
		}

		x3.SetInt(0)
		y3.SetInt(0)
		z3.SetInt(0)
		return
	}

	var a, b, c, d, e, f fieldVal
	var negX1, negY1, negE, negX3 fieldVal
	negX1.Set(x1).Negate(1)                
	negY1.Set(y1).Negate(1)                
	a.Set(&negX1).Add(x2)                  
	b.SquareVal(&a)                        
	c.Set(&negY1).Add(y2)                  
	d.SquareVal(&c)                        
	e.Mul2(x1, &b)                         
	negE.Set(&e).Negate(1)                 
	f.Mul2(x2, &b)                         
	x3.Add2(&e, &f).Negate(3).Add(&d)      
	negX3.Set(x3).Negate(5).Normalize()    
	y3.Set(y1).Mul(f.Add(&negE)).Negate(3) 
	y3.Add(e.Add(&negX3).Mul(&c))          
	z3.Mul2(z1, &a)                        

	x3.Normalize()
	y3.Normalize()
}

func (curve *KoblitzCurve) addZ2EqualsOne(x1, y1, z1, x2, y2, x3, y3, z3 *fieldVal) {

	var z1z1, u2, s2 fieldVal
	x1.Normalize()
	y1.Normalize()
	z1z1.SquareVal(z1)                        
	u2.Set(x2).Mul(&z1z1).Normalize()         
	s2.Set(y2).Mul(&z1z1).Mul(z1).Normalize() 
	if x1.Equals(&u2) {
		if y1.Equals(&s2) {

			curve.doubleJacobian(x1, y1, z1, x3, y3, z3)
			return
		}

		x3.SetInt(0)
		y3.SetInt(0)
		z3.SetInt(0)
		return
	}

	var h, hh, i, j, r, rr, v fieldVal
	var negX1, negY1, negX3 fieldVal
	negX1.Set(x1).Negate(1)                
	h.Add2(&u2, &negX1)                    
	hh.SquareVal(&h)                       
	i.Set(&hh).MulInt(4)                   
	j.Mul2(&h, &i)                         
	negY1.Set(y1).Negate(1)                
	r.Set(&s2).Add(&negY1).MulInt(2)       
	rr.SquareVal(&r)                       
	v.Mul2(x1, &i)                         
	x3.Set(&v).MulInt(2).Add(&j).Negate(3) 
	x3.Add(&rr)                            
	negX3.Set(x3).Negate(5)                
	y3.Set(y1).Mul(&j).MulInt(2).Negate(2) 
	y3.Add(v.Add(&negX3).Mul(&r))          
	z3.Add2(z1, &h).Square()               
	z3.Add(z1z1.Add(&hh).Negate(2))        

	x3.Normalize()
	y3.Normalize()
	z3.Normalize()
}

func (curve *KoblitzCurve) addGeneric(x1, y1, z1, x2, y2, z2, x3, y3, z3 *fieldVal) {

	var z1z1, z2z2, u1, u2, s1, s2 fieldVal
	z1z1.SquareVal(z1)                        
	z2z2.SquareVal(z2)                        
	u1.Set(x1).Mul(&z2z2).Normalize()         
	u2.Set(x2).Mul(&z1z1).Normalize()         
	s1.Set(y1).Mul(&z2z2).Mul(z2).Normalize() 
	s2.Set(y2).Mul(&z1z1).Mul(z1).Normalize() 
	if u1.Equals(&u2) {
		if s1.Equals(&s2) {

			curve.doubleJacobian(x1, y1, z1, x3, y3, z3)
			return
		}

		x3.SetInt(0)
		y3.SetInt(0)
		z3.SetInt(0)
		return
	}

	var h, i, j, r, rr, v fieldVal
	var negU1, negS1, negX3 fieldVal
	negU1.Set(&u1).Negate(1)               
	h.Add2(&u2, &negU1)                    
	i.Set(&h).MulInt(2).Square()           
	j.Mul2(&h, &i)                         
	negS1.Set(&s1).Negate(1)               
	r.Set(&s2).Add(&negS1).MulInt(2)       
	rr.SquareVal(&r)                       
	v.Mul2(&u1, &i)                        
	x3.Set(&v).MulInt(2).Add(&j).Negate(3) 
	x3.Add(&rr)                            
	negX3.Set(x3).Negate(5)                
	y3.Mul2(&s1, &j).MulInt(2).Negate(2)   
	y3.Add(v.Add(&negX3).Mul(&r))          
	z3.Add2(z1, z2).Square()               
	z3.Add(z1z1.Add(&z2z2).Negate(2))      
	z3.Mul(&h)                             

	x3.Normalize()
	y3.Normalize()
}

func (curve *KoblitzCurve) addJacobian(x1, y1, z1, x2, y2, z2, x3, y3, z3 *fieldVal) {

	if (x1.IsZero() && y1.IsZero()) || z1.IsZero() {
		x3.Set(x2)
		y3.Set(y2)
		z3.Set(z2)
		return
	}
	if (x2.IsZero() && y2.IsZero()) || z2.IsZero() {
		x3.Set(x1)
		y3.Set(y1)
		z3.Set(z1)
		return
	}

	z1.Normalize()
	z2.Normalize()
	isZ1One := z1.Equals(fieldOne)
	isZ2One := z2.Equals(fieldOne)
	switch {
	case isZ1One && isZ2One:
		curve.addZ1AndZ2EqualsOne(x1, y1, z1, x2, y2, x3, y3, z3)
		return
	case z1.Equals(z2):
		curve.addZ1EqualsZ2(x1, y1, z1, x2, y2, x3, y3, z3)
		return
	case isZ2One:
		curve.addZ2EqualsOne(x1, y1, z1, x2, y2, x3, y3, z3)
		return
	}

	curve.addGeneric(x1, y1, z1, x2, y2, z2, x3, y3, z3)
}

func (curve *KoblitzCurve) Add(x1, y1, x2, y2 *big.Int) (*big.Int, *big.Int) {

	if x1.Sign() == 0 && y1.Sign() == 0 {
		return x2, y2
	}
	if x2.Sign() == 0 && y2.Sign() == 0 {
		return x1, y1
	}

	fx1, fy1 := curve.bigAffineToField(x1, y1)
	fx2, fy2 := curve.bigAffineToField(x2, y2)
	fx3, fy3, fz3 := new(fieldVal), new(fieldVal), new(fieldVal)
	fOne := new(fieldVal).SetInt(1)
	curve.addJacobian(fx1, fy1, fOne, fx2, fy2, fOne, fx3, fy3, fz3)

	return curve.fieldJacobianToBigAffine(fx3, fy3, fz3)
}

func (curve *KoblitzCurve) doubleZ1EqualsOne(x1, y1, x3, y3, z3 *fieldVal) {

	var a, b, c, d, e, f fieldVal
	z3.Set(y1).MulInt(2)                     
	a.SquareVal(x1)                          
	b.SquareVal(y1)                          
	c.SquareVal(&b)                          
	b.Add(x1).Square()                       
	d.Set(&a).Add(&c).Negate(2)              
	d.Add(&b).MulInt(2)                      
	e.Set(&a).MulInt(3)                      
	f.SquareVal(&e)                          
	x3.Set(&d).MulInt(2).Negate(16)          
	x3.Add(&f)                               
	f.Set(x3).Negate(18).Add(&d).Normalize() 
	y3.Set(&c).MulInt(8).Negate(8)           
	y3.Add(f.Mul(&e))                        

	x3.Normalize()
	y3.Normalize()
	z3.Normalize()
}

func (curve *KoblitzCurve) doubleGeneric(x1, y1, z1, x3, y3, z3 *fieldVal) {

	var a, b, c, d, e, f fieldVal
	z3.Mul2(y1, z1).MulInt(2)                
	a.SquareVal(x1)                          
	b.SquareVal(y1)                          
	c.SquareVal(&b)                          
	b.Add(x1).Square()                       
	d.Set(&a).Add(&c).Negate(2)              
	d.Add(&b).MulInt(2)                      
	e.Set(&a).MulInt(3)                      
	f.SquareVal(&e)                          
	x3.Set(&d).MulInt(2).Negate(16)          
	x3.Add(&f)                               
	f.Set(x3).Negate(18).Add(&d).Normalize() 
	y3.Set(&c).MulInt(8).Negate(8)           
	y3.Add(f.Mul(&e))                        

	x3.Normalize()
	y3.Normalize()
	z3.Normalize()
}

func (curve *KoblitzCurve) doubleJacobian(x1, y1, z1, x3, y3, z3 *fieldVal) {

	if y1.IsZero() || z1.IsZero() {
		x3.SetInt(0)
		y3.SetInt(0)
		z3.SetInt(0)
		return
	}

	if z1.Normalize().Equals(fieldOne) {
		curve.doubleZ1EqualsOne(x1, y1, x3, y3, z3)
		return
	}

	curve.doubleGeneric(x1, y1, z1, x3, y3, z3)
}

func (curve *KoblitzCurve) Double(x1, y1 *big.Int) (*big.Int, *big.Int) {
	if y1.Sign() == 0 {
		return new(big.Int), new(big.Int)
	}

	fx1, fy1 := curve.bigAffineToField(x1, y1)
	fx3, fy3, fz3 := new(fieldVal), new(fieldVal), new(fieldVal)
	fOne := new(fieldVal).SetInt(1)
	curve.doubleJacobian(fx1, fy1, fOne, fx3, fy3, fz3)

	return curve.fieldJacobianToBigAffine(fx3, fy3, fz3)
}

func (curve *KoblitzCurve) splitK(k []byte) ([]byte, []byte, int, int) {

	bigIntK := new(big.Int)
	c1, c2 := new(big.Int), new(big.Int)
	tmp1, tmp2 := new(big.Int), new(big.Int)
	k1, k2 := new(big.Int), new(big.Int)

	bigIntK.SetBytes(k)

	c1.Mul(curve.b2, bigIntK)
	c1.Div(c1, curve.N)

	c2.Mul(curve.b1, bigIntK)
	c2.Div(c2, curve.N)

	tmp1.Mul(c1, curve.a1)
	tmp2.Mul(c2, curve.a2)
	k1.Sub(bigIntK, tmp1)
	k1.Add(k1, tmp2)

	tmp1.Mul(c1, curve.b1)
	tmp2.Mul(c2, curve.b2)
	k2.Sub(tmp2, tmp1)

	return k1.Bytes(), k2.Bytes(), k1.Sign(), k2.Sign()
}

func (curve *KoblitzCurve) moduloReduce(k []byte) []byte {

	if len(k) > curve.byteSize {

		tmpK := new(big.Int).SetBytes(k)
		tmpK.Mod(tmpK, curve.N)
		return tmpK.Bytes()
	}

	return k
}

func NAF(k []byte) ([]byte, []byte) {

	var carry, curIsOne, nextIsOne bool

	retPos := make([]byte, len(k)+1)
	retNeg := make([]byte, len(k)+1)
	for i := len(k) - 1; i >= 0; i-- {
		curByte := k[i]
		for j := uint(0); j < 8; j++ {
			curIsOne = curByte&1 == 1
			if j == 7 {
				if i == 0 {
					nextIsOne = false
				} else {
					nextIsOne = k[i-1]&1 == 1
				}
			} else {
				nextIsOne = curByte&2 == 2
			}
			if carry {
				if curIsOne {

				} else {

					if nextIsOne {

						retNeg[i+1] += 1 << j
					} else {

						carry = false
						retPos[i+1] += 1 << j
					}
				}
			} else if curIsOne {
				if nextIsOne {

					retNeg[i+1] += 1 << j
					carry = true
				} else {

					retPos[i+1] += 1 << j
				}
			}
			curByte >>= 1
		}
	}
	if carry {
		retPos[0] = 1
		return retPos, retNeg
	}
	return retPos[1:], retNeg[1:]
}

func (curve *KoblitzCurve) ScalarMult(Bx, By *big.Int, k []byte) (*big.Int, *big.Int) {

	qx, qy, qz := new(fieldVal), new(fieldVal), new(fieldVal)

	k1, k2, signK1, signK2 := curve.splitK(curve.moduloReduce(k))

	p1x, p1y := curve.bigAffineToField(Bx, By)
	p1yNeg := new(fieldVal).NegateVal(p1y, 1)
	p1z := new(fieldVal).SetInt(1)

	p2x := new(fieldVal).Mul2(p1x, curve.beta)
	p2y := new(fieldVal).Set(p1y)
	p2yNeg := new(fieldVal).NegateVal(p2y, 1)
	p2z := new(fieldVal).SetInt(1)

	if signK1 == -1 {
		p1y, p1yNeg = p1yNeg, p1y
	}
	if signK2 == -1 {
		p2y, p2yNeg = p2yNeg, p2y
	}

	k1PosNAF, k1NegNAF := NAF(k1)
	k2PosNAF, k2NegNAF := NAF(k2)
	k1Len := len(k1PosNAF)
	k2Len := len(k2PosNAF)

	m := k1Len
	if m < k2Len {
		m = k2Len
	}

	var k1BytePos, k1ByteNeg, k2BytePos, k2ByteNeg byte
	for i := 0; i < m; i++ {

		if i < m-k1Len {
			k1BytePos = 0
			k1ByteNeg = 0
		} else {
			k1BytePos = k1PosNAF[i-(m-k1Len)]
			k1ByteNeg = k1NegNAF[i-(m-k1Len)]
		}
		if i < m-k2Len {
			k2BytePos = 0
			k2ByteNeg = 0
		} else {
			k2BytePos = k2PosNAF[i-(m-k2Len)]
			k2ByteNeg = k2NegNAF[i-(m-k2Len)]
		}

		for j := 7; j >= 0; j-- {

			curve.doubleJacobian(qx, qy, qz, qx, qy, qz)

			if k1BytePos&0x80 == 0x80 {
				curve.addJacobian(qx, qy, qz, p1x, p1y, p1z,
					qx, qy, qz)
			} else if k1ByteNeg&0x80 == 0x80 {
				curve.addJacobian(qx, qy, qz, p1x, p1yNeg, p1z,
					qx, qy, qz)
			}

			if k2BytePos&0x80 == 0x80 {
				curve.addJacobian(qx, qy, qz, p2x, p2y, p2z,
					qx, qy, qz)
			} else if k2ByteNeg&0x80 == 0x80 {
				curve.addJacobian(qx, qy, qz, p2x, p2yNeg, p2z,
					qx, qy, qz)
			}
			k1BytePos <<= 1
			k1ByteNeg <<= 1
			k2BytePos <<= 1
			k2ByteNeg <<= 1
		}
	}

	return curve.fieldJacobianToBigAffine(qx, qy, qz)
}

func (curve *KoblitzCurve) ScalarBaseMult(k []byte) (*big.Int, *big.Int) {
	newK := curve.moduloReduce(k)
	diff := len(curve.bytePoints) - len(newK)

	qx, qy, qz := new(fieldVal), new(fieldVal), new(fieldVal)

	for i, byteVal := range newK {
		p := curve.bytePoints[diff+i][byteVal]
		curve.addJacobian(qx, qy, qz, &p[0], &p[1], &p[2], qx, qy, qz)
	}
	return curve.fieldJacobianToBigAffine(qx, qy, qz)
}

func (curve *KoblitzCurve) QPlus1Div4() *big.Int {
	return curve.q
}

var initonce sync.Once
var secp256k1 KoblitzCurve

func initAll() {
	initS256()
}

func fromHex(s string) *big.Int {
	r, ok := new(big.Int).SetString(s, 16)
	if !ok {
		panic("invalid hex in source file: " + s)
	}
	return r
}

func initS256() {

	secp256k1.CurveParams = new(elliptic.CurveParams)
	secp256k1.P = fromHex("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEFFFFFC2F")
	secp256k1.N = fromHex("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141")
	secp256k1.B = fromHex("0000000000000000000000000000000000000000000000000000000000000007")
	secp256k1.Gx = fromHex("79BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798")
	secp256k1.Gy = fromHex("483ADA7726A3C4655DA4FBFC0E1108A8FD17B448A68554199C47D08FFB10D4B8")
	secp256k1.BitSize = 256
	secp256k1.q = new(big.Int).Div(new(big.Int).Add(secp256k1.P,
		big.NewInt(1)), big.NewInt(4))
	secp256k1.H = 1
	secp256k1.halfOrder = new(big.Int).Rsh(secp256k1.N, 1)

	secp256k1.byteSize = secp256k1.BitSize / 8

	if err := loadS256BytePoints(); err != nil {
		panic(err)
	}

	secp256k1.lambda = fromHex("5363AD4CC05C30E0A5261C028812645A122E22EA20816678DF02967C1B23BD72")
	secp256k1.beta = new(fieldVal).SetHex("7AE96A2B657C07106E64479EAC3434E99CF0497512F58995C1396C28719501EE")
	secp256k1.a1 = fromHex("3086D221A7D46BCDE86C90E49284EB15")
	secp256k1.b1 = fromHex("-E4437ED6010E88286F547FA90ABFE4C3")
	secp256k1.a2 = fromHex("114CA50F7A8E2F3F657C1108D9D44CFD8")
	secp256k1.b2 = fromHex("3086D221A7D46BCDE86C90E49284EB15")

}

func S256() *KoblitzCurve {
	initonce.Do(initAll)
	return &secp256k1
}
