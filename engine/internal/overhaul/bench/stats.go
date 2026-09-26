package bench

import (
	"math"
	"sort"
)

// uNull returns P(U = u) for u = 0..n*m under the Mann-Whitney null with arm
// sizes n and m, from the generating function prod_i (1-q^(m+i))/(1-q^i).
// Cancellation leaves the upper half inaccurate; only the lower tail is read.
func uNull(n, m int) []float64 {
	c := make([]float64, n*m+1)
	c[0] = 1
	for i := 1; i <= n; i++ {
		for u := len(c) - 1; u >= m+i; u-- {
			c[u] -= c[u-m-i]
		}
		for u := i; u < len(c); u++ {
			c[u] += c[u-i]
		}
	}
	total := 0.0
	for _, v := range c {
		total += v
	}
	for i := range c {
		c[i] /= total
	}
	return c
}

// exactLimit bounds the arm sizes for the exact null distribution, well inside
// float64 range and cheap to compute; beyond it the (slightly conservative)
// normal approximation takes over.
const exactLimit = 400

// maxPairs caps n*m for the pairwise differences, which are held in memory.
const maxPairs = 4_000_000

// tailCount returns the largest k with P(U <= k-1) <= alpha/2: the pairwise
// differences D_(1..k-1) and D_(nm-k+2..nm) fall outside a (1-alpha) interval,
// whose endpoints are D_(k) and D_(nm+1-k). Zero means no finite interval.
func tailCount(n, m int, alpha float64) int {
	if n+m <= exactLimit {
		p, cum, k := uNull(n, m), 0.0, 0
		for u := range p {
			if cum += p[u]; cum > alpha/2 {
				break
			}
			k = u + 1
		}
		return k
	}
	mean := float64(n*m) / 2
	sd := math.Sqrt(float64(n*m*(n+m+1)) / 12)
	return max(0, int(math.Floor(mean-normalQuantile(1-alpha/2)*sd-0.5))+1)
}

// hlRatio is the Hodges-Lehmann ratio estimate exp(median(ln x_i - ln y_j))
// with its distribution-free (1-alpha) bounds. ok is false when a sample is
// not positive or the arms are too small for a finite interval.
func hlRatio(x, y []float64, alpha float64) (est, lo, hi float64, ok bool) {
	if len(x)*len(y) > maxPairs {
		return 0, 0, 0, false
	}
	z := make([]float64, 0, len(x)*len(y))
	for _, a := range x {
		for _, b := range y {
			if a <= 0 || b <= 0 {
				return 0, 0, 0, false
			}
			z = append(z, math.Log(a)-math.Log(b))
		}
	}
	k := tailCount(len(x), len(y), alpha)
	if k == 0 {
		return 0, 0, 0, false
	}
	sort.Float64s(z)
	return math.Exp(median(z)), math.Exp(z[k-1]), math.Exp(z[len(z)-k]), true
}

// quantileBounds returns one-sided (1-alpha) distribution-free bounds for the
// p-quantile of xs from its order statistics; a missing bound is ±Inf. With
// F(j) = P(Bin(n, p) <= j), the upper bound is x_(k) for the smallest k with
// F(k-1) >= 1-alpha and the lower bound x_(j) for the largest j with
// F(j-1) <= alpha.
func quantileBounds(xs []float64, p, alpha float64) (lcb, ucb float64) {
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	n := len(s)
	lcb, ucb = math.Inf(-1), math.Inf(1)
	lg := func(k int) float64 { v, _ := math.Lgamma(float64(k) + 1); return v }
	cdf := 0.0
	for i := 0; i < n; i++ { // cdf = F(i) after this step
		cdf += math.Exp(lg(n) - lg(i) - lg(n-i) + float64(i)*math.Log(p) + float64(n-i)*math.Log1p(-p))
		if cdf <= alpha {
			lcb = s[i] // j = i+1
		}
		if cdf >= 1-alpha {
			ucb = s[i] // k = i+1
			break
		}
	}
	return lcb, ucb
}

// quantile7 is the linearly interpolated sample quantile (Hyndman-Fan type 7).
func quantile7(xs []float64, p float64) float64 {
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	h := p * float64(len(s)-1)
	i := int(math.Floor(h))
	if i+1 >= len(s) {
		return s[len(s)-1]
	}
	return s[i] + (h-float64(i))*(s[i+1]-s[i])
}

// wilson returns the Wilson score interval for e events in n trials.
func wilson(p float64, n, z float64) (lo, hi float64) {
	d := 1 + z*z/n
	c := (p + z*z/(2*n)) / d
	h := z * math.Sqrt(p*(1-p)/n+z*z/(4*n*n)) / d
	return max(0, c-h), min(1, c+h)
}

// newcombe returns the hybrid score interval (Newcombe 1998, method 10) for
// p1 - p2 from proportions over n1 and n2 trials.
func newcombe(p1, n1, p2, n2, z float64) (lo, hi float64) {
	l1, u1 := wilson(p1, n1, z)
	l2, u2 := wilson(p2, n2, z)
	d := p1 - p2
	return d - math.Sqrt((p1-l1)*(p1-l1)+(u2-p2)*(u2-p2)), d + math.Sqrt((u1-p1)*(u1-p1)+(p2-l2)*(p2-l2))
}

// normalQuantile is the standard normal inverse CDF (Acklam's rational
// approximation, relative error below 1.2e-9).
func normalQuantile(p float64) float64 {
	a := []float64{-3.969683028665376e+01, 2.209460984245205e+02, -2.759285104469687e+02, 1.383577518672690e+02, -3.066479806614716e+01, 2.506628277459239e+00}
	b := []float64{-5.447609879822406e+01, 1.615858368580409e+02, -1.556989798598866e+02, 6.680131188771972e+01, -1.328068155288572e+01}
	c := []float64{-7.784894002430293e-03, -3.223964580411365e-01, -2.400758277161838e+00, -2.549732539343734e+00, 4.374664141464968e+00, 2.938163982698783e+00}
	d := []float64{7.784695709041462e-03, 3.224671290700398e-01, 2.445134137142996e+00, 3.754408661907416e+00}
	switch {
	case p < 0.02425:
		q := math.Sqrt(-2 * math.Log(p))
		return (((((c[0]*q+c[1])*q+c[2])*q+c[3])*q+c[4])*q + c[5]) / ((((d[0]*q+d[1])*q+d[2])*q+d[3])*q + 1)
	case p > 1-0.02425:
		return -normalQuantile(1 - p)
	}
	q := p - 0.5
	r := q * q
	return (((((a[0]*r+a[1])*r+a[2])*r+a[3])*r+a[4])*r + a[5]) * q / (((((b[0]*r+b[1])*r+b[2])*r+b[3])*r+b[4])*r + 1)
}
