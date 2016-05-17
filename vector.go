package minimu9

import (
	"bytes"
	"encoding/binary"

	"github.com/dasfoo/i2c"
	"github.com/golang/geo/r3"
)

type intVector struct {
	X, Y, Z int16
}

// ReadStatusAndVector reads status byte, and 3x2-byte X, Y and Z int16 vector values.
func ReadStatusAndVector(bus *i2c.Bus, addr, reg byte) (v *r3.Vector, e error) {
	data := make([]byte, 7)
	// Set MSB for the slave to advance the register on every read.
	if _, e = bus.ReadSliceFromReg(addr, reg|(1<<7), data); e != nil {
		return
	}
	dataReader := bytes.NewReader(data[1:])
	var iv intVector
	if e = binary.Read(dataReader, binary.LittleEndian, &iv); e != nil {
		return
	}
	if data[0]&0xf0 > 0 {
		e = &DataAvailabilityError{NewDataWasOverwritten: true}
	} else if data[0]&0x0f == 0 {
		e = &DataAvailabilityError{NewDataNotAvailable: true}
	}
	v = &r3.Vector{
		X: float64(iv.X),
		Y: float64(iv.Y),
		Z: float64(iv.Z),
	}
	return
}
