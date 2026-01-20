package uvc

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Media Foundation GUIDs
var (
	MF_DEVSOURCE_ATTRIBUTE_SOURCE_TYPE                      = windows.GUID{0xc60ac5fe, 0x252a, 0x478f, [8]byte{0xa0, 0xef, 0xbc, 0x8f, 0xa5, 0xf7, 0xca, 0xd3}}
	MF_DEVSOURCE_ATTRIBUTE_SOURCE_TYPE_VIDCAP               = windows.GUID{0x8ac3587a, 0x4ae7, 0x42d8, [8]byte{0x99, 0xe0, 0x0a, 0x60, 0x13, 0xee, 0xf9, 0x0f}}
	MF_DEVSOURCE_ATTRIBUTE_FRIENDLY_NAME                    = windows.GUID{0x60d0e559, 0x52f8, 0x4fa2, [8]byte{0xbb, 0xce, 0xac, 0xdb, 0x34, 0xa8, 0xec, 0x01}}
	MF_DEVSOURCE_ATTRIBUTE_SOURCE_TYPE_VIDCAP_SYMBOLIC_LINK = windows.GUID{0x58f0aad8, 0x22bf, 0x4f8a, [8]byte{0xbb, 0x3d, 0xd2, 0xc4, 0x97, 0x8c, 0x6e, 0x2f}}
	MF_MT_MAJOR_TYPE                                        = windows.GUID{0x48eba18e, 0xf8c9, 0x4687, [8]byte{0xbf, 0x11, 0x0a, 0x74, 0xc9, 0xf9, 0x6a, 0x8f}}
	MF_MT_SUBTYPE                                           = windows.GUID{0xf7e34c9a, 0x42e8, 0x4714, [8]byte{0xb7, 0x4b, 0xcb, 0x29, 0xd7, 0x2c, 0x35, 0xe5}}
	MF_MT_FRAME_SIZE                                        = windows.GUID{0x1652c33d, 0xd6b2, 0x4012, [8]byte{0xb8, 0x34, 0x72, 0x03, 0x08, 0x49, 0xa3, 0x7d}}
	MF_MT_FRAME_RATE                                        = windows.GUID{0xc459a2e8, 0x3d2c, 0x4e44, [8]byte{0xb1, 0x32, 0xfe, 0xe5, 0x15, 0x6c, 0x7b, 0xb0}}
	MFMediaType_Video                                       = windows.GUID{0x73646976, 0x0000, 0x0010, [8]byte{0x80, 0x00, 0x00, 0xaa, 0x00, 0x38, 0x9b, 0x71}}
	MFVideoFormat_MJPG                                      = windows.GUID{0x47504a4d, 0x0000, 0x0010, [8]byte{0x80, 0x00, 0x00, 0xaa, 0x00, 0x38, 0x9b, 0x71}}
	MFVideoFormat_NV12                                      = windows.GUID{0x3231564e, 0x0000, 0x0010, [8]byte{0x80, 0x00, 0x00, 0xaa, 0x00, 0x38, 0x9b, 0x71}}
	MFVideoFormat_YUY2                                      = windows.GUID{0x32595559, 0x0000, 0x0010, [8]byte{0x80, 0x00, 0x00, 0xaa, 0x00, 0x38, 0x9b, 0x71}}
	MFVideoFormat_RGB24                                     = windows.GUID{0x00000014, 0x0000, 0x0010, [8]byte{0x80, 0x00, 0x00, 0xaa, 0x00, 0x38, 0x9b, 0x71}}
	MFVideoFormat_RGB32                                     = windows.GUID{0x00000016, 0x0000, 0x0010, [8]byte{0x80, 0x00, 0x00, 0xaa, 0x00, 0x38, 0x9b, 0x71}}

	IID_IMFMediaSource   = windows.GUID{0x279a808d, 0xaec7, 0x40c8, [8]byte{0x9c, 0x6b, 0xa6, 0xb4, 0x92, 0xc7, 0x8a, 0x66}}
	IID_IMFSourceReader  = windows.GUID{0x70ae66f2, 0xc809, 0x4e4f, [8]byte{0x89, 0x15, 0xbd, 0xcb, 0x40, 0x6b, 0x79, 0x93}}
	IID_IAMCameraControl = windows.GUID{0xc6e13370, 0x30ac, 0x11d0, [8]byte{0xa1, 0x8c, 0x00, 0xa0, 0xc9, 0x11, 0x89, 0x56}}
	IID_IAMVideoProcAmp  = windows.GUID{0xc6e13360, 0x30ac, 0x11d0, [8]byte{0xa1, 0x8c, 0x00, 0xa0, 0xc9, 0x11, 0x89, 0x56}}
	IID_IUnknown         = windows.GUID{0x00000000, 0x0000, 0x0000, [8]byte{0xc0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
)

// Media Foundation constants
const (
	MF_SOURCE_READER_FIRST_VIDEO_STREAM = 0xFFFFFFFC
	MF_SOURCE_READER_ANY_STREAM         = 0xFFFFFFFE

	// CameraControl properties
	CameraControl_Pan      = 0
	CameraControl_Tilt     = 1
	CameraControl_Roll     = 2
	CameraControl_Zoom     = 3
	CameraControl_Exposure = 4
	CameraControl_Iris     = 5
	CameraControl_Focus    = 6

	// VideoProcAmp properties
	VideoProcAmp_Brightness         = 0
	VideoProcAmp_Contrast           = 1
	VideoProcAmp_Hue                = 2
	VideoProcAmp_Saturation         = 3
	VideoProcAmp_Sharpness          = 4
	VideoProcAmp_Gamma              = 5
	VideoProcAmp_ColorEnable        = 6
	VideoProcAmp_WhiteBalance       = 7
	VideoProcAmp_BacklightComp      = 8
	VideoProcAmp_Gain               = 9
	VideoProcAmp_DigitalMultiplier  = 10
	VideoProcAmp_DigitalMultLimit   = 11
	VideoProcAmp_WhiteBalanceComp   = 12
	VideoProcAmp_PowerLineFrequency = 13

	// Flags
	CameraControl_Flags_Auto   = 0x0001
	CameraControl_Flags_Manual = 0x0002
)

var (
	modmfplat      = windows.NewLazySystemDLL("mfplat.dll")
	modmfreadwrite = windows.NewLazySystemDLL("mfreadwrite.dll")
	modmf          = windows.NewLazySystemDLL("mf.dll")
	modole32       = windows.NewLazySystemDLL("ole32.dll")

	procMFStartup                           = modmfplat.NewProc("MFStartup")
	procMFShutdown                          = modmfplat.NewProc("MFShutdown")
	procMFEnumDeviceSources                 = modmf.NewProc("MFEnumDeviceSources")
	procMFCreateSourceReaderFromMediaSource = modmfreadwrite.NewProc("MFCreateSourceReaderFromMediaSource")
	procMFCreateAttributes                  = modmfplat.NewProc("MFCreateAttributes")
	procCoInitializeEx                      = modole32.NewProc("CoInitializeEx")
	procCoUninitialize                      = modole32.NewProc("CoUninitialize")
)

const (
	MF_VERSION               = 0x00020070 // MF 2.0
	COINIT_MULTITHREADED     = 0x0
	COINIT_APARTMENTTHREADED = 0x2
)

// IMFAttributes vtable
type IMFAttributesVtbl struct {
	QueryInterface     uintptr
	AddRef             uintptr
	Release            uintptr
	GetItem            uintptr
	GetItemType        uintptr
	CompareItem        uintptr
	Compare            uintptr
	GetUINT32          uintptr
	GetUINT64          uintptr
	GetDouble          uintptr
	GetGUID            uintptr
	GetStringLength    uintptr
	GetString          uintptr
	GetAllocatedString uintptr
	GetBlobSize        uintptr
	GetBlob            uintptr
	GetAllocatedBlob   uintptr
	GetUnknown         uintptr
	SetItem            uintptr
	DeleteItem         uintptr
	DeleteAllItems     uintptr
	SetUINT32          uintptr
	SetUINT64          uintptr
	SetDouble          uintptr
	SetGUID            uintptr
	SetString          uintptr
	SetBlob            uintptr
	SetUnknown         uintptr
	LockStore          uintptr
	UnlockStore        uintptr
	GetCount           uintptr
	GetItemByIndex     uintptr
	CopyAllItems       uintptr
}

type IMFAttributes struct {
	vtbl *IMFAttributesVtbl
}

func (a *IMFAttributes) Release() {
	if a != nil && a.vtbl != nil {
		syscall.SyscallN(a.vtbl.Release, uintptr(unsafe.Pointer(a)))
	}
}

func (a *IMFAttributes) SetGUID(key *windows.GUID, value *windows.GUID) error {
	hr, _, _ := syscall.SyscallN(a.vtbl.SetGUID,
		uintptr(unsafe.Pointer(a)),
		uintptr(unsafe.Pointer(key)),
		uintptr(unsafe.Pointer(value)))
	if hr != 0 {
		return fmt.Errorf("SetGUID failed: 0x%x", hr)
	}
	return nil
}

func (a *IMFAttributes) GetString(key *windows.GUID) (string, error) {
	var length uint32
	hr, _, _ := syscall.SyscallN(a.vtbl.GetStringLength,
		uintptr(unsafe.Pointer(a)),
		uintptr(unsafe.Pointer(key)),
		uintptr(unsafe.Pointer(&length)))
	if hr != 0 {
		return "", fmt.Errorf("GetStringLength failed: 0x%x", hr)
	}

	buf := make([]uint16, length+1)
	hr, _, _ = syscall.SyscallN(a.vtbl.GetString,
		uintptr(unsafe.Pointer(a)),
		uintptr(unsafe.Pointer(key)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(length+1),
		0)
	if hr != 0 {
		return "", fmt.Errorf("GetString failed: 0x%x", hr)
	}

	return windows.UTF16ToString(buf), nil
}

func (a *IMFAttributes) GetGUID(key *windows.GUID) (windows.GUID, error) {
	var guid windows.GUID
	hr, _, _ := syscall.SyscallN(a.vtbl.GetGUID,
		uintptr(unsafe.Pointer(a)),
		uintptr(unsafe.Pointer(key)),
		uintptr(unsafe.Pointer(&guid)))
	if hr != 0 {
		return guid, fmt.Errorf("GetGUID failed: 0x%x", hr)
	}
	return guid, nil
}

func (a *IMFAttributes) GetUINT64(key *windows.GUID) (uint64, error) {
	var val uint64
	hr, _, _ := syscall.SyscallN(a.vtbl.GetUINT64,
		uintptr(unsafe.Pointer(a)),
		uintptr(unsafe.Pointer(key)),
		uintptr(unsafe.Pointer(&val)))
	if hr != 0 {
		return 0, fmt.Errorf("GetUINT64 failed: 0x%x", hr)
	}
	return val, nil
}

// IMFActivate is an activation object that can create media sources
type IMFActivateVtbl struct {
	IMFAttributesVtbl
	ActivateObject uintptr
	ShutdownObject uintptr
	DetachObject   uintptr
}

type IMFActivate struct {
	vtbl *IMFActivateVtbl
}

func (a *IMFActivate) AsAttributes() *IMFAttributes {
	return (*IMFAttributes)(unsafe.Pointer(a))
}

func (a *IMFActivate) Release() {
	if a != nil && a.vtbl != nil {
		syscall.SyscallN(a.vtbl.Release, uintptr(unsafe.Pointer(a)))
	}
}

func (a *IMFActivate) ActivateObject(iid *windows.GUID) (uintptr, error) {
	var obj uintptr
	hr, _, _ := syscall.SyscallN(a.vtbl.ActivateObject,
		uintptr(unsafe.Pointer(a)),
		uintptr(unsafe.Pointer(iid)),
		uintptr(unsafe.Pointer(&obj)))
	if hr != 0 {
		return 0, fmt.Errorf("ActivateObject failed: 0x%x", hr)
	}
	return obj, nil
}

// IMFMediaSource vtable
type IMFMediaSourceVtbl struct {
	QueryInterface               uintptr
	AddRef                       uintptr
	Release                      uintptr
	GetEvent                     uintptr
	BeginGetEvent                uintptr
	EndGetEvent                  uintptr
	QueueEvent                   uintptr
	GetCharacteristics           uintptr
	CreatePresentationDescriptor uintptr
	Start                        uintptr
	Stop                         uintptr
	Pause                        uintptr
	Shutdown                     uintptr
}

type IMFMediaSource struct {
	vtbl *IMFMediaSourceVtbl
}

func (s *IMFMediaSource) Release() {
	if s != nil && s.vtbl != nil {
		syscall.SyscallN(s.vtbl.Release, uintptr(unsafe.Pointer(s)))
	}
}

func (s *IMFMediaSource) QueryInterface(iid *windows.GUID) (uintptr, error) {
	var obj uintptr
	hr, _, _ := syscall.SyscallN(s.vtbl.QueryInterface,
		uintptr(unsafe.Pointer(s)),
		uintptr(unsafe.Pointer(iid)),
		uintptr(unsafe.Pointer(&obj)))
	if hr != 0 {
		return 0, fmt.Errorf("QueryInterface failed: 0x%x", hr)
	}
	return obj, nil
}

func (s *IMFMediaSource) Shutdown() {
	if s != nil && s.vtbl != nil {
		syscall.SyscallN(s.vtbl.Shutdown, uintptr(unsafe.Pointer(s)))
	}
}

// IMFSourceReader vtable
type IMFSourceReaderVtbl struct {
	QueryInterface      uintptr
	AddRef              uintptr
	Release             uintptr
	GetStreamSelection  uintptr
	SetStreamSelection  uintptr
	GetNativeMediaType  uintptr
	GetCurrentMediaType uintptr
	SetCurrentMediaType uintptr
	SetCurrentPosition  uintptr
	ReadSample          uintptr
	Flush               uintptr
	GetServiceForStream uintptr
}

type IMFSourceReader struct {
	vtbl *IMFSourceReaderVtbl
}

func (r *IMFSourceReader) Release() {
	if r != nil && r.vtbl != nil {
		syscall.SyscallN(r.vtbl.Release, uintptr(unsafe.Pointer(r)))
	}
}

func (r *IMFSourceReader) GetNativeMediaType(streamIndex uint32, mediaTypeIndex uint32) (*IMFMediaType, error) {
	var mediaType *IMFMediaType
	hr, _, _ := syscall.SyscallN(r.vtbl.GetNativeMediaType,
		uintptr(unsafe.Pointer(r)),
		uintptr(streamIndex),
		uintptr(mediaTypeIndex),
		uintptr(unsafe.Pointer(&mediaType)))
	if hr != 0 {
		return nil, fmt.Errorf("GetNativeMediaType failed: 0x%x", hr)
	}
	return mediaType, nil
}

func (r *IMFSourceReader) SetCurrentMediaType(streamIndex uint32, mediaType *IMFMediaType) error {
	hr, _, _ := syscall.SyscallN(r.vtbl.SetCurrentMediaType,
		uintptr(unsafe.Pointer(r)),
		uintptr(streamIndex),
		0,
		uintptr(unsafe.Pointer(mediaType)))
	if hr != 0 {
		return fmt.Errorf("SetCurrentMediaType failed: 0x%x", hr)
	}
	return nil
}

func (r *IMFSourceReader) ReadSample(streamIndex uint32) (uint32, uint32, int64, *IMFSample, error) {
	var actualStreamIndex uint32
	var streamFlags uint32
	var timestamp int64
	var sample *IMFSample

	hr, _, _ := syscall.SyscallN(r.vtbl.ReadSample,
		uintptr(unsafe.Pointer(r)),
		uintptr(streamIndex),
		0,
		uintptr(unsafe.Pointer(&actualStreamIndex)),
		uintptr(unsafe.Pointer(&streamFlags)),
		uintptr(unsafe.Pointer(&timestamp)),
		uintptr(unsafe.Pointer(&sample)))
	if hr != 0 {
		return 0, 0, 0, nil, fmt.Errorf("ReadSample failed: 0x%x", hr)
	}
	return actualStreamIndex, streamFlags, timestamp, sample, nil
}

// IMFMediaType vtable (extends IMFAttributes)
type IMFMediaTypeVtbl struct {
	IMFAttributesVtbl
	GetMajorType       uintptr
	IsCompressedFormat uintptr
	IsEqual            uintptr
	GetRepresentation  uintptr
	FreeRepresentation uintptr
}

type IMFMediaType struct {
	vtbl *IMFMediaTypeVtbl
}

func (t *IMFMediaType) AsAttributes() *IMFAttributes {
	return (*IMFAttributes)(unsafe.Pointer(t))
}

func (t *IMFMediaType) Release() {
	if t != nil && t.vtbl != nil {
		syscall.SyscallN(t.vtbl.Release, uintptr(unsafe.Pointer(t)))
	}
}

// IMFSample vtable
type IMFSampleVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	// IMFAttributes methods
	GetItem            uintptr
	GetItemType        uintptr
	CompareItem        uintptr
	Compare            uintptr
	GetUINT32          uintptr
	GetUINT64          uintptr
	GetDouble          uintptr
	GetGUID            uintptr
	GetStringLength    uintptr
	GetString          uintptr
	GetAllocatedString uintptr
	GetBlobSize        uintptr
	GetBlob            uintptr
	GetAllocatedBlob   uintptr
	GetUnknown         uintptr
	SetItem            uintptr
	DeleteItem         uintptr
	DeleteAllItems     uintptr
	SetUINT32          uintptr
	SetUINT64          uintptr
	SetDouble          uintptr
	SetGUID            uintptr
	SetString          uintptr
	SetBlob            uintptr
	SetUnknown         uintptr
	LockStore          uintptr
	UnlockStore        uintptr
	GetCount           uintptr
	GetItemByIndex     uintptr
	CopyAllItems       uintptr
	// IMFSample methods
	GetSampleFlags            uintptr
	SetSampleFlags            uintptr
	GetSampleTime             uintptr
	SetSampleTime             uintptr
	GetSampleDuration         uintptr
	SetSampleDuration         uintptr
	GetBufferCount            uintptr
	GetBufferByIndex          uintptr
	ConvertToContiguousBuffer uintptr
	AddBuffer                 uintptr
	RemoveBufferByIndex       uintptr
	RemoveAllBuffers          uintptr
	GetTotalLength            uintptr
	CopyToBuffer              uintptr
}

type IMFSample struct {
	vtbl *IMFSampleVtbl
}

func (s *IMFSample) Release() {
	if s != nil && s.vtbl != nil {
		syscall.SyscallN(s.vtbl.Release, uintptr(unsafe.Pointer(s)))
	}
}

func (s *IMFSample) ConvertToContiguousBuffer() (*IMFMediaBuffer, error) {
	var buf *IMFMediaBuffer
	hr, _, _ := syscall.SyscallN(s.vtbl.ConvertToContiguousBuffer,
		uintptr(unsafe.Pointer(s)),
		uintptr(unsafe.Pointer(&buf)))
	if hr != 0 {
		return nil, fmt.Errorf("ConvertToContiguousBuffer failed: 0x%x", hr)
	}
	return buf, nil
}

// IMFMediaBuffer vtable
type IMFMediaBufferVtbl struct {
	QueryInterface   uintptr
	AddRef           uintptr
	Release          uintptr
	Lock             uintptr
	Unlock           uintptr
	GetCurrentLength uintptr
	SetCurrentLength uintptr
	GetMaxLength     uintptr
}

type IMFMediaBuffer struct {
	vtbl *IMFMediaBufferVtbl
}

func (b *IMFMediaBuffer) Release() {
	if b != nil && b.vtbl != nil {
		syscall.SyscallN(b.vtbl.Release, uintptr(unsafe.Pointer(b)))
	}
}

func (b *IMFMediaBuffer) Lock() ([]byte, error) {
	var ptr uintptr
	var maxLen, curLen uint32

	hr, _, _ := syscall.SyscallN(b.vtbl.Lock,
		uintptr(unsafe.Pointer(b)),
		uintptr(unsafe.Pointer(&ptr)),
		uintptr(unsafe.Pointer(&maxLen)),
		uintptr(unsafe.Pointer(&curLen)))
	if hr != 0 {
		return nil, fmt.Errorf("Lock failed: 0x%x", hr)
	}

	// Create a Go slice from the pointer
	data := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), curLen)
	result := make([]byte, curLen)
	copy(result, data)

	return result, nil
}

