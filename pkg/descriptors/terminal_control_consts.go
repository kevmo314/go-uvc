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
