//go:build windows

package uvc

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// UVCDevice represents a UVC camera device on Windows using Media Foundation
type UVCDevice struct {
	mfDevice     *MFCameraDevice
	mediaSource  *IMFMediaSource
	sourceReader *IMFSourceReader
	cameraCtrl   *IAMCameraControl
	videoProcAmp *IAMVideoProcAmp
	closed       *atomic.Bool
	mu           sync.Mutex
}

// NewUVCDevice creates a UVC device from a file descriptor.
// On Windows, this is not supported - use NewUVCDeviceByIndex or NewUVCDeviceByName instead.
func NewUVCDevice(fd uintptr) (*UVCDevice, error) {
	return nil, fmt.Errorf("NewUVCDevice(fd) is not supported on Windows; use NewUVCDeviceByIndex or EnumerateUVCDevices")
}

// EnumerateUVCDevices returns all available UVC camera devices
func EnumerateUVCDevices() ([]*MFCameraDevice, error) {
	return EnumerateMFCameras()
}

// NewUVCDeviceByIndex creates a UVC device by its index in the enumeration
func NewUVCDeviceByIndex(index int) (*UVCDevice, error) {
	devices, err := EnumerateMFCameras()
	if err != nil {
		return nil, err
	}

	if index < 0 || index >= len(devices) {
		return nil, fmt.Errorf("camera index %d out of range (0-%d)", index, len(devices)-1)
	}

	return newUVCDeviceFromMF(devices[index])
}

// NewUVCDeviceByName creates a UVC device by matching its friendly name
func NewUVCDeviceByName(name string) (*UVCDevice, error) {
	devices, err := EnumerateMFCameras()
	if err != nil {
		return nil, err
	}

	for _, dev := range devices {
		if dev.FriendlyName == name {
			return newUVCDeviceFromMF(dev)
		}
	}

	return nil, fmt.Errorf("camera '%s' not found", name)
}

func newUVCDeviceFromMF(mfDev *MFCameraDevice) (*UVCDevice, error) {
	dev := &UVCDevice{
		mfDevice: mfDev,
		closed:   &atomic.Bool{},
	}

	// Activate the media source
	sourcePtr, err := mfDev.Activate.ActivateObject(&IID_IMFMediaSource)
	if err != nil {
		return nil, fmt.Errorf("failed to activate media source: %w", err)
	}
	dev.mediaSource = (*IMFMediaSource)(unsafe.Pointer(sourcePtr))

	// Try to get camera control interface
	if ccPtr, err := dev.mediaSource.QueryInterface(&IID_IAMCameraControl); err == nil {
		dev.cameraCtrl = (*IAMCameraControl)(unsafe.Pointer(ccPtr))
	}

	// Try to get video proc amp interface
	if vpPtr, err := dev.mediaSource.QueryInterface(&IID_IAMVideoProcAmp); err == nil {
		dev.videoProcAmp = (*IAMVideoProcAmp)(unsafe.Pointer(vpPtr))
	}

	return dev, nil
}

func (d *UVCDevice) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed.Load() {
		return nil
	}
	d.closed.Store(true)

	if d.sourceReader != nil {
		d.sourceReader.Release()
		d.sourceReader = nil
	}
	if d.cameraCtrl != nil {
		d.cameraCtrl.Release()
		d.cameraCtrl = nil
	}
	if d.videoProcAmp != nil {
		d.videoProcAmp.Release()
		d.videoProcAmp = nil
	}
	if d.mediaSource != nil {
		d.mediaSource.Shutdown()
		d.mediaSource.Release()
		d.mediaSource = nil
	}

	return nil
}

// Handle returns nil on Windows as we don't use USB handles
func (d *UVCDevice) Handle() interface{} {
	return nil
}

func (d *UVCDevice) IsTISCamera() (bool, error) {
	return false, nil
}

