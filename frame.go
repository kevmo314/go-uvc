package uvc

/*
#cgo LDFLAGS: -lusb-1.0
#include <libusb-1.0/libusb.h>
#include <stdlib.h>
*/
import "C"
import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"unsafe"

	"github.com/kevmo314/go-uvc/pkg/descriptors"
	"github.com/kevmo314/go-uvc/pkg/transfers"
)

type FrameReader struct {
	si     *StreamingInterface
	config *descriptors.VideoProbeCommitControl
	reader io.Reader
	buf    []byte
}

type Frame struct {
	HeaderInfoBitmask uint8
	PTS               uint32
	SCR               struct {
		SourceTimeClock uint32
		TokenCounter    uint16
	}
	Data []byte
}

func (f *Frame) FrameID() bool {
	return f.HeaderInfoBitmask&0b00000001 != 0
}

func (f *Frame) EndOfFrame() bool {
	return f.HeaderInfoBitmask&0b00000010 != 0
}

func (f *Frame) HasPTS() bool {
	return f.HeaderInfoBitmask&0b00000100 != 0
}

func (f *Frame) HasSCR() bool {
	return f.HeaderInfoBitmask&0b00001000 != 0
}

func (f *Frame) PayloadSpecificBit() bool {
	return f.HeaderInfoBitmask&0b00010000 != 0
}

func (f *Frame) StillImage() bool {
	return f.HeaderInfoBitmask&0b00100000 != 0
}

func (f *Frame) Error() bool {
	return f.HeaderInfoBitmask&0b01000000 != 0
}

func (f *Frame) EndOfHeader() bool {
	return f.HeaderInfoBitmask&0b10000000 != 0
}

func (f *Frame) UnmarshalBinary(buf []byte) error {
	if len(buf) < int(buf[0]) {
		return io.ErrShortBuffer
	}
	f.HeaderInfoBitmask = buf[1]
	offset := 2
	if f.HasPTS() {
		f.PTS = binary.LittleEndian.Uint32(buf[offset : offset+4])
		offset += 4
	}
	if f.HasSCR() {
		f.SCR.SourceTimeClock = binary.LittleEndian.Uint32(buf[offset : offset+4])
		offset += 4
		f.SCR.TokenCounter = binary.LittleEndian.Uint16(buf[offset : offset+2])
		offset += 2
	}
	f.Data = buf[offset:]
	return nil
}

func NewFrameReader(si *StreamingInterface, config *descriptors.VideoProbeCommitControl) (*FrameReader, error) {
	useIsochronous := si.usb.num_altsetting > 1
	if useIsochronous {
		panic("not yet implemented")
	} else {
		inputs := si.InputHeaderDescriptors()
		if len(inputs) == 0 {
			return nil, fmt.Errorf("no input header descriptors found")
		}

		log.Printf("got input header descriptors: %#v", inputs[0].EndpointAddress)
		br, err := transfers.NewBulkReader(unsafe.Pointer(si.deviceHandle), inputs[0].EndpointAddress, config.MaxPayloadTransferSize)
		if err != nil {
			return nil, err
		}
		return &FrameReader{
			si:     si,
			config: config,
			reader: br,
			buf:    make([]byte, config.MaxPayloadTransferSize),
		}, nil
	}
}

func (r *FrameReader) ReadFrame() (*Frame, error) {
	n, err := r.reader.Read(r.buf)
	if err != nil {
		return nil, err
	}
	f := &Frame{}
	return f, f.UnmarshalBinary(r.buf[:n])
	// size := r.config.MaxPayloadTransferSize
	// buf := C.malloc(C.ulong(size))
	// defer C.free(buf)

	// inputs := r.si.InputHeaderDescriptors()
	// if len(inputs) == 0 {
	// 	return nil, fmt.Errorf("no input header descriptors found")
	// }

	// log.Printf("got input header descriptors: %#v", inputs[0].EndpointAddress)

	// log.Printf("num altsetting %d", (r.si.usb.num_altsetting))

	// if r.si.usb.num_altsetting > 1 {
	// 	// configure isochronous transfer
	// 	altsettings := (*[1 << 30]C.struct_libusb_interface_descriptor)(unsafe.Pointer(r.si.usb.altsetting))[:r.si.usb.num_altsetting]
	// 	for _, altsetting := range altsettings {
	// 		log.Printf("altsettin iface %d altsetting %d num endpoints %d", altsetting.bInterfaceNumber, altsetting.bAlternateSetting, altsetting.bNumEndpoints)
	// 		if altsetting.bNumEndpoints == 0 {
	// 			continue
	// 		}
	// 		endpoints := (*[1 << 30]C.struct_libusb_endpoint_descriptor)(unsafe.Pointer(altsetting.endpoint))[:altsetting.bNumEndpoints]
	// 		var bpp uint16
	// 		for _, endpoint := range endpoints {
	// 			// if endpoint.bmAttributes&0b11 != C.LIBUSB_TRANSFER_TYPE_ISOCHRONOUS || endpoint.bEndpointAddress&0b1100 != C.LIBUSB_ISO_SYNC_TYPE_ASYNC {
	// 			// 	log.Printf("skipping endpoint %d", endpoint.bEndpointAddress)
	// 			// 	continue
	// 			// }
	// 			var ssdesc *C.struct_libusb_ss_endpoint_companion_descriptor
	// 			if ret := C.libusb_get_ss_endpoint_companion_descriptor(nil, &endpoint, &ssdesc); ret == 0 {
	// 				bpp = uint16(ssdesc.wBytesPerInterval)
	// 				C.libusb_free_ss_endpoint_companion_descriptor(ssdesc)
	// 				break
	// 			} else {
	// 				// usb 2.0 endpoint
	// 				if endpoint.bEndpointAddress != C.uchar(inputs[0].EndpointAddress) {
	// 					continue
	// 				}
	// 				bpp = uint16(endpoint.wMaxPacketSize)
	// 				break
	// 			}
	// 		}
	// 		if uint32(bpp) >= size {
	// 			log.Printf("setting altsetting %d to %d", altsetting.bAlternateSetting, bpp)
	// 			if ret := C.libusb_set_interface_alt_setting(r.si.deviceHandle, C.int(altsetting.bInterfaceNumber), C.int(altsetting.bAlternateSetting)); ret < 0 {
	// 				return nil, fmt.Errorf("libusb_set_interface_alt_setting failed: %w", libusberror(ret))
	// 			}
	// 			break
	// 		}
	// 	}
	// }

	// var n C.int
	// if ret := C.libusb_bulk_transfer(
	// 	r.si.deviceHandle,
	// 	C.uchar(inputs[0].EndpointAddress), /* endpoint */
	// 	(*C.uchar)(buf),                    /* data */
	// 	C.int(size),                        /* length */
	// 	&n,                                 /* transferred */
	// 	0,                                  /* timeout */
	// ); ret < 0 {
	// 	return nil, fmt.Errorf("libusb_bulk_transfer failed: %w", libusberror(ret))
	// }

	// log.Printf("address: %d size: %d read: %d", inputs[0].EndpointAddress, size, n)
	// f := &Frame{}
	// return f, f.UnmarshalBinary(C.GoBytes(unsafe.Pointer(buf), C.int(n)))
}

func (r *FrameReader) Close() error {
	if ret := C.libusb_release_interface(r.si.deviceHandle, C.int(r.si.usb.altsetting.bInterfaceNumber)); ret < 0 {
		return fmt.Errorf("libusb_release_interface failed: %w", libusberror(ret))
	}
	return nil
}
