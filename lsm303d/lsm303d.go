package lsm303d

import (
	"math"

	"github.com/dasfoo/i2c"
	"github.com/dasfoo/minimu9"
	"github.com/golang/geo/r3"
)

// Accelerometer is a sensor driver implementation for LSM303D accelerometer.
// Documentation: http://goo.gl/qLMgFM
// Arduino code samples: https://github.com/pololu/lsm303-arduino
type Accelerometer struct {
	bus     i2c.Bus
	address byte
}

// Magnetometer is a sensor driver implementation for LSM303D magnetometer.
// Documentation: http://goo.gl/qLMgFM
// Arduino code samples: https://github.com/pololu/lsm303-arduino
type Magnetometer struct {
	bus     i2c.Bus
	address byte
}

// DefaultAddress is a default I2C address for this sensor.
const DefaultAddress = 0x1d

// NewAccelerometer creates a new instance bound to I2C bus and address.
func NewAccelerometer(bus i2c.Bus, addr byte) *Accelerometer {
	return &Accelerometer{
		bus:     bus,
		address: addr,
	}
}

// NewMagnetometer creates a new instance bound to I2C bus and address.
func NewMagnetometer(bus i2c.Bus, addr byte) *Magnetometer {
	return &Magnetometer{
		bus:     bus,
		address: addr,
	}
}

const (
	regCtrl1 = 0x20
	regCtrl2 = 0x21
	regCtrl5 = 0x24
	regCtrl6 = 0x25
	regCtrl7 = 0x26
)

// Sleep puts accelerometer in power-down mode.
func (a *Accelerometer) Sleep() error {
	return a.bus.WriteByteToReg(a.address, regCtrl1, 0x00)
}

// Wake returns accelerometer into operating mode.
func (a *Accelerometer) Wake() error {
	// Experimentally, frequencies above 100Hz are much less consistent / accurate.
	// 50 Hz Output Data Rate, all axes enabled.
	if err := a.bus.WriteByteToReg(a.address, regCtrl1, 0x57); err != nil {
		return err
	}
	// TODO(dotdoom): write to regCtrl7, it controls accel as well.
	// +/- 8g full scale.
	return a.bus.WriteByteToReg(a.address, regCtrl2, 0x18)
}

// Read reads acceleration vector, in g, from the accelerometer.
// Note: err might be a warning about data "freshness" if it's minimu9.DataAvailabilityError.
func (a *Accelerometer) Read() (r3.Vector, error) {
	// FIXME: +-8 is the full scale
	return minimu9.ReadStatusAndVector(a.bus, a.address, 0x27, 8)
}

// Wake returns magnetometer into operating mode.
func (m *Magnetometer) Wake() error {
	// 6.25 Hz Output Data Rate, High resolution mode.
	if err := m.bus.WriteByteToReg(m.address, regCtrl5, 0x64); err != nil {
		return err
	}
	// +/- 4 gauss scale range.
	if err := m.bus.WriteByteToReg(m.address, regCtrl6, 0x20); err != nil {
		return err
	}
	// Low power mode off, continuous mode.
	return m.bus.WriteByteToReg(m.address, regCtrl7, 0x00)
}

// Sleep puts magnetometer in power-down mode.
func (m *Magnetometer) Sleep() error {
	return m.bus.WriteByteToReg(m.address, regCtrl7, 1<<1)
}

func pushMinMax(v minimu9.IntVector, min, max *minimu9.IntVector) {
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

// Calibrate measures magnetometer bias until stop channel is written to.
// Calibration offset is then saved to the magnetometer and the range is returned.
func (m *Magnetometer) Calibrate(stop chan int) (r3.Vector, error) {
	// Reset any previously computed offset.
	if e := minimu9.WriteVector(m.bus, m.address, 0x16, minimu9.IntVector{}); e != nil {
		return r3.Vector{}, e
	}
	min := minimu9.IntVector{X: math.MaxInt16, Y: math.MaxInt16, Z: math.MaxInt16}
	max := minimu9.IntVector{X: math.MinInt16, Y: math.MinInt16, Z: math.MinInt16}
	for {
		select {
		case <-stop:
			avg := minimu9.IntVector{
				X: int16((int(min.X) + int(max.X)) >> 1),
				Y: int16((int(min.Y) + int(max.Y)) >> 1),
				Z: int16((int(min.Z) + int(max.Z)) >> 1),
			}
			// TODO(dotdoom): fetch the real configured scale
			return minimu9.ScaleVector(r3.Vector{
					X: float64(max.X) - float64(min.X),
					Y: float64(max.Y) - float64(min.Y),
					Z: float64(max.Z) - float64(min.Z)}, 4),
				minimu9.WriteVector(m.bus, m.address, 0x16, avg)
		default:
			v, e := minimu9.ReadVector(m.bus, m.address, 0x08)
			if e != nil {
				return r3.Vector{}, e
			}
			pushMinMax(v, &min, &max)
		}
	}
}

// Read reads new data from the magnetometer.
// Note: err might be a warning about data "freshness" if it's minimu9.DataAvailabilityError.
func (m *Magnetometer) Read() (r3.Vector, error) {
	// FIXME: +-4 is the full scale set above
	return minimu9.ReadStatusAndVector(m.bus, m.address, 0x07, 4)
}

// RelativeHeading returns current heading, in radians. May only be used in relative computations.
func RelativeHeading(a *Accelerometer, m *Magnetometer) (float64, error) {
	_ = a
	v, e := m.Read()
	return math.Atan2(v.Y, v.X), e
}
