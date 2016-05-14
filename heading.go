package minimu9

import "github.com/golang/geo/r3"

func Heading(a, m, g r3.Vector) r3.Vector {
	// If acceleration vector is close enough to 1, there's a good chance that the vector is
	// only representing gravity; we can use it to find out pitch / roll. Attenuate accelerometer
	// vector by how much we are close to 1g:
	// * 0.7g = 0
	// * 1g = 1
	// * 1.3g = 0
	/*accelerometerWeight := math.Max(1, math.Min(0,
		1-math.Abs(a.Norm()-1)/0.3,
	))*/
	return r3.Vector{}
}