func (b *IMFMediaBuffer) Unlock() {
	syscall.SyscallN(b.vtbl.Unlock, uintptr(unsafe.Pointer(b)))
}

// IAMCameraControl vtable
type IAMCameraControlVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	GetRange       uintptr
	Set            uintptr
	Get            uintptr
}

type IAMCameraControl struct {
	vtbl *IAMCameraControlVtbl
}

func (c *IAMCameraControl) Release() {
	if c != nil && c.vtbl != nil {
		syscall.SyscallN(c.vtbl.Release, uintptr(unsafe.Pointer(c)))
	}
}

func (c *IAMCameraControl) GetRange(property int32) (min, max, step, def, flags int32, err error) {
	hr, _, _ := syscall.SyscallN(c.vtbl.GetRange,
		uintptr(unsafe.Pointer(c)),
		uintptr(property),
		uintptr(unsafe.Pointer(&min)),
		uintptr(unsafe.Pointer(&max)),
		uintptr(unsafe.Pointer(&step)),
		uintptr(unsafe.Pointer(&def)),
		uintptr(unsafe.Pointer(&flags)))
	if hr != 0 {
		return 0, 0, 0, 0, 0, fmt.Errorf("GetRange failed: 0x%x", hr)
	}
	return
}

func (c *IAMCameraControl) Get(property int32) (value, flags int32, err error) {
	hr, _, _ := syscall.SyscallN(c.vtbl.Get,
		uintptr(unsafe.Pointer(c)),
		uintptr(property),
		uintptr(unsafe.Pointer(&value)),
		uintptr(unsafe.Pointer(&flags)))
	if hr != 0 {
		return 0, 0, fmt.Errorf("Get failed: 0x%x", hr)
	}
	return
}

