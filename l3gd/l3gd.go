package l3gd

import (
	"bytes"
	"encoding/binary"

	"github.com/dasfoo/i2c"
	"github.com/dasfoo/minimu9"
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
	// There's not just power control in CTRL1, we need to keep other values.
	var bw byte
	var err error
	if bw, err = l3g.bus.ReadByteFromReg(l3g.address, regCtrl1); err != nil {
		return err
	}
	// We are actually setting it to power-down mode rather than sleep.
	// Power-down consumes less power, but takes longer to wake.
	return l3g.bus.WriteByteToReg(l3g.address, regCtrl1, bw&^(1<<3))
}

// Wake enables sensor if it was put into power-down mode with Sleep().
func (l3g *Gyro) Wake() error {
	var bw byte
	var err error
	if bw, err = l3g.bus.ReadByteFromReg(l3g.address, regCtrl1); err != nil {
		return err
	}
	return l3g.bus.WriteByteToReg(l3g.address, regCtrl1, bw|0xf0)
}

var bitsLowodrDrForFrequency = [...][3]int{
	// There's also "bandwidth" in that register, but experimentally
	// proven useless for I2C. Perhaps only useful for SPI.
	{12, 1, 0x0f},  // 10
	{25, 1, 0x4f},  // 20
	{50, 1, 0x8f},  // 40
	{100, 0, 0x0f}, // 80
	{200, 0, 0x4f}, // 150
	{400, 0, 0x8f}, // 300
	{800, 0, 0xcf}, // 600
}

// SetFrequency sets Output Data Rate, in Hz, range 12 .. 800.
func (l3g *Gyro) SetFrequency(hz int) error {
	// ~250 dps full scale (gain).
	if err := l3g.bus.WriteByteToReg(l3g.address, regCtrl4, 0x00); err != nil {
		return err
	}
	for i := 0; i < len(bitsLowodrDrForFrequency); i++ {
		if bitsLowodrDrForFrequency[i][0] >= hz || i == len(bitsLowodrDrForFrequency)-1 {
			if err := l3g.bus.WriteByteToReg(l3g.address, regLowOdr,
				byte(bitsLowodrDrForFrequency[i][1])); err != nil {
				return err
			}
			return l3g.bus.WriteByteToReg(l3g.address, regCtrl1,
				byte(bitsLowodrDrForFrequency[i][2]))
		}
	}
	// This should never happen.
	return nil
}

// Read reads new data from the sensor.
// Note: err might be a warning about data "freshness" if it's minimu9.DataAvailabilityError.
// Call sequence:
//   SetFrequency(...)
//   in a loop: Read()
func (l3g *Gyro) Read() (v *minimu9.Vector, err error) {
	data := make([]byte, 7)
	if _, err = l3g.bus.ReadSliceFromReg(l3g.address, 0x27|(1<<7), data); err != nil {
		return
	}
	dataReader := bytes.NewReader(data[1:])
	v = &minimu9.Vector{}
	if err = binary.Read(dataReader, binary.LittleEndian, v); err != nil {
		return
	}
	if data[0]&0xf0 > 0 {
		err = &minimu9.DataAvailabilityError{NewDataWasOverwritten: true}
	} else if data[0]&0x0f == 0 {
		err = &minimu9.DataAvailabilityError{NewDataNotAvailable: true}
	}
	return
}
