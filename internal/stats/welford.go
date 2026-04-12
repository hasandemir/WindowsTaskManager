package stats

import "math"

// Welford computes running mean, variance, and standard deviation in O(1) per update.
type Welford struct {
	count uint64
	mean  float64
	m2    float64
}

func NewWelford() *Welford { return &Welford{} }

func (w *Welford) Add(value float64) {
	w.count++
	delta := value - w.mean
	w.mean += delta / float64(w.count)
	delta2 := value - w.mean
	w.m2 += delta * delta2
}

func (w *Welford) Mean() float64 { return w.mean }

func (w *Welford) Variance() float64 {
	if w.count < 2 {
		return 0
	}
	return w.m2 / float64(w.count-1)
}

func (w *Welford) StdDev() float64 { return math.Sqrt(w.Variance()) }

func (w *Welford) Count() uint64 { return w.count }

// IsAnomaly returns true if value lies more than nSigma standard deviations from the mean.
func (w *Welford) IsAnomaly(value float64, nSigma float64) bool {
	if w.count < 10 {
		return false
	}
	return math.Abs(value-w.mean) > nSigma*w.StdDev()
}
