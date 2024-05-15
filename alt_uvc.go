package uvc

/*
#cgo LDFLAGS: -lusb-1.0
#include <libusb-1.0/libusb.h>
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"log"

	"github.com/google/gousb"
	"github.com/kevmo314/go-uvc/pkg/descriptors"
	"github.com/kevmo314/go-uvc/pkg/requests"
)

type UVCDevice1 struct {
	device *gousb.Device
}

func NewUVCDevice1(fd uintptr) (*UVCDevice1, error) {
	ctx := gousb.NewContext()

	device, err := ctx.OpenDeviceWithFileDescriptor(uintptr(fd))
	if err != nil {
		return nil, err
	}
	return &UVCDevice1{device: device}, nil
}

func (d *UVCDevice1) IsTISCamera1() (bool, error) {
	desc := d.device.Desc
	return desc.Vendor.String() == "0x199e" && (desc.Product.String() == "0x8101" || desc.Product.String() == "0x8102"), nil
}

type DeviceInfo1 struct {
	bcdUVC              uint16               // cached since it's used a lot
	videoInterface      *gousb.InterfaceDesc // cached since it's used a lot
	controlInterface    []descriptors.ControlInterface
	StreamingInterfaces []*StreamingInterface1
}

func (d *UVCDevice1) GetDeviceInfo() (*DeviceInfo1, error) {
	configDesc := d.device.Desc.Configs[1]
	// scan control interfaces
	isTISCamera, err := d.IsTISCamera1()
	if err != nil {
		return nil, err
	}
	ifaceIdx := -1

	for i, iface := range configDesc.Interfaces {
		if isTISCamera && iface.AltSettings[0].Class == gousb.ClassVendorSpec && iface.AltSettings[0].SubClass == gousb.ClassAudio {
			ifaceIdx = i
			break
		} else if !isTISCamera && iface.AltSettings[0].Class == gousb.ClassVideo && iface.AltSettings[0].SubClass == gousb.ClassAudio {
			ifaceIdx = i
			break
		}
	}
	if ifaceIdx == -1 {
		return nil, fmt.Errorf("control interface not found")
	}
	info := &DeviceInfo1{}
	info.videoInterface = &configDesc.Interfaces[ifaceIdx]

	vcbuf := info.videoInterface.AltSettings[0].Extra
	for i := 0; i != len(vcbuf); i += int(vcbuf[i]) {
		block := vcbuf[i : i+int(vcbuf[i])]
		if block[1] != 0x24 {
			// ignore blocks that are not CS_INTERFACE 0x24
			continue
		}
		ci, err := descriptors.UnmarshalControlInterface(block)
		if err != nil {
			return nil, err
		}

		info.controlInterface = append(info.controlInterface, ci)
		switch ci := ci.(type) {
		case *descriptors.HeaderDescriptor:
			info.bcdUVC = ci.UVC
			// pull the streaming interfaces too
			for _, i := range ci.VideoStreamingInterfaceIndexes {
				vsbuf := configDesc.Interfaces[i].AltSettings[0].Extra
				asi := &StreamingInterface1{usb: &configDesc.Interfaces[i], bcdUVC: info.bcdUVC}
				for j := 0; j != len(vsbuf); j += int(vsbuf[j]) {
					block := vsbuf[j : j+int(vsbuf[j])]
					si, err := descriptors.UnmarshalStreamingInterface(block)
					if err != nil {
						return nil, err
					}
					asi.Descriptors = append(asi.Descriptors, si)
				}
				info.StreamingInterfaces = append(info.StreamingInterfaces, asi)
				log.Printf("got streaming interface InterfaceNumber %d %d", i, configDesc.Interfaces[i].AltSettings[0].Number)
			}
		}
	}

	return info, nil
}

type FrameReader1 struct {
	si     *StreamingInterface1
	config *descriptors.VideoProbeCommitControl
}

type StreamingInterface1 struct {
	bcdUVC      uint16 // cached since it's used a lot
	usb         *gousb.InterfaceDesc
	Descriptors []descriptors.StreamingInterface
}

// detach_kernel_driver for the specific interface
// claim_interface for the specific interface
// control_transfer
func (si *StreamingInterface1) ClaimDeviceFrameReader(uvc *UVCDevice1, bcdUVC uint16, formatIndex, frameIndex uint8) (*FrameReader1, error) {
	// gousb doesnt seem to expose the regular detach
	if err := uvc.device.SetAutoDetach(true); err != nil {
		return nil, err
	}

	ifnum := si.usb.AltSettings[0].Number

	// How to claim
	cfgNum, err := uvc.device.ActiveConfigNum()
	if err != nil {
		return nil, err
	}
	cfg, err := uvc.device.Config(cfgNum)
	if err != nil {
		return nil, err
	}

	_, err = cfg.Interface(ifnum, 0)
	if err != nil {
		return nil, err
	}

	// Control
	vpcc := &descriptors.VideoProbeCommitControl{}
	buf := make([]byte, vpcc.MarshalSize(bcdUVC))
	_, err = uvc.device.Control(
		uint8(requests.RequestTypeVideoInterfaceGetRequest),
		uint8(requests.RequestCodeGetMax),
		1<<8,
		uint16(ifnum),
		buf,
	)
	if err != nil {
		return nil, err
	}

	// assign the values
	if err := vpcc.UnmarshalBinary(buf); err != nil {
		return nil, err
	}

	log.Printf("%#v", vpcc)

	// Set
	_, err = uvc.device.Control(
		uint8(requests.RequestTypeVideoInterfaceSetRequest), /* bmRequestType */
		uint8(requests.RequestCodeSetCur),                   /* bRequest */
		1<<8,                                                /* wValue */
		uint16(ifnum),                                       /* wIndex */
		buf,
	)
	if err != nil {
		return nil, err
	}

	//Get
	_, err = uvc.device.Control(
		uint8(requests.RequestTypeVideoInterfaceGetRequest), /* bmRequestType */
		uint8(requests.RequestCodeGetCur),                   /* bRequest */
		1<<8,                                                /* wValue */
		uint16(ifnum),                                       /* wIndex */
		buf,                                                 /* data */

	)
	if err != nil {
		return nil, err
	}
	// Set
	_, err = uvc.device.Control(
		uint8(requests.RequestTypeVideoInterfaceSetRequest), /* bmRequestType */
		uint8(requests.RequestCodeSetCur),                   /* bRequest */
		2<<8,                                                /* wValue */
		uint16(ifnum),                                       /* wIndex */
		buf,                                                 /* data */

	)
	if err != nil {
		return nil, err
	}
	// unmarshal the negotiated values
	return &FrameReader1{
		si:     si,
		config: vpcc,
	}, vpcc.UnmarshalBinary(buf)

}
