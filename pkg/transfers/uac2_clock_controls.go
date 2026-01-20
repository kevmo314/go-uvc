package transfers

import (
	"fmt"
	"time"

	usb "github.com/kevmo314/go-usb"
)

// UAC2 Clock Source Control Selectors
const (
	CS_CONTROL_UNDEFINED   = 0x00
	CS_SAM_FREQ_CONTROL    = 0x01
	CS_CLOCK_VALID_CONTROL = 0x02
)

// UAC2 Clock Selector Control Selectors
const (
	CX_CONTROL_UNDEFINED = 0x00
	CX_CLOCK_SELECTOR    = 0x01
)

// UAC2 Clock Multiplier Control Selectors
const (
	CM_CONTROL_UNDEFINED   = 0x00
	CM_NUMERATOR_CONTROL   = 0x01
	CM_DENOMINATOR_CONTROL = 0x02
)

// UAC2ClockControl provides clock domain control for UAC2/3 devices
type UAC2ClockControl struct {
	handle *usb.DeviceHandle
	ifnum  uint8
}

func NewUAC2ClockControl(handle *usb.DeviceHandle, interfaceNumber uint8) *UAC2ClockControl {
	return &UAC2ClockControl{
		handle: handle,
		ifnum:  interfaceNumber,
	}
}

// Clock Source Controls

// SetClockFrequency sets the sampling frequency for a clock source
func (c *UAC2ClockControl) SetClockFrequency(clockID uint8, freq uint32) error {
	data := []byte{
		byte(freq & 0xFF),
		byte((freq >> 8) & 0xFF),
		byte((freq >> 16) & 0xFF),
		byte((freq >> 24) & 0xFF),
	}

	return c.controlTransfer(
		0x21,
		SET_CUR,
		CS_SAM_FREQ_CONTROL,
		clockID,
		data,
	)
}

// GetClockFrequency gets the current sampling frequency from a clock source
func (c *UAC2ClockControl) GetClockFrequency(clockID uint8) (uint32, error) {
	data := make([]byte, 4)

	err := c.controlTransfer(
		0xA1,
		GET_CUR,
		CS_SAM_FREQ_CONTROL,
		clockID,
		data,
	)

	freq := uint32(data[0]) | (uint32(data[1]) << 8) |
		(uint32(data[2]) << 16) | (uint32(data[3]) << 24)
	return freq, err
}

// GetClockFrequencyRange gets the min, max, and resolution for clock frequency
func (c *UAC2ClockControl) GetClockFrequencyRange(clockID uint8) (min, max, res uint32, err error) {
	data := make([]byte, 4)

	// Get MIN
	err = c.controlTransfer(0xA1, GET_MIN, CS_SAM_FREQ_CONTROL, clockID, data)
	if err != nil {
		return
	}
	min = uint32(data[0]) | (uint32(data[1]) << 8) |
		(uint32(data[2]) << 16) | (uint32(data[3]) << 24)

	// Get MAX
	err = c.controlTransfer(0xA1, GET_MAX, CS_SAM_FREQ_CONTROL, clockID, data)
	if err != nil {
		return
	}
	max = uint32(data[0]) | (uint32(data[1]) << 8) |
		(uint32(data[2]) << 16) | (uint32(data[3]) << 24)

	// Get RES
	err = c.controlTransfer(0xA1, GET_RES, CS_SAM_FREQ_CONTROL, clockID, data)
	if err != nil {
		return
	}
	res = uint32(data[0]) | (uint32(data[1]) << 8) |
		(uint32(data[2]) << 16) | (uint32(data[3]) << 24)

	return
}

// IsClockValid checks if a clock source is valid/locked
func (c *UAC2ClockControl) IsClockValid(clockID uint8) (bool, error) {
	data := make([]byte, 1)

	err := c.controlTransfer(
		0xA1,
		GET_CUR,
		CS_CLOCK_VALID_CONTROL,
		clockID,
		data,
	)

	return data[0] != 0, err
}

// Clock Selector Controls

// SetClockSelector selects which clock source to use
func (c *UAC2ClockControl) SetClockSelector(selectorID uint8, clockSourceID uint8) error {
	data := []byte{clockSourceID}

	return c.controlTransfer(
		0x21,
		SET_CUR,
		CX_CLOCK_SELECTOR,
		selectorID,
		data,
	)
}

// GetClockSelector gets the currently selected clock source
func (c *UAC2ClockControl) GetClockSelector(selectorID uint8) (uint8, error) {
	data := make([]byte, 1)

	err := c.controlTransfer(
		0xA1,
		GET_CUR,
		CX_CLOCK_SELECTOR,
		selectorID,
		data,
	)

	return data[0], err
}

// Clock Multiplier Controls

// SetClockMultiplierNumerator sets the numerator for clock multiplication
func (c *UAC2ClockControl) SetClockMultiplierNumerator(multiplierID uint8, numerator uint16) error {
	data := []byte{
		byte(numerator & 0xFF),
		byte((numerator >> 8) & 0xFF),
	}

	return c.controlTransfer(
		0x21,
		SET_CUR,
		CM_NUMERATOR_CONTROL,
		multiplierID,
		data,
	)
}

// GetClockMultiplierNumerator gets the current numerator
func (c *UAC2ClockControl) GetClockMultiplierNumerator(multiplierID uint8) (uint16, error) {
	data := make([]byte, 2)

	err := c.controlTransfer(
		0xA1,
		GET_CUR,
		CM_NUMERATOR_CONTROL,
		multiplierID,
		data,
	)

	num := uint16(data[0]) | (uint16(data[1]) << 8)
	return num, err
}

// SetClockMultiplierDenominator sets the denominator for clock multiplication
func (c *UAC2ClockControl) SetClockMultiplierDenominator(multiplierID uint8, denominator uint16) error {
	data := []byte{
		byte(denominator & 0xFF),
		byte((denominator >> 8) & 0xFF),
	}

	return c.controlTransfer(
		0x21,
		SET_CUR,
		CM_DENOMINATOR_CONTROL,
		multiplierID,
		data,
	)
}

// GetClockMultiplierDenominator gets the current denominator
func (c *UAC2ClockControl) GetClockMultiplierDenominator(multiplierID uint8) (uint16, error) {
	data := make([]byte, 2)

	err := c.controlTransfer(
		0xA1,
		GET_CUR,
		CM_DENOMINATOR_CONTROL,
		multiplierID,
		data,
	)

	denom := uint16(data[0]) | (uint16(data[1]) << 8)
	return denom, err
}

// Helper function for control transfers
func (c *UAC2ClockControl) controlTransfer(bmRequestType uint8, bRequest uint8,
	controlSelector uint8, clockID uint8, data []byte) error {

	// wValue: Control Selector in high byte, always 0 in low byte for clock entities
	wValue := uint16(controlSelector) << 8

	// wIndex: Clock Entity ID in high byte, Interface in low byte
	wIndex := (uint16(clockID) << 8) | uint16(c.ifnum)

	_, err := c.handle.ControlTransfer(
		bmRequestType,
		bRequest,
		wValue,
		wIndex,
		data,
		time.Second,
	)

	if err != nil {
		return fmt.Errorf("clock control transfer failed: %w", err)
	}

	return nil
}
