package uvc

type EndpointDescriptorSubtype int

const (
	EndpointDescriptorSubtypeUndefined EndpointDescriptorSubtype = 0x00
	EndpointDescriptorSubtypeGeneral                             = 0x01
	EndpointDescriptorSubtypeEndpoint                            = 0x02
	EndpointDescriptorSubtypeInterrupt                           = 0x03
)

type RequestCodes int

const (
	RequestCodesUndefined RequestCodes = 0x00
	RequestCodesSetCur                 = 0x01
	RequestCodesSetCurAll              = 0x11
	RequestCodesGetCur                 = 0x81
	RequestCodesGetMin                 = 0x82
	RequestCodesGetMax                 = 0x83
	RequestCodesGetRes                 = 0x84
	RequestCodesGetLen                 = 0x85
	RequestCodesGetInfo                = 0x86
	RequestCodesGetDef                 = 0x87
	RequestCodesGetCurAll              = 0x91
	RequestCodesGetMinAll              = 0x92
	RequestCodesGetMaxAll              = 0x93
	RequestCodesGetResAll              = 0x94
	RequestCodesGetDefAll              = 0x97
)

type InterfaceControlSelector int

const (
	InterfaceControlSelectorUndefined               InterfaceControlSelector = 0x00
	InterfaceControlSelectorVideoPowerModeControl                            = 0x01
	InterfaceControlSelectorRequestErrorCodeControl                          = 0x02
)

type TerminalControlSelector int

const (
	TerminalControlSelectorUndefined TerminalControlSelector = 0x00
)

type SelectorUnitControlSelector int

const (
	SelectorUnitControlSelectorUndefined SelectorUnitControlSelector = 0x00
	SelectorUnitInputSelectControl                                   = 0x01
)

type ProcessingUnitControlSelector int

const (
	ProcessingUnitControlSelectorUndefined           ProcessingUnitControlSelector = 0x00
	ProcessingUnitBacklightCompensationControl                                     = 0x01
	ProcessingUnitBrightnessControl                                                = 0x02
	ProcessingUnitContrastControl                                                  = 0x03
	ProcessingUnitGainControl                                                      = 0x04
	ProcessingUnitPowerLineFrequencyControl                                        = 0x05
	ProcessingUnitHueControl                                                       = 0x06
	ProcessingUnitSaturationControl                                                = 0x07
	ProcessingUnitSharpnessControl                                                 = 0x08
	ProcessingUnitGammaControl                                                     = 0x09
	ProcessingUnitWhiteBalanceTemperatureControl                                   = 0x0A
	ProcessingUnitWhiteBalanceTemperatureAutoControl                               = 0x0B
	ProcessingUnitWhiteBalanceComponentControl                                     = 0x0C
	ProcessingUnitWhiteBalanceComponentAutoControl                                 = 0x0D
	ProcessingUnitDigitalMultiplierControl                                         = 0x0E
	ProcessingUnitDigitalMultiplierLimitControl                                    = 0x0F
	ProcessingUnitHueAutoControl                                                   = 0x10
	ProcessingUnitAnalogVideoStandardControl                                       = 0x11
	ProcessingUnitAnalogVideoLockStatusControl                                     = 0x12
	ProcessingUnitContrastAutoControl                                              = 0x13
)

type EncodingUnitControlSelector int

const (
	EncodingUnitControlSelectorUndefined                 EncodingUnitControlSelector = 0x00
	EncodingUnitControlSelectorSelectLayerControl                                    = 0x01
	EncodingUnitControlSelectorProfileToolsetControl                                 = 0x02
	EncodingUnitControlSelectorVideoResolutionControl                                = 0x03
	EncodingUnitControlSelectorMinFrameIntervalControl                               = 0x04
	EncodingUnitControlSelectorSliceModeControl                                      = 0x05
	EncodingUnitControlSelectorRateControlModeControl                                = 0x06
	EncodingUnitControlSelectorAverageBitrateControl                                 = 0x07
	EncodingUnitControlSelectorCPBSizeControl                                        = 0x08
	EncodingUnitControlSelectorPeakBitRateControl                                    = 0x09
	EncodingUnitControlSelectorQuantizationParamsControl                             = 0x0A
	EncodingUnitControlSelectorSyncRefFrameControl                                   = 0x0B
	EncodingUnitControlSelectorLTRBufferControl                                      = 0x0C
	EncodingUnitControlSelectorLTRPictureControl                                     = 0x0D
	EncodingUnitControlSelectorLTRValidationControl                                  = 0x0E
	EncodingUnitControlSelectorLevelIDCControl                                       = 0x0F
	EncodingUnitControlSelectorSEIPayloadTypeControl                                 = 0x10
	EncodingUnitControlSelectorQPRangeControl                                        = 0x11
	EncodingUnitControlSelectorPriorityControl                                       = 0x12
	EncodingUnitControlSelectorStartOrStopLayerControl                               = 0x13
	EncodingUnitControlSelectorErrorResiliencyControl                                = 0x14
)

type ExtensionUnitControlSelector int

const (
	ExtensionUnitControlSelectorUndefined ExtensionUnitControlSelector = 0x00
)
