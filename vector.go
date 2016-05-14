package minimu9

import (
	"bytes"
	"encoding/binary"
	"math"
	"time"

	"github.com/dasfoo/i2c"
	"github.com/golang/geo/r3"
)

// IntVector is a 3D vector holding the dimensions in 2-byte signed integers,
// as all the hardware parts of miniIMU9 do.
type IntVector struct {
	X, Y, Z int16
}

// R3 converts IntVector into r3.Vector.
func (v *IntVector) R3() r3.Vector {
	return r3.Vector{
		X: float64(v.X),
		Y: float64(v.Y),
		Z: float64(v.Z),
	}
}

// ReadStatusAndVector reads status byte, and 3x2-byte X, Y and Z int16 vector values.
func ReadStatusAndVector(bus i2c.Bus, addr, reg byte) (
	v r3.Vector, e error) {
	var status byte
	if status, e = bus.ReadByteFromReg(addr, reg); e != nil {
		return
	}
	var iv IntVector
	if iv, e = ReadVector(bus, addr, reg+1); e != nil {
		return
	}
	if status&0xf0 > 0 {
		e = &DataAvailabilityError{NewDataWasOverwritten: true}
	} else if status&0x0f == 0 {
		e = &DataAvailabilityError{NewDataNotAvailable: true}
	}
	v = iv.R3()
	return
}

// ReadVector reads IntVector dimensions.
func ReadVector(bus i2c.Bus, addr, reg byte) (v IntVector, e error) {
	data := make([]byte, 6)
	// Set MSB for the slave to advance the register on every read.
	if _, e = bus.ReadSliceFromReg(addr, reg|(1<<7), data); e != nil {
		return
	}
	e = binary.Read(bytes.NewReader(data), binary.LittleEndian, &v)
	return
}

// WriteVector writes IntVector dimensions.
func WriteVector(bus i2c.Bus, addr, reg byte, v IntVector) error {
	var data bytes.Buffer
	if e := binary.Write(&data, binary.LittleEndian, &v); e != nil {
		return e
	}
	// Set MSB for the slave to advance the register on every read.
	_, e := bus.WriteSliceToReg(addr, reg|(1<<7), data.Bytes())
	return e
}

func pushMinMax(v IntVector, min, max *IntVector) {
	if v.X < min.X {
		min.X = v.X
	} else if v.X > max.X {
		max.X = v.X
	}
	if v.Y < min.Y {
		min.Y = v.Y
	} else if v.Y > max.Y {
		max.Y = v.Y
	}
	if v.Z < min.Z {
		min.Z = v.Z
	} else if v.Z > max.Z {
		max.Z = v.Z
	}
}

// GetOffsetAndRange computes average offset and range of vectors returned by read(),
// until stop channel is written to.
func GetOffsetAndRange(read func() (IntVector, error), stop chan int) (
	IntVector, r3.Vector, error) {
	min := IntVector{X: math.MaxInt16, Y: math.MaxInt16, Z: math.MaxInt16}
	max := IntVector{X: math.MinInt16, Y: math.MinInt16, Z: math.MinInt16}
	// TODO(dotdoom): store the values and discard some % of top/bottom values?
	for {
		select {
		case <-stop:
			return IntVector{
					X: int16((int(min.X) + int(max.X)) >> 1),
					Y: int16((int(min.Y) + int(max.Y)) >> 1),
					Z: int16((int(min.Z) + int(max.Z)) >> 1),
				},
				r3.Vector{
					X: float64(max.X) - float64(min.X),
					Y: float64(max.Y) - float64(min.Y),
					Z: float64(max.Z) - float64(min.Z),
				}, nil
		default:
			v, e := read()
			// Only accelerometer is capable of very high frequencies - there's a very little chance
			// we will miss something important during this sleep, but we give others some time.
			// TODO(dotdoom): re-evaluate this - calibration is a once-in-a-long-while thing.
			time.Sleep(time.Millisecond)
			if e != nil {
				return IntVector{}, r3.Vector{}, e
			}
			pushMinMax(v, &min, &max)
		}
	}
}
