package lsm303d

import (
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
	// +/- 2g full scale.
	return a.bus.WriteByteToReg(a.address, regCtrl2, 0x00)
}

// Read reads new data from the accelerometer.
// Note: err might be a warning about data "freshness" if it's minimu9.DataAvailabilityError.
func (a *Accelerometer) Read() (*r3.Vector, error) {
	return minimu9.ReadStatusAndVector(a.bus, a.address, 0x27)
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

// Read reads new data from the magnetometer.
// Note: err might be a warning about data "freshness" if it's minimu9.DataAvailabilityError.
func (m *Magnetometer) Read() (*r3.Vector, error) {
	return minimu9.ReadStatusAndVector(m.bus, m.address, 0x07)
}
