package reader

import (
	"fmt"
	"math"
)

func Strategy1(str, a int) int {
	if str <= a {
		return -1
	}
	for next := str + 1; ; next++ {
		if isPrime(next) {
			return next
		}
	}
}

func Strategy2(str, a int) int {
	if str%a == 0 {
		return 1
	}
	return 0
}

func Strategy3(str, a int) int {
	if str <= a {
		return -1
	}
	f0, f1 := 0, 1
	for f1 <= str {
		f0, f1 = f1, f0+f1
	}
	return f1
}

func Strategy4(str, _ int) (int, string) {
	if x, y, ok := pythagoreanLegs(int64(str)); ok {
		return 1, fmt.Sprintf("%d² + %d² = %d²", x, y, str)
	}
	return 0, fmt.Sprintf("%d is not a hypotenuse", str)
}

func isPrime(n int) bool {
	if n < 2 {
		return false
	}
	for d := 2; d*d <= n; d++ {
		if n%d == 0 {
			return false
		}
	}
	return true
}

func pythagoreanLegs(hyp int64) (x, y int64, ok bool) {
	c2 := hyp * hyp
	for x = 1; 2*x*x < c2; x++ {
		rem := c2 - x*x
		y = int64(math.Sqrt(float64(rem)))
		if y*y == rem {
			return x, y, true
		}
	}
	return 0, 0, false
}