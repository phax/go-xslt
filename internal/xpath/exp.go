package xpath

import "math"

// The code below is Go's portable math.Exp (src/math/exp.go), with every
// product wrapped in float64() so that no compiler fuses it into an FMA. The
// standard library's math.Exp is not one function: on amd64 it is assembly
// whose result depends on whether the CPU has FMA (math:exp(math:pi()) is one
// ulp off without it, math-exp-006), and on arm64 the compiler fuses the
// portable code's multiply-adds. This version gives the same bits on every
// platform, which the conformance verdicts must not depend on.
//
// Copyright 2009 The Go Authors. All rights reserved. Use of this source code
// is governed by a BSD-style license that can be found at
// https://go.dev/LICENSE. The original C code, the long comment, and the
// constants are from FreeBSD's /usr/src/lib/msun/src/e_exp.c and came with
// this notice:
//
// ====================================================
// Copyright (C) 2004 by Sun Microsystems, Inc. All rights reserved.
//
// Permission to use, copy, modify, and distribute this
// software is freely granted, provided that this notice
// is preserved.
// ====================================================

// exactExp returns e**x, bit for bit the same on every platform.
func exactExp(x float64) float64 {
	const (
		Ln2Hi = 6.93147180369123816490e-01
		Ln2Lo = 1.90821492927058770002e-10
		Log2e = 1.44269504088896338700e+00

		Overflow  = 7.09782712893383973096e+02
		Underflow = -7.45133219101941108420e+02
		NearZero  = 1.0 / (1 << 28) // 2**-28
	)

	switch {
	case math.IsNaN(x):
		return x
	case x > Overflow: // handles +Inf
		return math.Inf(1)
	case x < Underflow: // handles -Inf
		return 0
	case -NearZero < x && x < NearZero:
		return 1 + x
	}

	// reduce; computed as r = hi - lo for extra precision.
	var k int
	switch {
	case x < 0:
		k = int(float64(Log2e*x) - 0.5)
	case x > 0:
		k = int(float64(Log2e*x) + 0.5)
	}
	hi := x - float64(float64(k)*Ln2Hi)
	lo := float64(float64(k) * Ln2Lo)
	return expmulti(hi, lo, k)
}

// expmulti returns e**r × 2**k where r = hi - lo and |r| ≤ ln(2)/2.
func expmulti(hi, lo float64, k int) float64 {
	const (
		P1 = 1.66666666666666657415e-01  /* 0x3FC55555; 0x55555555 */
		P2 = -2.77777777770155933842e-03 /* 0xBF66C16C; 0x16BEBD93 */
		P3 = 6.61375632143793436117e-05  /* 0x3F11566A; 0xAF25DE2C */
		P4 = -1.65339022054652515390e-06 /* 0xBEBBBD41; 0xC5D26BF1 */
		P5 = 4.13813679705723846039e-08  /* 0x3E663769; 0x72BEA4D0 */
	)

	r := hi - lo
	t := float64(r * r)
	p := P4 + float64(t*P5)
	p = P3 + float64(t*p)
	p = P2 + float64(t*p)
	p = P1 + float64(t*p)
	c := r - float64(t*p)
	y := 1 - ((lo - float64(r*c)/(2-c)) - hi)
	return math.Ldexp(y, k)
}