func (c *IAMCameraControl) Set(property, value, flags int32) error {
	hr, _, _ := syscall.SyscallN(c.vtbl.Set,
		uintptr(unsafe.Pointer(c)),
		uintptr(property),
		uintptr(value),
		uintptr(flags))
	if hr != 0 {
		return fmt.Errorf("Set failed: 0x%x", hr)
	}
	return nil
}

// IAMVideoProcAmp vtable
type IAMVideoProcAmpVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	GetRange       uintptr
	Set            uintptr
	Get            uintptr
}

type IAMVideoProcAmp struct {
	vtbl *IAMVideoProcAmpVtbl
}

func (v *IAMVideoProcAmp) Release() {
	if v != nil && v.vtbl != nil {
		syscall.SyscallN(v.vtbl.Release, uintptr(unsafe.Pointer(v)))
	}
}

func (v *IAMVideoProcAmp) GetRange(property int32) (min, max, step, def, flags int32, err error) {
	hr, _, _ := syscall.SyscallN(v.vtbl.GetRange,
		uintptr(unsafe.Pointer(v)),
		uintptr(property),
		uintptr(unsafe.Pointer(&min)),
		uintptr(unsafe.Pointer(&max)),
		uintptr(unsafe.Pointer(&step)),
		uintptr(unsafe.Pointer(&def)),
		uintptr(unsafe.Pointer(&flags)))
	if hr != 0 {
		return 0, 0, 0, 0, 0, fmt.Errorf("GetRange failed: 0x%x", hr)
	}
	return
}

