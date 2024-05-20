package descriptors

import (
	"encoding"
	"encoding/binary"
	"time"
)

type CameraTerminalControlDescriptor interface {
	Value() CameraTerminalControlSelector
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

type VideoProbeCommitControl struct {
	HintBitmask            uint16
	FormatIndex            uint8
	FrameIndex             uint8
	FrameInterval          time.Duration
	KeyFrameRate           uint16
	PFrameRate             uint16
	CompQuality            uint16
	CompWindowSize         uint16
	Delay                  uint16
	MaxVideoFrameSize      uint32
	MaxPayloadTransferSize uint32

	// added in uvc 1.1
	ClockFrequency     uint32
	FramingInfoBitmask uint8
	PreferedVersion    uint8
	MinVersion         uint8
	MaxVersion         uint8

	// added in uvc 1.5
	Usage                     uint8
	BitDepthLuma              uint8
	SettingsBitmask           uint8
	MaxNumberOfRefFramesPlus1 uint8
	RateControlModes          uint16
	LayoutPerStream           [4]uint16
}

func (vpcc *VideoProbeCommitControl) MarshalInto(buf []byte) error {
	binary.LittleEndian.PutUint16(buf[0:2], vpcc.HintBitmask)
	buf[2] = vpcc.FormatIndex
	buf[3] = vpcc.FrameIndex
	binary.LittleEndian.PutUint32(buf[4:8], uint32(vpcc.FrameInterval/100/time.Nanosecond))
	binary.LittleEndian.PutUint16(buf[8:10], vpcc.KeyFrameRate)
	binary.LittleEndian.PutUint16(buf[10:12], vpcc.PFrameRate)
	binary.LittleEndian.PutUint16(buf[12:14], vpcc.CompQuality)
	binary.LittleEndian.PutUint16(buf[14:16], vpcc.CompWindowSize)
	binary.LittleEndian.PutUint16(buf[16:18], vpcc.Delay)
	binary.LittleEndian.PutUint32(buf[18:22], vpcc.MaxVideoFrameSize)
	binary.LittleEndian.PutUint32(buf[22:26], vpcc.MaxPayloadTransferSize)
	if len(buf) > 26 {
		binary.LittleEndian.PutUint32(buf[26:30], vpcc.ClockFrequency)
		buf[30] = vpcc.FramingInfoBitmask
		buf[31] = vpcc.PreferedVersion
		buf[32] = vpcc.MinVersion
		buf[33] = vpcc.MaxVersion
	}

	if len(buf) > 34 {
		buf[34] = vpcc.Usage
		buf[35] = vpcc.BitDepthLuma
		buf[36] = vpcc.SettingsBitmask
		buf[37] = vpcc.MaxNumberOfRefFramesPlus1
		binary.LittleEndian.PutUint16(buf[38:40], vpcc.RateControlModes)
		binary.LittleEndian.PutUint16(buf[40:42], vpcc.LayoutPerStream[0])
		binary.LittleEndian.PutUint16(buf[42:44], vpcc.LayoutPerStream[1])
		binary.LittleEndian.PutUint16(buf[44:46], vpcc.LayoutPerStream[2])
		binary.LittleEndian.PutUint16(buf[46:48], vpcc.LayoutPerStream[3])
	}
	return nil
}

func (vpcc *VideoProbeCommitControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 48)
	return buf, vpcc.MarshalInto(buf)
}

