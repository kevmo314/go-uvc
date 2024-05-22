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

// Control Request for Focus Absolute as defined in UVC spec 1.5, 4.2.2.1.6
type FocusAbsoluteControl struct {
	Focus uint16
}

func (fac *FocusAbsoluteControl) Value() CameraTerminalControlSelector {
	return CameraTerminalControlSelectorFocusAbsoluteControl
}

func (fac *FocusAbsoluteControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, fac.Focus)
	return buf, nil
}

func (fac *FocusAbsoluteControl) UnmarshalBinary(buf []byte) error {
	fac.Focus = binary.LittleEndian.Uint16(buf)
	return nil
}

// Control Request for Focus Relative as defined in UVC spec 1.5, 4.2.2.1.7
type FocusRelativeControl struct {
	Focus FocusRelative
	Speed uint8
}

func (frc *FocusRelativeControl) Value() CameraTerminalControlSelector {
	return CameraTerminalControlSelectorFocusRelativeControl
}

func (frc *FocusRelativeControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 2)
	buf[0] = uint8(frc.Focus)
	buf[1] = frc.Speed
	return buf, nil
}

func (frc *FocusRelativeControl) UnmarshalBinary(buf []byte) error {
	frc.Focus = FocusRelative(buf[0])
	frc.Speed = buf[1]
	return nil
}

// Control Request for Focus Simple Range as defined in UVC spec 1.5, 4.2.2.1.8
type FocusSimpleRangeControl struct {
	Focus FocusSimple
}

func (fsrc *FocusSimpleRangeControl) Value() CameraTerminalControlSelector {
	return CameraTerminalControlSelectorFocusSimpleControl
}

func (fsrc *FocusSimpleRangeControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 1)
	buf[0] = uint8(fsrc.Focus)
	return buf, nil
}

func (fsrc *FocusSimpleRangeControl) UnmarshalBinary(buf []byte) error {
	fsrc.Focus = FocusSimple(buf[0])
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

// Control Request for Iris Absolute as defined in UVC spec 1.5, 4.2.2.1.10
type IrisAbsoluteControl struct {
	Aperture uint16
}

func (iac *IrisAbsoluteControl) Value() CameraTerminalControlSelector {
	return CameraTerminalControlSelectorIrisAbsoluteControl
}

func (iac *IrisAbsoluteControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, iac.Aperture)
	return buf, nil
}

func (iac *IrisAbsoluteControl) UnmarshalBinary(buf []byte) error {
	iac.Aperture = binary.LittleEndian.Uint16(buf)
	return nil
}

// Control Request for Iris Relative as defined in UVC spec 1.5, 4.2.2.1.11
type IrisRelativeControl struct {
	Aperture IrisRelative
}

func (irc *IrisRelativeControl) Value() CameraTerminalControlSelector {
	return CameraTerminalControlSelectorIrisRelativeControl
}

func (irc *IrisRelativeControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 1)
	buf[0] = byte(irc.Aperture)
	return buf, nil
}

func (irc *IrisRelativeControl) UnmarshalBinary(buf []byte) error {
	irc.Aperture = IrisRelative(buf[0])
	return nil
}

// Control Request for Zoom Absolute as defined in UVC spec 1.5, 4.2.2.1.12
type ZoomAbsoluteControl struct {
	ObjectiveFocalLength uint16
}

func (zac *ZoomAbsoluteControl) Value() CameraTerminalControlSelector {
	return CameraTerminalControlSelectorZoomAbsoluteControl
}

func (zac *ZoomAbsoluteControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, uint16(zac.ObjectiveFocalLength))
	return buf, nil
}

func (zac *ZoomAbsoluteControl) UnmarshalBinary(buf []byte) error {
	zac.ObjectiveFocalLength = binary.LittleEndian.Uint16(buf)
	return nil
}

// Control Request for Zoom Relative as defined in UVC spec 1.5, 4.2.2.1.13
type ZoomRelativeControl struct {
	Zoom        ZoomRelative
	DigitalZoom bool
	Speed       uint8
}