func (v *IAMVideoProcAmp) Get(property int32) (value, flags int32, err error) {
	hr, _, _ := syscall.SyscallN(v.vtbl.Get,
		uintptr(unsafe.Pointer(v)),
		uintptr(property),
		uintptr(unsafe.Pointer(&value)),
		uintptr(unsafe.Pointer(&flags)))
	if hr != 0 {
		return 0, 0, fmt.Errorf("Get failed: 0x%x", hr)
	}
	return
}

func (v *IAMVideoProcAmp) Set(property, value, flags int32) error {
	hr, _, _ := syscall.SyscallN(v.vtbl.Set,
		uintptr(unsafe.Pointer(v)),
		uintptr(property),
		uintptr(value),
		uintptr(flags))
	if hr != 0 {
		return fmt.Errorf("Set failed: 0x%x", hr)
	}
	return nil
}

// MF helper functions
var mfInitialized bool

func mfStartup() error {
	if mfInitialized {
		return nil
	}

	// Initialize COM
	hr, _, _ := syscall.SyscallN(procCoInitializeEx.Addr(), 0, COINIT_MULTITHREADED)
	if hr != 0 && hr != 1 { // S_OK or S_FALSE (already initialized)
		return fmt.Errorf("CoInitializeEx failed: 0x%x", hr)
	}

	// Initialize Media Foundation
	hr, _, _ = syscall.SyscallN(procMFStartup.Addr(), MF_VERSION, 0)
	if hr != 0 {
		return fmt.Errorf("MFStartup failed: 0x%x", hr)
	}

	mfInitialized = true
	return nil
}