// DeviceInfo returns device information including supported formats
func (d *UVCDevice) DeviceInfo() (*DeviceInfo, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed.Load() {
		return nil, fmt.Errorf("device is closed")
	}

	// Create source reader if not exists
	if d.sourceReader == nil {
		var reader *IMFSourceReader
		hr, _, _ := procMFCreateSourceReaderFromMediaSource.Call(
			uintptr(unsafe.Pointer(d.mediaSource)),
			0,
			uintptr(unsafe.Pointer(&reader)))
		if hr != 0 {
			return nil, fmt.Errorf("MFCreateSourceReaderFromMediaSource failed: 0x%x", hr)
		}
		d.sourceReader = reader
	}

	info := &DeviceInfo{
		device:      d,
		friendlyName: d.mfDevice.FriendlyName,
	}

	// Enumerate supported media types
	for i := uint32(0); ; i++ {
		mediaType, err := d.sourceReader.GetNativeMediaType(MF_SOURCE_READER_FIRST_VIDEO_STREAM, i)
		if err != nil {
			break // No more media types
		}

		format := &MFMediaFormat{}

		// Get subtype (format GUID)
		if subtype, err := mediaType.AsAttributes().GetGUID(&MF_MT_SUBTYPE); err == nil {
			format.Subtype = subtype
			format.FourCC = guidToFourCC(subtype)
		}

		// Get frame size
		if frameSize, err := mediaType.AsAttributes().GetUINT64(&MF_MT_FRAME_SIZE); err == nil {
			format.Width = uint32(frameSize >> 32)
			format.Height = uint32(frameSize & 0xFFFFFFFF)
		}

		// Get frame rate
		if frameRate, err := mediaType.AsAttributes().GetUINT64(&MF_MT_FRAME_RATE); err == nil {
			num := uint32(frameRate >> 32)
			den := uint32(frameRate & 0xFFFFFFFF)
			if den > 0 {
				format.FrameRate = float64(num) / float64(den)
			}
		}

		info.Formats = append(info.Formats, format)
		mediaType.Release()
	}

	return info, nil
}

// MFMediaFormat represents a Media Foundation video format
type MFMediaFormat struct {
	Subtype   windows.GUID
	FourCC    string
	Width     uint32
	Height    uint32
	FrameRate float64
}

// DeviceInfo for Windows
type DeviceInfo struct {
	device       *UVCDevice
	friendlyName string
	Formats      []*MFMediaFormat
}

func (d *DeviceInfo) FriendlyName() string {
	return d.friendlyName
}

func (d *DeviceInfo) Close() error {
	return nil
}

// StreamingInterface for Windows
type StreamingInterface struct {
	device *UVCDevice
	format *MFMediaFormat
}

// MFFrameReader reads frames using Media Foundation
type MFFrameReader struct {
	device *UVCDevice
	format *MFMediaFormat
	closed bool
	mu     sync.Mutex
}

// ClaimFrameReader creates a frame reader for the specified format
func (d *UVCDevice) ClaimFrameReader(width, height uint32, fourCC string) (*MFFrameReader, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed.Load() {
		return nil, fmt.Errorf("device is closed")
	}

	// Create source reader if not exists
	if d.sourceReader == nil {
		var reader *IMFSourceReader
		hr, _, _ := procMFCreateSourceReaderFromMediaSource.Call(
			uintptr(unsafe.Pointer(d.mediaSource)),
			0,
			uintptr(unsafe.Pointer(&reader)))
		if hr != 0 {
			return nil, fmt.Errorf("MFCreateSourceReaderFromMediaSource failed: 0x%x", hr)
		}
		d.sourceReader = reader
	}

	// Find matching format
	var selectedType *IMFMediaType
	var selectedFormat *MFMediaFormat

	for i := uint32(0); ; i++ {
		mediaType, err := d.sourceReader.GetNativeMediaType(MF_SOURCE_READER_FIRST_VIDEO_STREAM, i)
		if err != nil {
			break
		}

		format := &MFMediaFormat{}

		if subtype, err := mediaType.AsAttributes().GetGUID(&MF_MT_SUBTYPE); err == nil {
			format.Subtype = subtype
			format.FourCC = guidToFourCC(subtype)
		}

		if frameSize, err := mediaType.AsAttributes().GetUINT64(&MF_MT_FRAME_SIZE); err == nil {
			format.Width = uint32(frameSize >> 32)
			format.Height = uint32(frameSize & 0xFFFFFFFF)
		}

		// Check if this matches our requirements
		if format.Width == width && format.Height == height {
			if fourCC == "" || format.FourCC == fourCC {
				selectedType = mediaType
				selectedFormat = format
				break
			}
		}

		mediaType.Release()
	}

	if selectedType == nil {
		return nil, fmt.Errorf("no matching format found for %dx%d %s", width, height, fourCC)
	}

	// Set the selected media type
	if err := d.sourceReader.SetCurrentMediaType(MF_SOURCE_READER_FIRST_VIDEO_STREAM, selectedType); err != nil {
		selectedType.Release()
		return nil, fmt.Errorf("failed to set media type: %w", err)
	}
	selectedType.Release()

	return &MFFrameReader{
		device: d,
		format: selectedFormat,
	}, nil
}

