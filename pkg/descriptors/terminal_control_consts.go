package descriptors

type CameraTerminalControlSelector int

const (
	CameraTerminalControlSelectorUndefined                   CameraTerminalControlSelector = 0x00
	CameraTerminalControlSelectorScanningModeControl         CameraTerminalControlSelector = 0x01
	CameraTerminalControlSelectorAutoExposureModeControl     CameraTerminalControlSelector = 0x02
	CameraTerminalControlSelectorAutoExposurePriorityControl CameraTerminalControlSelector = 0x03
	CameraTerminalControlSelectorExposureTimeAbsoluteControl CameraTerminalControlSelector = 0x04
	CameraTerminalControlSelectorExposureTimeRelativeControl CameraTerminalControlSelector = 0x05
	CameraTerminalControlSelectorFocusAbsoluteControl        CameraTerminalControlSelector = 0x06
	CameraTerminalControlSelectorFocusRelativeControl        CameraTerminalControlSelector = 0x07
	CameraTerminalControlSelectorFocusAutoControl            CameraTerminalControlSelector = 0x08
	CameraTerminalControlSelectorIrisAbsoluteControl         CameraTerminalControlSelector = 0x09
	CameraTerminalControlSelectorIrisRelativeControl         CameraTerminalControlSelector = 0x0A
	CameraTerminalControlSelectorZoomAbsoluteControl         CameraTerminalControlSelector = 0x0B
	CameraTerminalControlSelectorZoomRelativeControl         CameraTerminalControlSelector = 0x0C
	CameraTerminalControlSelectorPanTiltAbsoluteControl      CameraTerminalControlSelector = 0x0D
	CameraTerminalControlSelectorPanTiltRelativeControl      CameraTerminalControlSelector = 0x0E
	CameraTerminalControlSelectorRollAbsoluteControl         CameraTerminalControlSelector = 0x0F
	CameraTerminalControlSelectorRollRelativeControl         CameraTerminalControlSelector = 0x10
	CameraTerminalControlSelectorPrivacyControl              CameraTerminalControlSelector = 0x11
	CameraTerminalControlSelectorFocusSimpleControl          CameraTerminalControlSelector = 0x12
	CameraTerminalControlSelectorWindowControl               CameraTerminalControlSelector = 0x13
	CameraTerminalControlSelectorRegionOfInterestControl     CameraTerminalControlSelector = 0x14
)

type ScanningMode int

const (
	ScanningModeInterlaced  ScanningMode = 0
	ScanningModeProgressive ScanningMode = 1
)

type AutoExposureMode int

const (
	AutoExposureModeManual           AutoExposureMode = 1
	AutoExposureModeAuto             AutoExposureMode = 2
	AutoExposureModeShutterPriority  AutoExposureMode = 4
	AutoExposureModeAperturePriority AutoExposureMode = 8
)

type AutoExposurePriority int

const (
	AutoExposurePriorityConstant AutoExposurePriority = 0
	AutoExposurePriorityDynamic  AutoExposurePriority = 1
)

type ExposureTimeRelative int

const (
	ExposureTimeRelativeDefault   ExposureTimeRelative = 0x00
	ExposureTimeRelativeIncrement ExposureTimeRelative = 0x01
	ExposureTimeRelativeDecrement ExposureTimeRelative = 0xFF
)

type FocusRelative uint8

const (
	FocusRelativeStop              FocusRelative = 0x00
	FocusRelativeNearDirection     FocusRelative = 0x01
	FocusRelativeInfiniteDirection FocusRelative = 0xFF
)

type FocusSimple uint8

const (
	FocusSimpleFullRange FocusSimple = 0x00
	FocusSimpleMacro     FocusSimple = 0x01 // Less than 0.3 meters
	FocusSimplePeople    FocusSimple = 0x02 // 0.3 meters to 3 meters
	FocusSimpleScene     FocusSimple = 0x03 // 3 meters to infinity
)

type IrisRelative uint8

const (
	IrisRelativeDefault   IrisRelative = 0x00
	IrisRelativeOpenStep  IrisRelative = 0x01
	IrisRelativeCloseStep IrisRelative = 0x02
)

type ZoomRelative uint8

const (
	ZoomRelativeStop      ZoomRelative = 0x00
	ZoomRelativeTelephoto ZoomRelative = 0x01
	ZoomRelativeWideAngle ZoomRelative = 0xFF
)

type PanRelative int

const (
	PanRelativeStop             PanRelative = 0x00
	PanRelativeClockwise        PanRelative = 0x01
	PanRelativeCounterClockwise PanRelative = 0xFF
)

type TiltRelative int

const (
	TiltRelativeStop TiltRelative = 0x00
	TiltRelativeUp   TiltRelative = 0x01
	TiltRelativeDown TiltRelative = 0xFF
)

type RollRelative int

const (
	RollRelativeStop             RollRelative = 0x00
	RollRelativeClockwise        RollRelative = 0x01
	RollRelativeCounterClockwise RollRelative = 0xFF
)

type StepUnits int

const (
	StepUnitsVideoFrames StepUnits = 0x00
	StepUnitsMiliseconds StepUnits = 0x01
)

type RegionOfInterestAutoControl int

const (
	RegionOfInterestAutoControlExposure           RegionOfInterestAutoControl = 0x00
	RegionOfInterestAutoControlIris               RegionOfInterestAutoControl = 0x01
	RegionOfInterestAutoControlWhiteBalance       RegionOfInterestAutoControl = 0x02
	RegionOfInterestAutoControlFocus              RegionOfInterestAutoControl = 0x03
	RegionOfInterestAutoControlFaceDetect         RegionOfInterestAutoControl = 0x04
	RegionOfInterestAutoControlDetectTrack        RegionOfInterestAutoControl = 0x05
	RegionOfInterestAutoControlImageStabilization RegionOfInterestAutoControl = 0x06
	RegionOfInterestAutoControlHigherQuality      RegionOfInterestAutoControl = 0x07
)
