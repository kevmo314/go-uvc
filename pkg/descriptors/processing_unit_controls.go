package descriptors

import (
	"encoding"
	"encoding/binary"
)

type ProcessingUnitControlSelector int

const (
	ProcessingUnitControlSelectorUndefined           ProcessingUnitControlSelector = 0x00
	ProcessingUnitBacklightCompensationControl       ProcessingUnitControlSelector = 0x01
	ProcessingUnitBrightnessControl                  ProcessingUnitControlSelector = 0x02
	ProcessingUnitContrastControl                    ProcessingUnitControlSelector = 0x03
	ProcessingUnitGainControl                        ProcessingUnitControlSelector = 0x04
	ProcessingUnitPowerLineFrequencyControl          ProcessingUnitControlSelector = 0x05
	ProcessingUnitHueControl                         ProcessingUnitControlSelector = 0x06
	ProcessingUnitSaturationControl                  ProcessingUnitControlSelector = 0x07
	ProcessingUnitSharpnessControl                   ProcessingUnitControlSelector = 0x08
	ProcessingUnitGammaControl                       ProcessingUnitControlSelector = 0x09
	ProcessingUnitWhiteBalanceTemperatureControl     ProcessingUnitControlSelector = 0x0A
	ProcessingUnitWhiteBalanceTemperatureAutoControl ProcessingUnitControlSelector = 0x0B
	ProcessingUnitWhiteBalanceComponentControl       ProcessingUnitControlSelector = 0x0C
	ProcessingUnitWhiteBalanceComponentAutoControl   ProcessingUnitControlSelector = 0x0D
	ProcessingUnitDigitalMultiplierControl           ProcessingUnitControlSelector = 0x0E
	ProcessingUnitDigitalMultiplierLimitControl      ProcessingUnitControlSelector = 0x0F
	ProcessingUnitHueAutoControl                     ProcessingUnitControlSelector = 0x10
	ProcessingUnitAnalogVideoStandardControl         ProcessingUnitControlSelector = 0x11
	ProcessingUnitAnalogVideoLockStatusControl       ProcessingUnitControlSelector = 0x12
	ProcessingUnitContrastAutoControl                ProcessingUnitControlSelector = 0x13
)

type ProcessingUnitControlDescriptor interface {
	Value() ProcessingUnitControlSelector
	FeatureBit() int //Indicates the position of the control on the controls bitmap
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

type BacklightCompensationControl struct {
	BacklightCompensation uint16
}

func (bcc *BacklightCompensationControl) FeatureBit() int {
	return 8
}

func (bcc *BacklightCompensationControl) Value() ProcessingUnitControlSelector {
	return ProcessingUnitBacklightCompensationControl
}

func (bcc *BacklightCompensationControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, bcc.BacklightCompensation)
	return buf, nil
}

func (bcc *BacklightCompensationControl) UnmarshalBinary(buf []byte) error {
	bcc.BacklightCompensation = binary.LittleEndian.Uint16(buf)
	return nil
}

type BrightnessControl struct {
	Brightness uint16
}

func (bc *BrightnessControl) FeatureBit() int {
	return 0
}

func (bc *BrightnessControl) Value() ProcessingUnitControlSelector {
	return ProcessingUnitBrightnessControl
}

func (bc *BrightnessControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, bc.Brightness)
	return buf, nil
}

func (bc *BrightnessControl) UnmarshalBinary(buf []byte) error {
	bc.Brightness = binary.LittleEndian.Uint16(buf)
	return nil
}

type ContrastControl struct {
	Contrast uint16
}

func (cc *ContrastControl) FeatureBit() int {
	return 1
}

func (cc *ContrastControl) Value() ProcessingUnitControlSelector {
	return ProcessingUnitContrastControl
}

func (cc *ContrastControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, cc.Contrast)
	return buf, nil
}

func (cc *ContrastControl) UnmarshalBinary(buf []byte) error {
	cc.Contrast = binary.LittleEndian.Uint16(buf)
	return nil
}

type GainControl struct {
	Gain uint16
}

func (gc *GainControl) FeatureBit() int {
	return 9
}

func (gc *GainControl) Value() ProcessingUnitControlSelector {
	return ProcessingUnitGainControl
}

func (gc *GainControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, gc.Gain)
	return buf, nil
}

func (gc *GainControl) UnmarshalBinary(buf []byte) error {
	gc.Gain = binary.LittleEndian.Uint16(buf)
	return nil
}

type SaturationControl struct {
	Saturation uint16
}

func (sc *SaturationControl) FeatureBit() int {
	return 3
}

func (sc *SaturationControl) Value() ProcessingUnitControlSelector {
	return ProcessingUnitSaturationControl
}

func (sc *SaturationControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, sc.Saturation)
	return buf, nil
}

func (sc *SaturationControl) UnmarshalBinary(buf []byte) error {
	sc.Saturation = binary.LittleEndian.Uint16(buf)
	return nil
}