// ReadFrame reads a single video frame
func (r *MFFrameReader) ReadFrame() ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil, io.EOF
	}

	r.device.mu.Lock()
	defer r.device.mu.Unlock()

	if r.device.closed.Load() {
		return nil, fmt.Errorf("device is closed")
	}

	_, flags, _, sample, err := r.device.sourceReader.ReadSample(MF_SOURCE_READER_FIRST_VIDEO_STREAM)
	if err != nil {
		return nil, err
	}
	if sample == nil {
		if flags&0x1 != 0 { // MF_SOURCE_READERF_ENDOFSTREAM
			return nil, io.EOF
		}
		return nil, fmt.Errorf("no sample returned, flags: 0x%x", flags)
	}
	defer sample.Release()

	// Get the buffer from the sample
	buffer, err := sample.ConvertToContiguousBuffer()
	if err != nil {
		return nil, err
	}
	defer buffer.Release()

	// Lock and copy the data
	data, err := buffer.Lock()
	if err != nil {
		return nil, err
	}
	buffer.Unlock()

	return data, nil
}

// ReadFrameImage reads a frame and decodes it to an image
func (r *MFFrameReader) ReadFrameImage() (image.Image, error) {
	data, err := r.ReadFrame()
	if err != nil {
		return nil, err
	}

	// Decode based on format
	switch r.format.FourCC {
	case "MJPG":
		return decodeJPEG(data)
	case "YUY2":
		return decodeYUY2(data, int(r.format.Width), int(r.format.Height))
	case "NV12":
		return decodeNV12(data, int(r.format.Width), int(r.format.Height))
	default:
		return nil, fmt.Errorf("unsupported format: %s", r.format.FourCC)
	}
}

func (r *MFFrameReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	return nil
}

func (r *MFFrameReader) Format() *MFMediaFormat {
	return r.format
}

// Camera control methods

// GetExposure gets the current exposure value
func (d *UVCDevice) GetExposure() (value int32, auto bool, err error) {
	if d.cameraCtrl == nil {
		return 0, false, fmt.Errorf("camera control not available")
	}
	val, flags, err := d.cameraCtrl.Get(CameraControl_Exposure)
	if err != nil {
		return 0, false, err
	}
	return val, (flags & CameraControl_Flags_Auto) != 0, nil
}

// SetExposure sets the exposure value
func (d *UVCDevice) SetExposure(value int32, auto bool) error {
	if d.cameraCtrl == nil {
		return fmt.Errorf("camera control not available")
	}
	flags := int32(CameraControl_Flags_Manual)
	if auto {
		flags = CameraControl_Flags_Auto
	}
	return d.cameraCtrl.Set(CameraControl_Exposure, value, flags)
}

// GetFocus gets the current focus value
func (d *UVCDevice) GetFocus() (value int32, auto bool, err error) {
	if d.cameraCtrl == nil {
		return 0, false, fmt.Errorf("camera control not available")
	}
	val, flags, err := d.cameraCtrl.Get(CameraControl_Focus)
	if err != nil {
		return 0, false, err
	}
	return val, (flags & CameraControl_Flags_Auto) != 0, nil
}

// SetFocus sets the focus value
func (d *UVCDevice) SetFocus(value int32, auto bool) error {
	if d.cameraCtrl == nil {
		return fmt.Errorf("camera control not available")
	}
	flags := int32(CameraControl_Flags_Manual)
	if auto {
		flags = CameraControl_Flags_Auto
	}
	return d.cameraCtrl.Set(CameraControl_Focus, value, flags)
}

