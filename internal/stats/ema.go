package stats

// EMA is an exponential moving average smoother.
type EMA struct {
	alpha  float64
	value  float64
	primed bool
}

func NewEMA(alpha float64) *EMA {
	if alpha <= 0 {
		alpha = 0.1
	}
	if alpha > 1 {
		alpha = 1
	}
	return &EMA{alpha: alpha}
}

func (e *EMA) Add(value float64) float64 {
	if !e.primed {
		e.value = value
		e.primed = true
		return e.value
	}
	e.value = e.alpha*value + (1-e.alpha)*e.value
	return e.value
}

func (e *EMA) Value() float64 { return e.value }

func (e *EMA) Reset() {
	e.value = 0
	e.primed = false
}
