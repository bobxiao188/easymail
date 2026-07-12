package fasttext

import (
	"math"
)

// fastExp computes exp(x) using math.Exp
func fastExp(x float64) float64 {
	return math.Exp(x)
}