// GetZoom gets the current zoom value
func (d *UVCDevice) GetZoom() (value int32, err error) {
	if d.cameraCtrl == nil {
		return 0, fmt.Errorf("camera control not available")
	}
	val, _, err := d.cameraCtrl.Get(CameraControl_Zoom)
	return val, err
}

// SetZoom sets the zoom value
func (d *UVCDevice) SetZoom(value int32) error {
	if d.cameraCtrl == nil {
		return fmt.Errorf("camera control not available")
	}
	return d.cameraCtrl.Set(CameraControl_Zoom, value, CameraControl_Flags_Manual)
}

// GetBrightness gets the current brightness value
func (d *UVCDevice) GetBrightness() (value int32, err error) {
	if d.videoProcAmp == nil {
		return 0, fmt.Errorf("video proc amp not available")
	}
	val, _, err := d.videoProcAmp.Get(VideoProcAmp_Brightness)
	return val, err
}

// SetBrightness sets the brightness value
func (d *UVCDevice) SetBrightness(value int32) error {
	if d.videoProcAmp == nil {
		return fmt.Errorf("video proc amp not available")
	}
	return d.videoProcAmp.Set(VideoProcAmp_Brightness, value, CameraControl_Flags_Manual)
}

// GetContrast gets the current contrast value
func (d *UVCDevice) GetContrast() (value int32, err error) {
	if d.videoProcAmp == nil {
		return 0, fmt.Errorf("video proc amp not available")
	}
	val, _, err := d.videoProcAmp.Get(VideoProcAmp_Contrast)
	return val, err
}

// SetContrast sets the contrast value
func (d *UVCDevice) SetContrast(value int32) error {
	if d.videoProcAmp == nil {
		return fmt.Errorf("video proc amp not available")
	}
	return d.videoProcAmp.Set(VideoProcAmp_Contrast, value, CameraControl_Flags_Manual)
}

// ControlTransfer is not supported on Windows MF backend
func (d *UVCDevice) ControlTransfer(requestType, request uint8, value, index uint16, data []byte, timeout time.Duration) (int, error) {
	return 0, fmt.Errorf("raw USB control transfers not supported on Windows; use camera control methods instead")
}

// Helper functions

func guidToFourCC(g windows.GUID) string {
	switch g {
	case MFVideoFormat_MJPG:
		return "MJPG"
	case MFVideoFormat_YUY2:
		return "YUY2"
	case MFVideoFormat_NV12:
		return "NV12"
	case MFVideoFormat_RGB24:
		return "RGB3"
	case MFVideoFormat_RGB32:
		return "RGB4"
	default:
		// Try to interpret as FourCC
		b := (*[4]byte)(unsafe.Pointer(&g.Data1))[:]
		return string(b)
	}
}

// Image decoding helpers

func decodeJPEG(data []byte) (image.Image, error) {
	return jpeg.Decode(bytes.NewReader(data))
}

func decodeYUY2(data []byte, width, height int) (image.Image, error) {
	img := image.NewYCbCr(image.Rect(0, 0, width, height), image.YCbCrSubsampleRatio422)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x += 2 {
			i := (y*width + x) * 2
			if i+3 >= len(data) {
				break
			}

			y0 := data[i]
			u := data[i+1]
			y1 := data[i+2]
			v := data[i+3]

			yi := y*img.YStride + x
			ci := y*img.CStride + x/2

			img.Y[yi] = y0
			img.Y[yi+1] = y1
			img.Cb[ci] = u
			img.Cr[ci] = v
		}
	}

	return img, nil
}

func decodeNV12(data []byte, width, height int) (image.Image, error) {
	img := image.NewYCbCr(image.Rect(0, 0, width, height), image.YCbCrSubsampleRatio420)

	ySize := width * height
	if len(data) < ySize+ySize/2 {
		return nil, fmt.Errorf("NV12 data too short")
	}

	// Copy Y plane
	copy(img.Y, data[:ySize])

	// Deinterleave UV plane
	uvData := data[ySize:]
	for i := 0; i < len(img.Cb); i++ {
		img.Cb[i] = uvData[i*2]
		img.Cr[i] = uvData[i*2+1]
	}

	return img, nil
}
