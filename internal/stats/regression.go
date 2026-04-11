package stats

// LinearRegression computes slope, intercept, and coefficient of determination (R²)
// using ordinary least squares.
func LinearRegression(xs, ys []float64) (slope, intercept, rSquared float64) {
	n := float64(len(xs))
	if n < 2 || len(xs) != len(ys) {
		return 0, 0, 0
	}

	var sumX, sumY, sumXY, sumX2 float64
	for i := range xs {
		sumX += xs[i]
		sumY += ys[i]
		sumXY += xs[i] * ys[i]
		sumX2 += xs[i] * xs[i]
	}

	denom := n*sumX2 - sumX*sumX
	if denom == 0 {
		return 0, sumY / n, 0
	}

	slope = (n*sumXY - sumX*sumY) / denom
	intercept = (sumY - slope*sumX) / n

	meanY := sumY / n
	var ssRes, ssTot float64
	for i := range xs {
		predicted := slope*xs[i] + intercept
		ssRes += (ys[i] - predicted) * (ys[i] - predicted)
		ssTot += (ys[i] - meanY) * (ys[i] - meanY)
	}

	if ssTot == 0 {
		rSquared = 1
	} else {
		rSquared = 1 - ssRes/ssTot
	}
	return slope, intercept, rSquared
}
