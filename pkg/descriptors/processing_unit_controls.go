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
	ProcessingUnitDigitalMultiplierControl           ProcessingUnitControlSelector = 0x0E // Deprecated
	ProcessingUnitDigitalMultiplierLimitControl      ProcessingUnitControlSelector = 0x0F
	ProcessingUnitHueAutoControl                     ProcessingUnitControlSelector = 0x10
	ProcessingUnitAnalogVideoStandardControl         ProcessingUnitControlSelector = 0x11
	ProcessingUnitAnalogVideoLockStatusControl       ProcessingUnitControlSelector = 0x12
	ProcessingUnitContrastAutoControl                ProcessingUnitControlSelector = 0x13
)

type PowerLineFrequency int

const (
	PowerLineFrequencyDisabled PowerLineFrequency = 0
	PowerLineFrequency50Hz     PowerLineFrequency = 1
	PowerLineFrequency60Hz     PowerLineFrequency = 2
	PowerLineFrequencyAuto     PowerLineFrequency = 3
)

type AnalogVideoStandard int

const (
	AnalogVideoStandardNone    AnalogVideoStandard = 0
	AnalogVideoStandardNTSC525 AnalogVideoStandard = 1 // NTSC 525/60
	AnalogVideoStandardPAL625  AnalogVideoStandard = 2 // PAL 625/50
	AnalogVideoStandardSECAM   AnalogVideoStandard = 3 // SECAM 625/50
	AnalogVideoStandardNTSC625 AnalogVideoStandard = 4 // NTSC 625/50
	AnalogVideoStandardPAL525  AnalogVideoStandard = 5 // PAL 525/60
)

type AnalogVideoLockStatus int

const (
	AnalogVideoLockStatusLocked    AnalogVideoLockStatus = 0
	AnalogVideoLockStatusNotLocked AnalogVideoLockStatus = 1
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

type ContrastAutoControl struct {
	Auto uint16
}

func (cac *ContrastAutoControl) FeatureBit() int {
	return 18
}

func (cac *ContrastAutoControl) Value() ProcessingUnitControlSelector {
	return ProcessingUnitContrastAutoControl
}

func (cac *ContrastAutoControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, cac.Auto)
	return buf, nil
}

func (cac *ContrastAutoControl) UnmarshalBinary(buf []byte) error {
	cac.Auto = binary.LittleEndian.Uint16(buf)
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

type PowerLineFrequencyControl struct {
	Frequency PowerLineFrequency
}

func (plfc *PowerLineFrequencyControl) FeatureBit() int {
	return 10
}

func (plfc *PowerLineFrequencyControl) Value() ProcessingUnitControlSelector {
	return ProcessingUnitPowerLineFrequencyControl
}

func (plfc *PowerLineFrequencyControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, uint16(plfc.Frequency))
	return buf, nil
}

func (plfc *PowerLineFrequencyControl) UnmarshalBinary(buf []byte) error {
	plfc.Frequency = PowerLineFrequency(binary.LittleEndian.Uint16(buf))
	return nil
}

type HueControl struct {
	Hue uint16
}

func (hc *HueControl) FeatureBit() int {
	return 2
}

func (hc *HueControl) Value() ProcessingUnitControlSelector {
	return ProcessingUnitHueControl
}

func (hc *HueControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, hc.Hue)
	return buf, nil
}

func (hc *HueControl) UnmarshalBinary(buf []byte) error {
	hc.Hue = binary.LittleEndian.Uint16(buf)
	return nil
}

type HueAutoControl struct {
	Auto uint8
}

func (hac *HueAutoControl) FeatureBit() int {
	return 11
}

func (hac *HueAutoControl) Value() ProcessingUnitControlSelector {
	return ProcessingUnitHueAutoControl
}

func (hac *HueAutoControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 1)
	buf[0] = uint8(hac.Auto)
	return buf, nil
}

func (hac *HueAutoControl) UnmarshalBinary(buf []byte) error {
	hac.Auto = buf[0]
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

type SharpnessControl struct {
	Sharpness uint16
}

func (sc *SharpnessControl) FeatureBit() int {
	return 4
}

func (sc *SharpnessControl) Value() ProcessingUnitControlSelector {
	return ProcessingUnitSharpnessControl
}

func (sc *SharpnessControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, sc.Sharpness)
	return buf, nil
}

func (sc *SharpnessControl) UnmarshalBinary(buf []byte) error {
	sc.Sharpness = binary.LittleEndian.Uint16(buf)
	return nil
}

type GammaControl struct {
	Gamma uint16
}

func (gc *GammaControl) FeatureBit() int {
	return 5
}

func (gc *GammaControl) Value() ProcessingUnitControlSelector {
	return ProcessingUnitGammaControl
}

func (gc *GammaControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, gc.Gamma)
	return buf, nil
}

func (gc *GammaControl) UnmarshalBinary(buf []byte) error {
	gc.Gamma = binary.LittleEndian.Uint16(buf)
	return nil
}

type WhiteBalanceTemperatureControl struct {
	WhiteBalanceTemperature uint16
}

func (wbt *WhiteBalanceTemperatureControl) FeatureBit() int {
	return 6
}

func (wbt *WhiteBalanceTemperatureControl) Value() ProcessingUnitControlSelector {
	return ProcessingUnitWhiteBalanceTemperatureControl
}

func (wbt *WhiteBalanceTemperatureControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, wbt.WhiteBalanceTemperature)
	return buf, nil
}

