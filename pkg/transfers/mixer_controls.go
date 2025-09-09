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

// Mixer Unit Control Selectors
const (
	MU_CONTROL_UNDEFINED = 0x00
	MU_MIXER_CONTROL     = 0x01
)

// Extension Unit Control Selectors
const (
	XU_CONTROL_UNDEFINED = 0x00
	XU_ENABLE_CONTROL    = 0x01
	// Extension-specific controls start at 0x02
)

// MixerControl provides control over mixer units
type MixerControl struct {
	handle *C.struct_libusb_device_handle
	ifnum  uint8
}

func NewMixerControl(handle unsafe.Pointer, interfaceNumber uint8) *MixerControl {
	return &MixerControl{
		handle: (*C.struct_libusb_device_handle)(handle),
		ifnum:  interfaceNumber,
	}
}

// SetMixerLevel sets the mixing level between an input and output channel
// Level is in dB * 256 (signed 16-bit)
func (m *MixerControl) SetMixerLevel(unitID uint8, inputChannel uint8, outputChannel uint8, levelDB int16) error {
	data := []byte{
		byte(levelDB & 0xFF),
		byte((levelDB >> 8) & 0xFF),
	}

	// For mixer control, the channel number encodes both input and output
	// Format: (input << 4) | output (for up to 16 channels each)
	channelPair := (inputChannel << 4) | outputChannel

	return m.controlTransfer(
		0x21,
		SET_CUR,
		MU_MIXER_CONTROL,
		unitID,
		channelPair,
		data,
	)
}

// GetMixerLevel gets the mixing level between an input and output channel
func (m *MixerControl) GetMixerLevel(unitID uint8, inputChannel uint8, outputChannel uint8) (int16, error) {
	data := make([]byte, 2)
	channelPair := (inputChannel << 4) | outputChannel

	err := m.controlTransfer(
		0xA1,
		GET_CUR,
		MU_MIXER_CONTROL,
		unitID,
		channelPair,
		data,
	)

	level := int16(data[0]) | (int16(data[1]) << 8)
	return level, err
}

// GetMixerLevelRange gets the min, max, and resolution for mixer level
func (m *MixerControl) GetMixerLevelRange(unitID uint8, inputChannel uint8, outputChannel uint8) (min, max, res int16, err error) {
	data := make([]byte, 2)
	channelPair := (inputChannel << 4) | outputChannel

	// Get MIN
	err = m.controlTransfer(0xA1, GET_MIN, MU_MIXER_CONTROL, unitID, channelPair, data)
	if err != nil {
		return
	}
	min = int16(data[0]) | (int16(data[1]) << 8)

	// Get MAX
	err = m.controlTransfer(0xA1, GET_MAX, MU_MIXER_CONTROL, unitID, channelPair, data)
	if err != nil {
		return
	}
	max = int16(data[0]) | (int16(data[1]) << 8)

	// Get RES
	err = m.controlTransfer(0xA1, GET_RES, MU_MIXER_CONTROL, unitID, channelPair, data)
	if err != nil {
		return
	}
	res = int16(data[0]) | (int16(data[1]) << 8)

	return
}

// SetMixerMatrix sets the entire mixer matrix at once
// Matrix is row-major: matrix[input][output]
func (m *MixerControl) SetMixerMatrix(unitID uint8, numInputs uint8, numOutputs uint8, matrix [][]int16) error {
	if len(matrix) != int(numInputs) {
		return fmt.Errorf("matrix rows must equal number of inputs")
	}

	for i := uint8(0); i < numInputs; i++ {
		if len(matrix[i]) != int(numOutputs) {
			return fmt.Errorf("matrix columns must equal number of outputs")
		}

		for o := uint8(0); o < numOutputs; o++ {
			err := m.SetMixerLevel(unitID, i+1, o+1, matrix[i][o])
			if err != nil {
				return fmt.Errorf("failed to set mixer[%d][%d]: %w", i, o, err)
			}
		}
	}

	return nil
}

