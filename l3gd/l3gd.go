package l3gd

import (
	"github.com/dasfoo/i2c"
	"github.com/dasfoo/minimu9"
	"github.com/golang/geo/r3"
)

// Gyro is a sensor driver implementation for L3GD20H Gyro.
// Documentation: http://goo.gl/Nb95rx
// Arduino code samples: https://github.com/pololu/l3g-arduino
type Gyro struct {
	bus     *i2c.Bus
	address byte
}

// DefaultAddress is a default I2C address for this sensor.
const DefaultAddress = 0x6b

// NewGyro creates new instance bound to I2C bus and address.
func NewGyro(bus *i2c.Bus, addr byte) *Gyro {
	return &Gyro{
		bus:     bus,
		address: addr,
	}
}

const (
	regCtrl1  = 0x20
	regCtrl4  = 0x23
	regLowOdr = 0x39
)

// Sleep puts the sensor in low power consumption mode.
func (l3g *Gyro) Sleep() error {
	// We are actually setting it to power-down mode rather than sleep.
	// Power-down consumes less power, but takes longer to wake.
	return l3g.bus.WriteByteToReg(l3g.address, regCtrl1, 0x00)
}

// Wake enables sensor if it was put into power-down mode with Sleep().
func (l3g *Gyro) Wake() error {
	// 200 Hz Output Data Rate, 50 Hz Bandwidth, normal mode with all 3 axes enabled.
	if err := l3g.bus.WriteByteToReg(l3g.address, regCtrl1, 0x6F); err != nil {
		return err
	}
	// +/- 250 dps (degrees per second) scale range.
	if err := l3g.bus.WriteByteToReg(l3g.address, regCtrl4, 0x00); err != nil {
		return err
	}
	// Disable Low Output Data Rate.
	return l3g.bus.WriteByteToReg(l3g.address, regLowOdr, 0x00)
}

// Read reads new data from the sensor.
// Note: err might be a warning about data "freshness" if it's minimu9.DataAvailabilityError.
func (l3g *Gyro) Read() (v *r3.Vector, err error) {
	return minimu9.ReadStatusAndVector(l3g.bus, l3g.address, 0x27)
}
