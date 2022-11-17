package trie_kzg_bn256

import "go.dedis.ch/kyber/v3"

// commit commits to vector vect[0], ...., vect[D-1]
// it is [f(s)]1 where f is polynomial  in evaluation (Lagrange) form,
// i.e. with f(rou[i]) = vect[i], i = 0..D-1
// vect[k] == nil equivalent to 0
func (sd *TrustedSetup) commit(vect []kyber.Scalar) kyber.Point {
	ret := sd.Suite.G1().Point().Null()
	elem := sd.Suite.G1().Point()
	for i, e := range vect {
		if e == nil {
			continue
		}
		elem.Mul(e, sd.LagrangeBasis[i])
		ret.Add(ret, elem)
	}
	return ret
}

// prove returns pi = [(f(s)-vect<index>)/(s-rou<index>)]1
// This is the proof sent to verifier
func (sd *TrustedSetup) prove(vect []kyber.Scalar, i int) kyber.Point {
	ret := sd.Suite.G1().Point().Null()
	e := sd.Suite.G1().Point()
	qij := sd.Suite.G1().Scalar()
	for j := range sd.Domain {
		sd.qPoly(vect, i, j, vect[i], qij)
		e.Mul(qij, sd.LagrangeBasis[j])
		ret.Add(ret, e)
	}
	return ret
}

func (sd *TrustedSetup) qPoly(vect []kyber.Scalar, i, m int, y kyber.Scalar, ret kyber.Scalar) {
	numer := sd.Suite.G1().Scalar()
	if i != m {
		sd.diff(vect[m], y, numer)
		if numer.Equal(sd.ZeroG1) {
			ret.Zero()
			return
		}
		ret.Mul(numer, sd.invsub(m, i))
		return
	}
	// i == m
	ret.Zero()
	t := sd.Suite.G1().Scalar()
	t1 := sd.Suite.G1().Scalar()

	for j := range vect {
		if j == m || vect[j] == nil {
			continue
		}
		t.Mul(vect[j], sd.ta(m, j, t1))
		ret.Add(ret, t)
	}
	if vect[m] != nil {
		t.Mul(vect[m], sd.tk(m, t1))
		ret.Sub(ret, t)
	}
}

func (sd *TrustedSetup) diff(vi, vj kyber.Scalar, ret kyber.Scalar) {
	switch {
	case vi == nil && vj == nil:
		ret.Zero()
		return
	case vi != nil && vj == nil:
		ret.Set(vi)
	case vi == nil && vj != nil:
		ret.Neg(vj)
	default:
		ret.Sub(vi, vj)
	}
}

// verify verifies KZG proof that polynomial f committed with C has f(rou<atIndex>) = v
// c is commitment to the polynomial
// pi is commitment to the value point (proof)
// value is the value of the polynomial
// adIndex is index of the root of unity where polynomial is expected to have value = v
func (sd *TrustedSetup) verify(c, pi kyber.Point, v kyber.Scalar, atIndex int) bool {
	p1 := sd.Suite.Pair(pi, sd.Diff2[atIndex])
	e := sd.Suite.G1().Point().Mul(v, nil)
	e.Sub(c, e)
	p2 := sd.Suite.Pair(e, sd.Suite.G2().Point().Base())
	return p1.Equal(p2)
}

// verifyVector calculates proofs and verifies all elements in the vector against commitment C
func (sd *TrustedSetup) verifyVector(vect []kyber.Scalar, c kyber.Point) bool {
	pi := make([]kyber.Point, sd.D)
	for i := range vect {
		pi[i] = sd.prove(vect, i)
	}
	for i := range pi {
		v := vect[i]
		if v == nil {
			v = sd.ZeroG1
		}
		if !sd.verify(c, pi[i], v, i) {
			return false
		}
	}
	return true
}

// commitAll return commit to the whole vector and to each of values of it
// Generate commitment to the vector and proofs to all values.
// Expensive. Usually used only in tests
func (sd *TrustedSetup) commitAll(vect []kyber.Scalar) (kyber.Point, []kyber.Point) { // nolint:unused	// TODO: use function or delete it
	retC := sd.commit(vect)
	retPi := make([]kyber.Point, sd.D)
	for i := range vect {
		if vect[i] == nil {
			continue
		}
		retPi[i] = sd.prove(vect, i)
	}
	return retC, retPi
}