func (vpcc *VideoProbeCommitControl) UnmarshalBinary(buf []byte) error {
	// this descriptor is not length and control-selector prefixed because
	// libusb unwraps the control transfers for us.
	vpcc.HintBitmask = binary.LittleEndian.Uint16(buf[0:2])
	vpcc.FormatIndex = buf[2]
	vpcc.FrameIndex = buf[3]
	vpcc.FrameInterval = time.Duration(binary.LittleEndian.Uint32(buf[4:8])) * 100 * time.Nanosecond

	vpcc.KeyFrameRate = binary.LittleEndian.Uint16(buf[8:10])
	vpcc.PFrameRate = binary.LittleEndian.Uint16(buf[10:12])

	vpcc.CompQuality = binary.LittleEndian.Uint16(buf[12:14])
	vpcc.CompWindowSize = binary.LittleEndian.Uint16(buf[14:16])

	vpcc.Delay = binary.LittleEndian.Uint16(buf[16:18])

	vpcc.MaxVideoFrameSize = binary.LittleEndian.Uint32(buf[18:22])
	vpcc.MaxPayloadTransferSize = binary.LittleEndian.Uint32(buf[22:26])

	if len(buf) > 26 {
		vpcc.ClockFrequency = binary.LittleEndian.Uint32(buf[26:30])
		vpcc.FramingInfoBitmask = buf[30]
		vpcc.PreferedVersion = buf[31]
		vpcc.MinVersion = buf[32]
		vpcc.MaxVersion = buf[33]
	}

	if len(buf) > 34 {
		vpcc.Usage = buf[34]
		vpcc.BitDepthLuma = buf[35]
		vpcc.SettingsBitmask = buf[36]
		vpcc.MaxNumberOfRefFramesPlus1 = buf[37]
		vpcc.RateControlModes = binary.LittleEndian.Uint16(buf[38:40])
		vpcc.LayoutPerStream[0] = binary.LittleEndian.Uint16(buf[40:42])
		vpcc.LayoutPerStream[1] = binary.LittleEndian.Uint16(buf[42:44])
		vpcc.LayoutPerStream[2] = binary.LittleEndian.Uint16(buf[44:46])
		vpcc.LayoutPerStream[3] = binary.LittleEndian.Uint16(buf[46:48])
	}
	return nil
}

// Control Request for Auto-Exposure Mode as defined in UVC spec 1.5, 4.2.2.1.2
type AutoExposureModeControl struct {
	Mode AutoExposureMode
}

func (aemc *AutoExposureModeControl) Value() CameraTerminalControlSelector {
	return CameraTerminalControlSelectorAutoExposurePriorityControl
}

func (aemc *AutoExposureModeControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 1)
	buf[0] = byte(aemc.Mode)
	return buf, nil
}

func (aemc *AutoExposureModeControl) UnmarshalBinary(buf []byte) error {
	aemc.Mode = AutoExposureMode(buf[0])
	return nil
}

// Control Request for Auto-Exposure Priority as defined in UVC spec 1.5, 4.2.2.1.3
type AutoExposurePriorityControl struct {
	Priority AutoExposurePriority
}

func (aepc *AutoExposurePriorityControl) Value() CameraTerminalControlSelector {
	return CameraTerminalControlSelectorAutoExposurePriorityControl
}

func (aepc *AutoExposurePriorityControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 1)
	buf[0] = byte(aepc.Priority)
	return buf, nil
}

func (aepc *AutoExposurePriorityControl) UnmarshalBinary(buf []byte) error {
	aepc.Priority = AutoExposurePriority(buf[0])
	return nil
}

// Control Request for Exposure Time (Absolute) as defined in UVC spec 1.5, 4.2.2.1.4
type ExposureTimeAbsoluteControl struct {
	Time uint32
}

func (etac *ExposureTimeAbsoluteControl) Value() CameraTerminalControlSelector {
	return CameraTerminalControlSelectorExposureTimeAbsoluteControl
}

func (etac *ExposureTimeAbsoluteControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, etac.Time)
	return buf, nil
}

func (etrc *ExposureTimeAbsoluteControl) UnmarshalBinary(buf []byte) error {
	etrc.Time = binary.LittleEndian.Uint32(buf)
	return nil
}

// Control Request for Exposure Time (Relative) as defined in UVC spec 1.5, 4.2.2.1.5
type ExposureTimeRelativeControl struct {
	Time ExposureTimeRelative
}

func (etrc *ExposureTimeRelativeControl) Value() CameraTerminalControlSelector {
	return CameraTerminalControlSelectorExposureTimeRelativeControl
}

func (etrc *ExposureTimeRelativeControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 1)
	buf[0] = byte(etrc.Time)
	return buf, nil
}

func (etrc *ExposureTimeRelativeControl) UnmarshalBinary(buf []byte) error {
	etrc.Time = ExposureTimeRelative(buf[0])
	return nil
}

// Control Request for Focus, Auto Control as defined in UVC spec 1.5, 4.2.2.1.9
type FocusAutoControl struct {
	FocusAuto bool
}

func (fac *FocusAutoControl) Value() CameraTerminalControlSelector {
	return CameraTerminalControlSelectorFocusAutoControl
}

func (fac *FocusAutoControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 1)
	byteValue := byte(0)
	if fac.FocusAuto {
		byteValue = byte(1)
	}
	buf[0] = byteValue
	return buf, nil
}

func (fac *FocusAutoControl) UnmarshalBinary(buf []byte) error {
	fac.FocusAuto = buf[0] == 1
	return nil
}
