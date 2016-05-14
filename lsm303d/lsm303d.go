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
	bus            i2c.Bus
	address        byte
	fullScaleIndex byte
	frequency      float64
}

// Magnetometer is a sensor driver implementation for LSM303D magnetometer.
// Documentation: http://goo.gl/qLMgFM
// Arduino code samples: https://github.com/pololu/lsm303-arduino
type Magnetometer struct {
	bus            i2c.Bus
	address        byte
	fullScaleIndex byte
	Offset         r3.Vector
}

// DefaultAddress is a default I2C address for this sensor.
const DefaultAddress = 0x1d

// NewAccelerometer creates a new instance bound to I2C bus and address.
func NewAccelerometer(bus i2c.Bus, addr byte) *Accelerometer {
	return &Accelerometer{
		bus:            bus,
		address:        addr,
		fullScaleIndex: 0,
		frequency:      400,
	}
}

// NewMagnetometer creates a new instance bound to I2C bus and address.
func NewMagnetometer(bus i2c.Bus, addr byte) *Magnetometer {
	return &Magnetometer{
		bus:            bus,
		address:        addr,
		fullScaleIndex: 1,
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
	return minimu9.WriteBitsToReg(a.bus, a.address, regCtrl1, 0xF0, 0x00)
}

// Wake returns accelerometer into operating mode.
func (a *Accelerometer) Wake() error {
	return a.SetFrequency(a.frequency)
}

var (
	magnetometerScaleBits  = []float64{2, 4, 8, 12}
	magnetometerScaleRatio = []float64{0.000080, 0.000160, 0.000320, 0.000479}
)

// SetFullScale sets magnetometer scale that affects sensitivity. Values: 2, 4, 8, 12 (gauss).
func (m *Magnetometer) SetFullScale(value float64) error {
	m.fullScaleIndex = byte(len(magnetometerScaleBits) - 1)
	for index, maxScale := range magnetometerScaleBits {
		if maxScale >= value {
			m.fullScaleIndex = byte(index)
			break
		}
	}
	return minimu9.WriteBitsToReg(m.bus, m.address, regCtrl6,
		(1<<5)|(1<<6), m.fullScaleIndex<<5)
}

var magnetometerFrequencies = []byte{3, 6, 12, 25, 50, 100}

// SetFrequency sets magnetometer output data rate, in Hz. Values: 3, 6, 12, 25, 50, 100.
func (m *Magnetometer) SetFrequency(value byte) error {
	frequencyIndex := byte(len(magnetometerFrequencies) - 1)
	for index, maxFrequency := range magnetometerFrequencies {
		if maxFrequency >= value {
			frequencyIndex = byte(index)
			break
		}
	}
	// Set requested frequency and highest resolution.
	return m.bus.WriteByteToReg(m.address, regCtrl5, (frequencyIndex<<2)|(1<<5)|(1<<6))
}

var (
	accelerometerScaleBits  = []float64{2, 4, 6, 8, 16}
	accelerometerScaleRatio = []float64{0.000061, 0.000122, 0.000183, 0.000244, 0.000732}
)

// Read reads acceleration vector, in g, from the accelerometer.
// Note: err might be a warning about data "freshness" if it's minimu9.DataAvailabilityError.
func (a *Accelerometer) Read() (r3.Vector, error) {
	v, e := minimu9.ReadStatusAndVector(a.bus, a.address, 0x27)
	return v.Mul(accelerometerScaleRatio[a.fullScaleIndex]), e
}

// SetFullScale sets accelerometer scale, which affects sensitivity. Values: 2, 4, 6, 8, 16 (g)
func (a *Accelerometer) SetFullScale(value float64) error {
	a.fullScaleIndex = byte(len(accelerometerScaleBits) - 1)
	for index, maxScale := range accelerometerScaleBits {
		if maxScale >= value {
			a.fullScaleIndex = byte(index)
			break
		}
	}
	return minimu9.WriteBitsToReg(a.bus, a.address, regCtrl2,
		(1<<3)|(1<<4)|(1<<5), a.fullScaleIndex<<3)
}

// SetFrequency sets accelerometer output data rate, in Hz. Values: 3.125 .. 1600.
func (a *Accelerometer) SetFrequency(value float64) error {
	a.frequency = value
	return minimu9.WriteBitsToReg(a.bus, a.address, regCtrl1,
		0xF0, byte(math.Max(math.Log2(value/3.125)+1, 1))<<4)
}

var accelerometerAntiAliasBandwidthBits = [][]uint16{
	{50, 3},
	{194, 1},
	{362, 2},
	{773, 0},
}

// SetAntiAliasBandwidth sets accelerometer's antialias filter bandwidth, in Hz.
// Supported values: 50, 194, 362, 773.
func (a *Accelerometer) SetAntiAliasBandwidth(value uint16) error {
	for index, maxBandwidthBits := range accelerometerAntiAliasBandwidthBits {
		if maxBandwidthBits[0] >= value || index == len(accelerometerAntiAliasBandwidthBits)-1 {
			return minimu9.WriteBitsToReg(a.bus, a.address, regCtrl2,
				(1<<6)|(1<<7), byte(maxBandwidthBits[1])<<6)
		}
	}
	// This should not happen.
	return nil
}

// Wake returns magnetometer into operating mode.
func (m *Magnetometer) Wake() error {
	return minimu9.WriteBitsToReg(m.bus, m.address, regCtrl7, 0x03, 0x00)
}

// Sleep puts magnetometer in power-down mode.
func (m *Magnetometer) Sleep() error {
	return minimu9.WriteBitsToReg(m.bus, m.address, regCtrl7, 0x03, 0x03)
}

// Calibrate measures magnetometer bias until stop channel is written to.
// Calibration offset is then saved to the Offset field and the range is returned.
// NOTE: during calibration, rotate the sensor through all 3 axes.
func (m *Magnetometer) Calibrate(stop chan int) (r3.Vector, error) {
	ioffset, vrange, e := minimu9.GetOffsetAndRange(
		func() (minimu9.IntVector, error) { return minimu9.ReadVector(m.bus, m.address, 0x08) },
		stop)
	offset := ioffset.R3().Mul(magnetometerScaleRatio[m.fullScaleIndex])
	if e == nil {
		m.Offset = offset
	}
	return vrange.Mul(magnetometerScaleRatio[m.fullScaleIndex]), e
}

// Read reads new data from the magnetometer.
// Note: err might be a warning about data "freshness" if it's minimu9.DataAvailabilityError.
func (m *Magnetometer) Read() (r3.Vector, error) {
	v, e := minimu9.ReadStatusAndVector(m.bus, m.address, 0x07)
	return v.Mul(magnetometerScaleRatio[m.fullScaleIndex]).Sub(m.Offset), e
}

// RelativeHeading returns current heading, in radians. May only be used in relative computations.
func RelativeHeading(a *Accelerometer, m *Magnetometer) (float64, error) {
	_ = a
	v, e := m.Read()
	return math.Atan2(v.Y, v.X), e
}