func mfShutdown() {
	if mfInitialized {
		syscall.SyscallN(procMFShutdown.Addr())
		syscall.SyscallN(procCoUninitialize.Addr())
		mfInitialized = false
	}
}

func mfCreateAttributes(count uint32) (*IMFAttributes, error) {
	var attrs *IMFAttributes
	hr, _, _ := syscall.SyscallN(procMFCreateAttributes.Addr(),
		uintptr(unsafe.Pointer(&attrs)),
		uintptr(count))
	if hr != 0 {
		return nil, fmt.Errorf("MFCreateAttributes failed: 0x%x", hr)
	}
	return attrs, nil
}

// MFCameraDevice represents a camera device found via Media Foundation
type MFCameraDevice struct {
	FriendlyName string
	SymbolicLink string
	Activate     *IMFActivate
}

// EnumerateMFCameras lists all video capture devices using Media Foundation
func EnumerateMFCameras() ([]*MFCameraDevice, error) {
	if err := mfStartup(); err != nil {
		return nil, err
	}

	// Create attributes to specify we want video capture devices
	attrs, err := mfCreateAttributes(1)
	if err != nil {
		return nil, err
	}
	defer attrs.Release()

	err = attrs.SetGUID(&MF_DEVSOURCE_ATTRIBUTE_SOURCE_TYPE, &MF_DEVSOURCE_ATTRIBUTE_SOURCE_TYPE_VIDCAP)
	if err != nil {
		return nil, err
	}

	// Enumerate devices
	var devices **IMFActivate
	var count uint32

	hr, _, _ := syscall.SyscallN(procMFEnumDeviceSources.Addr(),
		uintptr(unsafe.Pointer(attrs)),
		uintptr(unsafe.Pointer(&devices)),
		uintptr(unsafe.Pointer(&count)))
	if hr != 0 {
		return nil, fmt.Errorf("MFEnumDeviceSources failed: 0x%x", hr)
	}

	if count == 0 {
		return nil, nil
	}

	// Convert to slice
	deviceSlice := unsafe.Slice(devices, count)
	result := make([]*MFCameraDevice, 0, count)

	for _, activate := range deviceSlice {
		device := &MFCameraDevice{
			Activate: activate,
		}

		// Get friendly name
		if name, err := activate.AsAttributes().GetString(&MF_DEVSOURCE_ATTRIBUTE_FRIENDLY_NAME); err == nil {
			device.FriendlyName = name
		}

		// Get symbolic link
		if link, err := activate.AsAttributes().GetString(&MF_DEVSOURCE_ATTRIBUTE_SOURCE_TYPE_VIDCAP_SYMBOLIC_LINK); err == nil {
			device.SymbolicLink = link
		}

		result = append(result, device)
	}

	return result, nil
}
