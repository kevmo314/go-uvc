package transfers

/*
#cgo LDFLAGS: -lusb-1.0
#include <libusb-1.0/libusb.h>
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"io"
	"log"
	"unsafe"

	"github.com/kevmo314/go-uvc/pkg/descriptors"
)

type FrameReader struct {
	deviceHandle *C.struct_libusb_device_handle
	usb          *C.struct_libusb_interface
	vpcc         *descriptors.VideoProbeCommitControl
	pr           PayloadReader

	// this can happen if the device does not correctly set the end of frame bit.
	bufferedPayload *Payload

	fid    *bool
	buffer []byte
}

type Frame struct {
	Payloads      []*Payload
	index, offset int
}

// Read reads the payload datas concatenated together.
func (f *Frame) Read(buf []byte) (int, error) {
	total := 0
	for _, p := range f.Payloads {
		total += len(p.Data)
	}
	n := 0
	for n < len(buf) {
		if f.index == len(f.Payloads) {
			if n == 0 {
				return 0, io.EOF
			}
			return n, nil
		}
		p := f.Payloads[f.index]
		m := copy(buf[n:], p.Data[f.offset:])
		f.offset += m
		n += m
		if f.offset >= len(p.Data) {
			f.index++
			f.offset = 0
		}
	}
	return n, nil
}

func NewFrameReader(usbctxp, deviceHandlep, usbp unsafe.Pointer, endpointAddress uint8, vpcc *descriptors.VideoProbeCommitControl) (*FrameReader, error) {
	usbctx := (*C.struct_libusb_context)(usbctxp)
	deviceHandle := (*C.struct_libusb_device_handle)(deviceHandlep)
	usb := (*C.struct_libusb_interface)(usbp)
	useIsochronous := usb.num_altsetting > 1
	if useIsochronous {
		altsetting, packetSize, err := findIsochronousAltSetting(usbctx, usb, C.uchar(endpointAddress), vpcc.MaxPayloadTransferSize)
		if err != nil {
			return nil, err
		}
		if ret := C.libusb_set_interface_alt_setting(deviceHandle, C.int(altsetting.bInterfaceNumber), C.int(altsetting.bAlternateSetting)); ret < 0 {
			return nil, fmt.Errorf("libusb_set_interface_alt_setting failed: %s", C.GoString(C.libusb_error_name(ret)))
		}
		packets := min((vpcc.MaxVideoFrameSize+packetSize-1)/packetSize, 128)
		ir, err := NewIsochronousReader(unsafe.Pointer(deviceHandle), endpointAddress, packets, packetSize)
		if err != nil {
			return nil, err
		}
		return &FrameReader{
			deviceHandle: deviceHandle,
			usb:          usb,
			vpcc:         vpcc,
			pr:           ir,
		}, nil
	} else {
		br, err := NewBulkReader(unsafe.Pointer(deviceHandle), endpointAddress, vpcc.MaxPayloadTransferSize)
		if err != nil {
			return nil, err
		}
		return &FrameReader{
			deviceHandle: deviceHandle,
			usb:          usb,
			vpcc:         vpcc,
			pr:           br,
		}, nil
	}
}

func findAltEndpoint(endpoints []C.struct_libusb_endpoint_descriptor, endpointAddress C.uchar) (int, error) {
	for i, endpoint := range endpoints {
		if endpoint.bEndpointAddress == endpointAddress {
			return i, nil
		}
	}
	return 0, fmt.Errorf("endpoint not found")
}

func getEndpointMaxPacketSize(ctx *C.struct_libusb_context, endpoint C.struct_libusb_endpoint_descriptor) uint32 {
	var ssdesc *C.struct_libusb_ss_endpoint_companion_descriptor
	ret := C.libusb_get_ss_endpoint_companion_descriptor(ctx, &endpoint, &ssdesc)
	if ret == 0 {
		defer C.libusb_free_ss_endpoint_companion_descriptor(ssdesc)
		return uint32(ssdesc.wBytesPerInterval)
	}
	val := endpoint.wMaxPacketSize & 0x07ff
	endpointType := endpoint.bmAttributes & 0x03
	if endpointType == C.LIBUSB_TRANSFER_TYPE_ISOCHRONOUS || endpointType == C.LIBUSB_TRANSFER_TYPE_INTERRUPT {
		val *= 1 + ((val >> 1) & 3)
	}
	return uint32(val)

}

// findIsochronousAltSetting sets the isochronous alternate setting for the given interface and endpoint address to the
// first alternate setting that has a max packet size of at least mtu.
//
// UVC spec 1.5, section 2.4.3: A typical use of alternate settings is to provide a way to change the bandwidth requirements an active
// isochronous pipe imposes on the USB.
func findIsochronousAltSetting(ctx *C.struct_libusb_context, iface *C.struct_libusb_interface, endpointAddress C.uchar, payloadSize uint32) (*C.struct_libusb_interface_descriptor, uint32, error) {
	altsettings := (*[1 << 30]C.struct_libusb_interface_descriptor)(unsafe.Pointer(iface.altsetting))[:iface.num_altsetting]
	for i, altsetting := range altsettings {
		if altsetting.bNumEndpoints == 0 {
			// UVC spec 1.5, section 2.4.3: All devices that transfer isochronous video data must
			// incorporate a zero-bandwidth alternate setting for each VideoStreaming interface that has an
			// isochronous video endpoint, and it must be the default alternate setting (alternate setting zero).
			//
			// in other words, if there aren't any endpoints on this alternate setting it's reserved for a zero-bandwidth
			// alternate setting so we can't use it and should skip it.
			continue
		}
		endpoints := (*[1 << 30]C.struct_libusb_endpoint_descriptor)(unsafe.Pointer(altsetting.endpoint))[:altsetting.bNumEndpoints]

		j, err := findAltEndpoint(endpoints, endpointAddress)
		if err != nil {
			return nil, 0, err
		}

		packetSize := getEndpointMaxPacketSize(ctx, endpoints[j])
		if packetSize >= payloadSize || i == len(altsettings)-1 {
			return &altsetting, packetSize, nil
		}
	}
	panic("invalid state")
}

// ReadFrame reads individual payloads from the USB device and returns a constructed frame.
func (r *FrameReader) ReadFrame() (*Frame, error) {
	var f *Frame
	for {
		var p *Payload
		if r.bufferedPayload != nil {
			p = r.bufferedPayload
			r.bufferedPayload = nil
		} else {
			q, err := r.pr.ReadPayload()
			if err != nil {
				return nil, err
			}
			p = q
		}
		log.Printf("payload: %s", p.String())
		if r.fid == nil || p.FrameID() != *r.fid {
			// frame id bit flipped, this is a new frame
			if f != nil {
				r.bufferedPayload = p
				return f, nil
			}
			f = &Frame{}
			fid := p.FrameID()
			r.fid = &fid
		}
		if f == nil {
			continue
		}
		f.Payloads = append(f.Payloads, p)
		if p.EndOfFrame() {
			return f, nil
		}
	}
}

func (r *FrameReader) Close() error {
	if ret := C.libusb_release_interface(r.deviceHandle, C.int(r.usb.altsetting.bInterfaceNumber)); ret < 0 {
		return fmt.Errorf("libusb_release_interface failed: %s", C.GoString(C.libusb_error_name(ret)))
	}
	return nil
}
