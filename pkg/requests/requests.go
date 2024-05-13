package requests

type RequestType uint8

const (
	RequestTypeVideoInterfaceSetRequest RequestType = 0b00100001
	RequestTypeDataEndpointSetRequest   RequestType = 0b00100010
	RequestTypeVideoInterfaceGetRequest RequestType = 0b10100001
	RequestTypeDataEndpointGetRequest   RequestType = 0b10100010
)

type RequestCode uint8

const (
	RequestCodeUndefined RequestCode = 0x00
	RequestCodeSetCur    RequestCode = 0x01
	RequestCodeSetCurAll RequestCode = 0x11
	RequestCodeGetCur    RequestCode = 0x81
	RequestCodeGetMin    RequestCode = 0x82
	RequestCodeGetMax    RequestCode = 0x83
	RequestCodeGetRes    RequestCode = 0x84
	RequestCodeGetLen    RequestCode = 0x85
	RequestCodeGetInfo   RequestCode = 0x86
	RequestCodeGetDef    RequestCode = 0x87
	RequestCodeGetCurAll RequestCode = 0x91
	RequestCodeGetMinAll RequestCode = 0x92
	RequestCodeGetMaxAll RequestCode = 0x93
	RequestCodeGetResAll RequestCode = 0x94
	RequestCodeGetDefAll RequestCode = 0x97
)
