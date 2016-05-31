package minimu9

import (
	"bytes"
	"encoding/binary"

	"github.com/dasfoo/i2c"
	"github.com/golang/geo/r3"
)

type IntVector struct {
	X, Y, Z int16
}

// ReadStatusAndVector reads status byte, and 3x2-byte X, Y and Z int16 vector values.
func ReadStatusAndVector(bus i2c.Bus, addr, reg byte) (v r3.Vector, e error) {
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
	v = r3.Vector{
		X: float64(iv.X),
		Y: float64(iv.Y),
		Z: float64(iv.Z),
	}
	return
}

// ReadVector reads 3x2-byte X, Y and Z int16 vector values.
func ReadVector(bus i2c.Bus, addr, reg byte) (v IntVector, e error) {
	data := make([]byte, 6)
	// Set MSB for the slave to advance the register on every read.
	if _, e = bus.ReadSliceFromReg(addr, reg|(1<<7), data); e != nil {
		return
	}
	dataReader := bytes.NewReader(data)
	e = binary.Read(dataReader, binary.LittleEndian, &v)
	return
}

// WriteVector writes 3x2-byte X, Y and Z int16 vector values.
func WriteVector(bus i2c.Bus, addr, reg byte, v IntVector) error {
	var data bytes.Buffer
	if e := binary.Write(&data, binary.LittleEndian, &v); e != nil {
		return e
	}
	// Set MSB for the slave to advance the register on every read.
	_, e := bus.WriteSliceToReg(addr, reg|(1<<7), data.Bytes())
	return e
}