func (zrc *ZoomRelativeControl) Value() CameraTerminalControlSelector {
	return CameraTerminalControlSelectorZoomRelativeControl
}

func (zrc *ZoomRelativeControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 3)
	buf[0] = byte(zrc.Zoom)

	byteValue := byte(0)
	if zrc.DigitalZoom {
		byteValue = byte(1)
	}
	buf[1] = byteValue

	buf[2] = zrc.Speed
	return buf, nil
}

func (zrc *ZoomRelativeControl) UnmarshalBinary(buf []byte) error {
	zrc.Zoom = ZoomRelative(buf[0])
	zrc.DigitalZoom = buf[1] == 1
	zrc.Speed = buf[2]
	return nil
}

// Control Request for Pan Tilt Absolute as defined in UVC spec 1.5, 4.2.2.1.14
type PanTiltAbsoluteControl struct {
	PanAbsolute  int32
	TiltAbsolute int32
}

func (ptac *PanTiltAbsoluteControl) Value() CameraTerminalControlSelector {
	return CameraTerminalControlSelectorPanTiltAbsoluteControl
}

func (ptac *PanTiltAbsoluteControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(ptac.PanAbsolute))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(ptac.TiltAbsolute))
	return buf, nil
}

func (ptac *PanTiltAbsoluteControl) UnmarshalBinary(buf []byte) error {
	ptac.PanAbsolute = int32(binary.LittleEndian.Uint32(buf[0:4]))
	ptac.TiltAbsolute = int32(binary.LittleEndian.Uint32(buf[4:8]))
	return nil
}

// Control Request for Pan Tilt Relative as defined in UVC spec 1.5, 4.2.2.1.15
type PanTiltRelativeControl struct {
	PanRelative  PanRelative
	PanSpeed     uint8
	TiltRelative TiltRelative
	TiltSpeed    uint8
}

func (ptrc *PanTiltRelativeControl) Value() CameraTerminalControlSelector {
	return CameraTerminalControlSelectorPanTiltRelativeControl
}

func (ptrc *PanTiltRelativeControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 8)
	buf[0] = byte(ptrc.PanRelative)
	buf[1] = byte(ptrc.PanSpeed)
	buf[2] = byte(ptrc.TiltRelative)
	buf[3] = byte(ptrc.TiltSpeed)
	return buf, nil
}

func (ptrc *PanTiltRelativeControl) UnmarshalBinary(buf []byte) error {
	ptrc.PanRelative = PanRelative(buf[0])
	ptrc.PanSpeed = uint8(buf[1])
	ptrc.TiltRelative = TiltRelative(buf[2])
	ptrc.TiltSpeed = uint8(buf[3])
	return nil
}

// Control Request for Roll Absolute as defined in UVC spec 1.5, 4.2.2.1.16
type RollAbsoluteControl struct {
	RollAbsolute int16
}

func (rac *RollAbsoluteControl) Value() CameraTerminalControlSelector {
	return CameraTerminalControlSelectorRollAbsoluteControl
}

func (rac *RollAbsoluteControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, uint16(rac.RollAbsolute))
	return buf, nil
}

func (rac *RollAbsoluteControl) UnmarshalBinary(buf []byte) error {
	rac.RollAbsolute = int16(binary.LittleEndian.Uint32(buf))
	return nil
}

// Control Request for Roll Relative as defined in UVC spec 1.5, 4.2.2.1.17
type RollRelativeControl struct {
	RollRelative RollRelative
	Speed        uint8
}

func (rrc *RollRelativeControl) Value() CameraTerminalControlSelector {
	return CameraTerminalControlSelectorRollRelativeControl
}

func (rrc *RollRelativeControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 2)
	buf[0] = byte(rrc.RollRelative)
	buf[1] = rrc.Speed
	return buf, nil
}

func (rrc *RollRelativeControl) UnmarshalBinary(buf []byte) error {
	rrc.RollRelative = RollRelative(buf[0])
	rrc.Speed = buf[1]
	return nil
}