// GetMixerMatrix gets the entire mixer matrix
func (m *MixerControl) GetMixerMatrix(unitID uint8, numInputs uint8, numOutputs uint8) ([][]int16, error) {
	matrix := make([][]int16, numInputs)

	for i := uint8(0); i < numInputs; i++ {
		matrix[i] = make([]int16, numOutputs)

		for o := uint8(0); o < numOutputs; o++ {
			level, err := m.GetMixerLevel(unitID, i+1, o+1)
			if err != nil {
				return nil, fmt.Errorf("failed to get mixer[%d][%d]: %w", i, o, err)
			}
			matrix[i][o] = level
		}
	}

	return matrix, nil
}

// Extension Unit Controls

// SetExtensionEnable enables/disables an extension unit
func (m *MixerControl) SetExtensionEnable(unitID uint8, enable bool) error {
	data := []byte{0x00}
	if enable {
		data[0] = 0x01
	}

	return m.controlTransfer(
		0x21,
		SET_CUR,
		XU_ENABLE_CONTROL,
		unitID,
		0,
		data,
	)
}

// SetExtensionControl sets a vendor-specific extension control
func (m *MixerControl) SetExtensionControl(unitID uint8, controlID uint8, data []byte) error {
	return m.controlTransfer(
		0x21,
		SET_CUR,
		controlID,
		unitID,
		0,
		data,
	)
}

// GetExtensionControl gets a vendor-specific extension control
func (m *MixerControl) GetExtensionControl(unitID uint8, controlID uint8, dataLen int) ([]byte, error) {
	data := make([]byte, dataLen)

	err := m.controlTransfer(
		0xA1,
		GET_CUR,
		controlID,
		unitID,
		0,
		data,
	)

	return data, err
}

// Helper function for control transfers
func (m *MixerControl) controlTransfer(bmRequestType uint8, bRequest uint8,
	controlSelector uint8, unitID uint8, channel uint8, data []byte) error {

	// wValue: Control Selector in high byte, Channel in low byte
	wValue := (uint16(controlSelector) << 8) | uint16(channel)

	// wIndex: Unit ID in high byte, Interface in low byte
	wIndex := (uint16(unitID) << 8) | uint16(m.ifnum)

	ret := C.libusb_control_transfer(
		m.handle,
		C.uint8_t(bmRequestType),
		C.uint8_t(bRequest),
		C.uint16_t(wValue),
		C.uint16_t(wIndex),
		(*C.uchar)(unsafe.Pointer(&data[0])),
		C.uint16_t(len(data)),
		1000,
	)

	if ret < 0 {
		return fmt.Errorf("mixer control transfer failed: %s", C.GoString(C.libusb_error_name(ret)))
	}

	return nil
}

// Utility functions for common mixer configurations

// SetCrossfade sets a crossfade between two inputs
// fade: 0.0 = full input1, 1.0 = full input2
func (m *MixerControl) SetCrossade(unitID uint8, input1, input2, output uint8, fade float32) error {
	if fade < 0 || fade > 1 {
		return fmt.Errorf("fade must be between 0.0 and 1.0")
	}

	// Calculate levels (assuming 0 dB = 0x0000, -inf = 0x8000)
	level1 := int16((1.0 - fade) * 0x7FFF)
	level2 := int16(fade * 0x7FFF)

	err := m.SetMixerLevel(unitID, input1, output, level1)
	if err != nil {
		return err
	}

	return m.SetMixerLevel(unitID, input2, output, level2)
}

// MuteInput mutes all outputs from a specific input
func (m *MixerControl) MuteInput(unitID uint8, inputChannel uint8, numOutputs uint8) error {
	for o := uint8(1); o <= numOutputs; o++ {
		err := m.SetMixerLevel(unitID, inputChannel, o, -32768) // -inf dB
		if err != nil {
			return err
		}
	}
	return nil
}

// UnmuteInput unmutes all outputs from a specific input (sets to 0 dB)
func (m *MixerControl) UnmuteInput(unitID uint8, inputChannel uint8, numOutputs uint8) error {
	for o := uint8(1); o <= numOutputs; o++ {
		err := m.SetMixerLevel(unitID, inputChannel, o, 0) // 0 dB
		if err != nil {
			return err
		}
	}
	return nil
}
