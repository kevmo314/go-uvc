package transfers

/*
#cgo LDFLAGS: -lusb-1.0
#include <libusb-1.0/libusb.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// UAC Request Codes
const (
	REQUEST_CODE_UNDEFINED = 0x00
	SET_CUR                = 0x01
	GET_CUR                = 0x81
	SET_MIN                = 0x02
	GET_MIN                = 0x82
	SET_MAX                = 0x03
	GET_MAX                = 0x83
	SET_RES                = 0x04
	GET_RES                = 0x84
	SET_MEM                = 0x05
	GET_MEM                = 0x85
	GET_STAT               = 0xFF
)

// Feature Unit Control Selectors
const (
	FU_CONTROL_UNDEFINED  = 0x00
	FU_MUTE_CONTROL       = 0x01
	FU_VOLUME_CONTROL     = 0x02
	FU_BASS_CONTROL       = 0x03
	FU_MID_CONTROL        = 0x04
	FU_TREBLE_CONTROL     = 0x05
	FU_GRAPHIC_EQ_CONTROL = 0x06
	FU_AGC_CONTROL        = 0x07
	FU_DELAY_CONTROL      = 0x08
	FU_BASS_BOOST_CONTROL = 0x09
	FU_LOUDNESS_CONTROL   = 0x0A
)

// Terminal Control Selectors
const (
	TE_CONTROL_UNDEFINED    = 0x00
	TE_COPY_PROTECT_CONTROL = 0x01
)

// Endpoint Control Selectors
const (
	EP_CONTROL_UNDEFINED     = 0x00
	EP_SAMPLING_FREQ_CONTROL = 0x01
	EP_PITCH_CONTROL         = 0x02
)

// Processing Unit Control Selectors
const (
	PU_CONTROL_UNDEFINED   = 0x00
	PU_ENABLE_CONTROL      = 0x01
	PU_MODE_SELECT_CONTROL = 0x02
	// Specific to each processing unit type
	PU_REVERB_LEVEL_CONTROL      = 0x03
	PU_REVERB_TIME_CONTROL       = 0x04
	PU_REVERB_FEEDBACK_CONTROL   = 0x05
	PU_CHORUS_LEVEL_CONTROL      = 0x03
	PU_CHORUS_RATE_CONTROL       = 0x04
	PU_CHORUS_DEPTH_CONTROL      = 0x05
	PU_COMPRESSION_RATIO_CONTROL = 0x03
	PU_MAX_AMPL_CONTROL          = 0x04
	PU_THRESHOLD_CONTROL         = 0x05
	PU_ATTACK_TIME_CONTROL       = 0x06
	PU_RELEASE_TIME_CONTROL      = 0x07
)

// Selector Unit Control
const (
	SU_CONTROL_UNDEFINED = 0x00
	SU_SELECTOR_CONTROL  = 0x01
)

// UACControl provides methods to control audio features
type UACControl struct {
	handle *C.struct_libusb_device_handle
	ifnum  uint8
}

func NewUACControl(handle unsafe.Pointer, interfaceNumber uint8) *UACControl {
	return &UACControl{
		handle: (*C.struct_libusb_device_handle)(handle),
		ifnum:  interfaceNumber,
	}
}

// Feature Unit Controls

// SetMute sets the mute state for a feature unit
func (c *UACControl) SetMute(unitID uint8, channelNum uint8, mute bool) error {
	data := []byte{0x00}
	if mute {
		data[0] = 0x01
	}

	return c.controlTransfer(
		0x21, // bmRequestType: Class, Interface, Host to Device
		SET_CUR,
		FU_MUTE_CONTROL,
		unitID,
		channelNum,
		data,
	)
}

// GetMute gets the mute state for a feature unit
func (c *UACControl) GetMute(unitID uint8, channelNum uint8) (bool, error) {
	data := make([]byte, 1)

	err := c.controlTransfer(
		0xA1, // bmRequestType: Class, Interface, Device to Host
		GET_CUR,
		FU_MUTE_CONTROL,
		unitID,
		channelNum,
		data,
	)

	return data[0] != 0, err
}

// SetVolume sets the volume for a feature unit (in dB * 256)
func (c *UACControl) SetVolume(unitID uint8, channelNum uint8, volumeDB int16) error {
	data := []byte{
		byte(volumeDB & 0xFF),
		byte((volumeDB >> 8) & 0xFF),
	}

	return c.controlTransfer(
		0x21,
		SET_CUR,
		FU_VOLUME_CONTROL,
		unitID,
		channelNum,
		data,
	)
}

// GetVolume gets the current volume (in dB * 256)
func (c *UACControl) GetVolume(unitID uint8, channelNum uint8) (int16, error) {
	data := make([]byte, 2)

	err := c.controlTransfer(
		0xA1,
		GET_CUR,
		FU_VOLUME_CONTROL,
		unitID,
		channelNum,
		data,
	)

	volume := int16(data[0]) | (int16(data[1]) << 8)
	return volume, err
}

// GetVolumeRange gets the min, max, and resolution for volume control
func (c *UACControl) GetVolumeRange(unitID uint8, channelNum uint8) (min, max, res int16, err error) {
	data := make([]byte, 2)

	// Get MIN
	err = c.controlTransfer(0xA1, GET_MIN, FU_VOLUME_CONTROL, unitID, channelNum, data)
	if err != nil {
		return
	}
	min = int16(data[0]) | (int16(data[1]) << 8)

	// Get MAX
	err = c.controlTransfer(0xA1, GET_MAX, FU_VOLUME_CONTROL, unitID, channelNum, data)
	if err != nil {
		return
	}
	max = int16(data[0]) | (int16(data[1]) << 8)

	// Get RES
	err = c.controlTransfer(0xA1, GET_RES, FU_VOLUME_CONTROL, unitID, channelNum, data)
	if err != nil {
		return
	}
	res = int16(data[0]) | (int16(data[1]) << 8)

	return
}

// SetBass sets the bass level (in dB * 256)
func (c *UACControl) SetBass(unitID uint8, channelNum uint8, bassDB int8) error {
	data := []byte{byte(bassDB)}

	return c.controlTransfer(
		0x21,
		SET_CUR,
		FU_BASS_CONTROL,
		unitID,
		channelNum,
		data,
	)
}

// SetTreble sets the treble level (in dB * 256)
func (c *UACControl) SetTreble(unitID uint8, channelNum uint8, trebleDB int8) error {
	data := []byte{byte(trebleDB)}

	return c.controlTransfer(
		0x21,
		SET_CUR,
		FU_TREBLE_CONTROL,
		unitID,
		channelNum,
		data,
	)
}

// SetAGC sets automatic gain control on/off
func (c *UACControl) SetAGC(unitID uint8, channelNum uint8, enable bool) error {
	data := []byte{0x00}
	if enable {
		data[0] = 0x01
	}

	return c.controlTransfer(
		0x21,
		SET_CUR,
		FU_AGC_CONTROL,
		unitID,
		channelNum,
		data,
	)
}

// Endpoint Controls

// SetSamplingFrequency sets the sampling frequency for an endpoint
func (c *UACControl) SetSamplingFrequency(endpoint uint8, freq uint32) error {
	data := []byte{
		byte(freq & 0xFF),
		byte((freq >> 8) & 0xFF),
		byte((freq >> 16) & 0xFF),
	}

	return c.controlTransfer(
		0x22, // bmRequestType: Class, Endpoint, Host to Device
		SET_CUR,
		EP_SAMPLING_FREQ_CONTROL,
		0, // Always 0 for endpoint
		endpoint,
		data,
	)
}

// GetSamplingFrequency gets the current sampling frequency
func (c *UACControl) GetSamplingFrequency(endpoint uint8) (uint32, error) {
	data := make([]byte, 3)

	err := c.controlTransfer(
		0xA2, // bmRequestType: Class, Endpoint, Device to Host
		GET_CUR,
		EP_SAMPLING_FREQ_CONTROL,
		0,
		endpoint,
		data,
	)

	freq := uint32(data[0]) | (uint32(data[1]) << 8) | (uint32(data[2]) << 16)
	return freq, err
}

// SetPitch sets the pitch control (1.0 = 0x10000)
func (c *UACControl) SetPitch(endpoint uint8, pitch uint32) error {
	data := []byte{
		byte(pitch & 0xFF),
		byte((pitch >> 8) & 0xFF),
		byte((pitch >> 16) & 0xFF),
		byte((pitch >> 24) & 0xFF),
	}

	return c.controlTransfer(
		0x22,
		SET_CUR,
		EP_PITCH_CONTROL,
		0,
		endpoint,
		data,
	)
}

// Selector Unit Control

// SetSelector sets the input selector for a selector unit
func (c *UACControl) SetSelector(unitID uint8, inputNum uint8) error {
	data := []byte{inputNum}

	return c.controlTransfer(
		0x21,
		SET_CUR,
		SU_SELECTOR_CONTROL,
		unitID,
		0, // No channel for selector
		data,
	)
}

// GetSelector gets the current input selector
func (c *UACControl) GetSelector(unitID uint8) (uint8, error) {
	data := make([]byte, 1)

	err := c.controlTransfer(
		0xA1,
		GET_CUR,
		SU_SELECTOR_CONTROL,
		unitID,
		0,
		data,
	)

	return data[0], err
}

// Processing Unit Controls

// SetProcessingEnable enables/disables a processing unit
func (c *UACControl) SetProcessingEnable(unitID uint8, enable bool) error {
	data := []byte{0x00}
	if enable {
		data[0] = 0x01
	}

	return c.controlTransfer(
		0x21,
		SET_CUR,
		PU_ENABLE_CONTROL,
		unitID,
		0,
		data,
	)
}

// SetReverbLevel sets reverb level for a reverb processing unit
func (c *UACControl) SetReverbLevel(unitID uint8, level uint8) error {
	data := []byte{level}

	return c.controlTransfer(
		0x21,
		SET_CUR,
		PU_REVERB_LEVEL_CONTROL,
		unitID,
		0,
		data,
	)
}

// SetCompressionRatio sets compression ratio (1:N where N = value/256)
func (c *UACControl) SetCompressionRatio(unitID uint8, ratio uint16) error {
	data := []byte{
		byte(ratio & 0xFF),
		byte((ratio >> 8) & 0xFF),
	}

	return c.controlTransfer(
		0x21,
		SET_CUR,
		PU_COMPRESSION_RATIO_CONTROL,
		unitID,
		0,
		data,
	)
}

// Helper function for control transfers
func (c *UACControl) controlTransfer(bmRequestType uint8, bRequest uint8,
	controlSelector uint8, unitID uint8, channelOrEndpoint uint8, data []byte) error {

	// wValue: Control Selector in high byte, Channel/Endpoint in low byte
	wValue := (uint16(controlSelector) << 8) | uint16(channelOrEndpoint)

	// wIndex: Unit ID in high byte, Interface in low byte
	// For endpoint requests, wIndex is just the endpoint address
	var wIndex uint16
	if bmRequestType&0x1F == 0x02 { // Endpoint type
		wIndex = uint16(channelOrEndpoint)
	} else {
		wIndex = (uint16(unitID) << 8) | uint16(c.ifnum)
	}

	ret := C.libusb_control_transfer(
		c.handle,
		C.uint8_t(bmRequestType),
		C.uint8_t(bRequest),
		C.uint16_t(wValue),
		C.uint16_t(wIndex),
		(*C.uchar)(unsafe.Pointer(&data[0])),
		C.uint16_t(len(data)),
		1000, // 1 second timeout
	)

	if ret < 0 {
		return fmt.Errorf("control transfer failed: %s", C.GoString(C.libusb_error_name(ret)))
	}

	return nil
}

// GetStatus gets the status/interrupt data for a control
func (c *UACControl) GetStatus(unitID uint8, controlSelector uint8) ([]byte, error) {
	// Status data can be up to 2 bytes (interrupt status word)
	data := make([]byte, 2)

	err := c.controlTransfer(
		0xA1,
		GET_STAT,
		controlSelector,
		unitID,
		0,
		data,
	)

	return data, err
}

// Memory operations for storing/recalling settings

// SetMemory stores current settings to a memory location
func (c *UACControl) SetMemory(unitID uint8, controlSelector uint8, memoryLocation uint8, data []byte) error {
	return c.controlTransfer(
		0x21,
		SET_MEM,
		controlSelector,
		unitID,
		memoryLocation,
		data,
	)
}

// GetMemory recalls settings from a memory location
func (c *UACControl) GetMemory(unitID uint8, controlSelector uint8, memoryLocation uint8, dataLen int) ([]byte, error) {
	data := make([]byte, dataLen)

	err := c.controlTransfer(
		0xA1,
		GET_MEM,
		controlSelector,
		unitID,
		memoryLocation,
		data,
	)

	return data, err
}
