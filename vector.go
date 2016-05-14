package minimu9

import "github.com/golang/geo/r3"

// 3D Vector with integer dimensions.
type Vector struct {
	X, Y, Z int16
}

// ToR3 converts the vector to r3.Vector, which has useful methods.
func (v *Vector) ToR3() *r3.Vector {
	return &r3.Vector{
		X: float64(v.X),
		Y: float64(v.Y),
		Z: float64(v.Z),
	}
}
