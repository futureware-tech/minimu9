package l3gd

import (
	"math"

	"github.com/dasfoo/i2c"
	"github.com/dasfoo/minimu9"
	"github.com/golang/geo/r3"
)

// Gyro is a sensor driver implementation for L3GD20H Gyro.
// Documentation: http://goo.gl/Nb95rx
// Arduino code samples: https://github.com/pololu/l3g-arduino
type Gyro struct {
	bus            i2c.Bus
	address        byte
	fullScaleIndex byte
	frequency      float64
	Offset         r3.Vector
}

// DefaultAddress is a default I2C address for this sensor.
const DefaultAddress = 0x6b

// NewGyro creates new instance bound to I2C bus and address.
func NewGyro(bus i2c.Bus, addr byte) *Gyro {
	return &Gyro{
		bus:            bus,
		address:        addr,
		fullScaleIndex: 0,
		frequency:      12.5,
	}
}

const (
	regCtrl1  = 0x20
	regCtrl4  = 0x23
	regLowOdr = 0x39
)

// Sleep puts the sensor in low power consumption mode.
func (g *Gyro) Sleep() error {
	// We are actually setting it to power-down mode rather than sleep.
	// Power-down consumes less power, but takes longer to wake.
	return g.bus.WriteByteToReg(g.address, regCtrl1, 0x00)
}

// SetFrequency sets gyro output data rate, in Hz. Values: 12.5 .. 800.
func (g *Gyro) SetFrequency(value float64) error {
	g.frequency = value
	frequencyBits := byte(math.Log2(value / 12.5))
	var lowOdr = byte(1)
	if frequencyBits > 2 {
		frequencyBits -= 3
		lowOdr = 0
	}
	if e := minimu9.WriteBitsToReg(g.bus, g.address, regLowOdr, 1, lowOdr); e != nil {
		return e
	}
	return g.bus.WriteByteToReg(g.address, regCtrl1, 0x0F|frequencyBits<<6)
}

var (
	scaleBits  = []float64{245, 500, 2000}
	scaleRatio = []float64{0.00875, 0.0175, 0.07}
)

// SetFullScale sets gyro full scale, which affects sensitivity. Values: 245, 500, 2000 (degrees/s)
func (g *Gyro) SetFullScale(value float64) error {
	g.fullScaleIndex = byte(len(scaleBits) - 1)
	for index, maxScale := range scaleBits {
		if maxScale >= value {
			g.fullScaleIndex = byte(index)
			break
		}
	}
	return minimu9.WriteBitsToReg(g.bus, g.address, regCtrl4,
		(1<<4)|(1<<5), g.fullScaleIndex<<4)
}

// Wake enables sensor if it was put into power-down mode with Sleep().
func (g *Gyro) Wake() error {
	return g.SetFrequency(g.frequency)
}

// Calibrate measures gyro offset until stop channel is written to.
// Gyro offset is then saved to Offset field.
// NOTE: during calibration, the sensor has to be static (not moving).
func (g *Gyro) Calibrate(stop chan int) error {
	ioffset, _, e := minimu9.GetOffsetAndRange(
		func() (minimu9.IntVector, error) { return minimu9.ReadVector(g.bus, g.address, 0x28) },
		stop)
	offset := ioffset.R3().Mul(scaleRatio[g.fullScaleIndex])
	if e == nil {
		g.Offset = offset
	}
	return e
}

// Read reads angular speed data from the sensor, in degrees per second.
// Note: err might be a warning about data "freshness" if it's minimu9.DataAvailabilityError.
func (g *Gyro) Read() (r3.Vector, error) {
	v, e := minimu9.ReadStatusAndVector(g.bus, g.address, 0x27)
	return v.Mul(scaleRatio[g.fullScaleIndex]).Sub(g.Offset), e
}