func (wbt *WhiteBalanceTemperatureControl) UnmarshalBinary(buf []byte) error {
	wbt.WhiteBalanceTemperature = binary.LittleEndian.Uint16(buf)
	return nil
}

type WhiteBalanceTemperatureAutoControl struct {
	WhiteBalanceTemperatureAuto uint8
}

func (wbtac *WhiteBalanceTemperatureAutoControl) FeatureBit() int {
	return 12
}

func (wbtac *WhiteBalanceTemperatureAutoControl) Value() ProcessingUnitControlSelector {
	return ProcessingUnitWhiteBalanceTemperatureAutoControl
}

func (wbtac *WhiteBalanceTemperatureAutoControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 1)
	buf[0] = uint8(wbtac.WhiteBalanceTemperatureAuto)
	return buf, nil
}

func (wbtac *WhiteBalanceTemperatureAutoControl) UnmarshalBinary(buf []byte) error {
	wbtac.WhiteBalanceTemperatureAuto = buf[0]
	return nil
}

type WhiteBalanceComponentControl struct {
	Blue uint16
	Red  uint16
}

func (wbcc *WhiteBalanceComponentControl) FeatureBit() int {
	return 7
}

func (wbcc *WhiteBalanceComponentControl) Value() ProcessingUnitControlSelector {
	return ProcessingUnitWhiteBalanceComponentControl
}

func (wbcc *WhiteBalanceComponentControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 4)
	binary.LittleEndian.AppendUint16(buf[0:2], wbcc.Blue)
	binary.LittleEndian.AppendUint16(buf[2:4], wbcc.Red)
	return buf, nil
}

func (wbcc *WhiteBalanceComponentControl) UnmarshalBinary(buf []byte) error {
	wbcc.Blue = binary.LittleEndian.Uint16(buf[0:2])
	wbcc.Blue = binary.LittleEndian.Uint16(buf[2:4])
	return nil
}

type WhiteBalanceComponentAutoControl struct {
	WhiteBalanceComponentAuto uint8
}

func (wbcac *WhiteBalanceComponentAutoControl) FeatureBit() int {
	return 13
}

func (wbcac *WhiteBalanceComponentAutoControl) Value() ProcessingUnitControlSelector {
	return ProcessingUnitWhiteBalanceComponentAutoControl
}

func (wbcac *WhiteBalanceComponentAutoControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 1)
	buf[0] = uint8(wbcac.WhiteBalanceComponentAuto)
	return buf, nil
}

func (wbcac *WhiteBalanceComponentAutoControl) UnmarshalBinary(buf []byte) error {
	wbcac.WhiteBalanceComponentAuto = buf[0]
	return nil
}

// Deprecated in 1.5
type DigitalMultiplerControl struct {
	DigitalMultipler uint16
}

func (dmc *DigitalMultiplerControl) FeatureBit() int {
	return 14
}

func (dmc *DigitalMultiplerControl) Value() ProcessingUnitControlSelector {
	return ProcessingUnitDigitalMultiplierControl
}

func (dmc *DigitalMultiplerControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, dmc.DigitalMultipler)
	return buf, nil
}

func (dmc *DigitalMultiplerControl) UnmarshalBinary(buf []byte) error {
	dmc.DigitalMultipler = binary.LittleEndian.Uint16(buf)
	return nil
}

type DigitalMultiplerLimitControl struct {
	DigitalMultiplerLimit uint16
}

func (dmlc *DigitalMultiplerLimitControl) FeatureBit() int {
	return 15
}

func (dmlc *DigitalMultiplerLimitControl) Value() ProcessingUnitControlSelector {
	return ProcessingUnitDigitalMultiplierLimitControl
}

func (dmlc *DigitalMultiplerLimitControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, dmlc.DigitalMultiplerLimit)
	return buf, nil
}

func (dmlc *DigitalMultiplerLimitControl) UnmarshalBinary(buf []byte) error {
	dmlc.DigitalMultiplerLimit = binary.LittleEndian.Uint16(buf)
	return nil
}

type AnalogVideoStandardControl struct {
	AnalogVideoStandard AnalogVideoStandard
}

func (avsc *AnalogVideoStandardControl) FeatureBit() int {
	return 16
}

func (avsc *AnalogVideoStandardControl) Value() ProcessingUnitControlSelector {
	return ProcessingUnitAnalogVideoStandardControl
}

func (avsc *AnalogVideoStandardControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 1)
	buf[0] = uint8(avsc.AnalogVideoStandard)
	return buf, nil
}

func (avsc *AnalogVideoStandardControl) UnmarshalBinary(buf []byte) error {
	avsc.AnalogVideoStandard = AnalogVideoStandard(buf[0])
	return nil
}

type AnalogVideoLockStatusControl struct {
	AnalogVideoLockStatus AnalogVideoLockStatus
}

func (avlsc *AnalogVideoLockStatusControl) FeatureBit() int {
	return 17
}

func (avlsc *AnalogVideoLockStatusControl) Value() ProcessingUnitControlSelector {
	return ProcessingUnitAnalogVideoLockStatusControl
}

func (avlsc *AnalogVideoLockStatusControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 1)
	buf[0] = uint8(avlsc.AnalogVideoLockStatus)
	return buf, nil
}

func (avlsc *AnalogVideoLockStatusControl) UnmarshalBinary(buf []byte) error {
	avlsc.AnalogVideoLockStatus = AnalogVideoLockStatus(buf[0])
	return nil
}
