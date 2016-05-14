package minimu9

import "github.com/dasfoo/i2c"

// WriteBitsToReg reads a byte from a specified address, clears the bits set in mask,
// then sets them to a specified value and writes back.
func WriteBitsToReg(bus i2c.Bus, address, reg, mask, value byte) error {
	var (
		previousValue byte
		e             error
	)
	if previousValue, e = bus.ReadByteFromReg(address, reg); e != nil {
		return e
	}
	return bus.WriteByteToReg(address, reg, (previousValue&^mask)|(value&mask))
}