// Control Request for Privacy Control as defined in UVC spec 1.5, 4.2.2.1.18
type PrivacyControl struct {
	Privacy bool
}

func (pc *PrivacyControl) Value() CameraTerminalControlSelector {
	return CameraTerminalControlSelectorPrivacyControl
}

func (pc *PrivacyControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 1)

	byteValue := byte(0)
	if pc.Privacy {
		byteValue = byte(1)
	}
	buf[0] = byteValue

	return buf, nil
}

func (pc *PrivacyControl) UnmarshalBinary(buf []byte) error {
	pc.Privacy = buf[0] == 1
	return nil
}

// Control Request for Digital Window as defined in UVC spec 1.5, 4.2.2.1.19
type DigitalWindowControl struct {
	//TODO where should we validate bottom >= top and right >= left ?
	Top    int16 // Pixels
	Left   int16 // Pixels
	Bottom int16 // Pixels
	Right  int16 // Pixels

	Steps      int16
	StepsUnits StepUnits
}

func (dwc *DigitalWindowControl) Value() CameraTerminalControlSelector {
	return CameraTerminalControlSelectorWindowControl
}

func (dwc *DigitalWindowControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 12)

	binary.LittleEndian.PutUint16(buf[0:2], uint16(dwc.Top))
	binary.LittleEndian.PutUint16(buf[2:4], uint16(dwc.Left))
	binary.LittleEndian.PutUint16(buf[4:6], uint16(dwc.Bottom))
	binary.LittleEndian.PutUint16(buf[6:8], uint16(dwc.Right))

	binary.LittleEndian.PutUint16(buf[8:10], uint16(dwc.Steps))
	binary.LittleEndian.PutUint16(buf[10:12], uint16(dwc.StepsUnits))

	return buf, nil
}

func (dwc *DigitalWindowControl) UnmarshalBinary(buf []byte) error {
	dwc.Top = int16(binary.LittleEndian.Uint16(buf[0:2]))
	dwc.Left = int16(binary.LittleEndian.Uint16(buf[2:4]))
	dwc.Bottom = int16(binary.LittleEndian.Uint16(buf[4:6]))
	dwc.Right = int16(binary.LittleEndian.Uint16(buf[6:8]))
	dwc.Steps = int16(binary.LittleEndian.Uint16(buf[8:10]))
	dwc.StepsUnits = StepUnits(binary.LittleEndian.Uint16(buf[10:12]))
	return nil
}

// Control Request for Digital Region of Interest as defined in UVC spec 1.5, 4.2.2.1.20
type RegionOfInterestControl struct {
	Top          int16
	Left         int16
	Bottom       int16
	Right        int16
	AutoControls RegionOfInterestAutoControl
}

func (roic *RegionOfInterestControl) Value() CameraTerminalControlSelector {
	return CameraTerminalControlSelectorRegionOfInterestControl
}

func (roic *RegionOfInterestControl) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 10)

	binary.LittleEndian.PutUint16(buf[0:2], uint16(roic.Top))
	binary.LittleEndian.PutUint16(buf[2:4], uint16(roic.Left))
	binary.LittleEndian.PutUint16(buf[4:6], uint16(roic.Bottom))
	binary.LittleEndian.PutUint16(buf[6:8], uint16(roic.Right))

	binary.LittleEndian.PutUint16(buf[8:10], uint16(roic.AutoControls))

	return buf, nil
}

func (roic *RegionOfInterestControl) UnmarshalBinary(buf []byte) error {
	roic.Top = int16(binary.LittleEndian.Uint16(buf[0:2]))
	roic.Left = int16(binary.LittleEndian.Uint16(buf[2:4]))
	roic.Bottom = int16(binary.LittleEndian.Uint16(buf[4:6]))
	roic.Right = int16(binary.LittleEndian.Uint16(buf[6:8]))
	roic.AutoControls = RegionOfInterestAutoControl(binary.LittleEndian.Uint16(buf[8:10]))
	return nil
}
