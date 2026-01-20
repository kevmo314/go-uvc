//go:build !windows

package uvc

import (
	"fmt"
	"sync/atomic"
	"time"

	usb "github.com/kevmo314/go-usb"
	"github.com/kevmo314/go-uvc/pkg/descriptors"
	"github.com/kevmo314/go-uvc/pkg/transfers"
)

type UVCDevice struct {
	handle *usb.DeviceHandle
	closed *atomic.Bool
}

func (d *UVCDevice) Handle() *usb.DeviceHandle {
	return d.handle
}

func (d *UVCDevice) IsTISCamera() (bool, error) {
	desc := d.handle.Descriptor()
	return desc.VendorID == 0x199e && (desc.ProductID == 0x8101 || desc.ProductID == 0x8102), nil
}

func (d *UVCDevice) Close() error {
	d.closed.Store(true)
	return d.handle.Close()
}

type ControlInterface struct {
	CameraTerminal *CameraTerminal
	ProcessingUnit *ProcessingUnit
	Descriptor     descriptors.ControlInterface
}

type DeviceInfo struct {
	bcdUVC              uint16 // cached since it's used a lot
	handle              *usb.DeviceHandle
	configDesc          *usb.ConfigDescriptor
	ControlInterfaces   []*ControlInterface
	StreamingInterfaces []*transfers.StreamingInterface
}

func (d *UVCDevice) DeviceInfo() (*DeviceInfo, error) {
	configDesc, err := d.handle.ConfigDescriptorByValue(0)
	if err != nil {
		return nil, fmt.Errorf("failed to get config descriptor: %w", err)
	}

	// scan control interfaces
	isTISCamera, err := d.IsTISCamera()
	if err != nil {
		return nil, err
	}

	var controlIfaceIdx int = -1
	for i, iface := range configDesc.Interfaces {
		if len(iface.AltSettings) == 0 {
			continue
		}
		alt := iface.AltSettings[0]
		if isTISCamera && alt.InterfaceClass == 255 && alt.InterfaceSubClass == 1 {
			controlIfaceIdx = i
			break
		} else if !isTISCamera && alt.InterfaceClass == 14 && alt.InterfaceSubClass == 1 {
			controlIfaceIdx = i
			break
		}
	}
	if controlIfaceIdx == -1 {
		return nil, fmt.Errorf("control interface not found")
	}

	info := &DeviceInfo{handle: d.handle, configDesc: configDesc}

	videoInterface := &configDesc.Interfaces[controlIfaceIdx]
	if len(videoInterface.AltSettings) == 0 {
		return nil, fmt.Errorf("no alt settings for control interface")
	}
	vcbuf := videoInterface.AltSettings[0].Extra

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
		switch ci := ci.(type) {
		case *descriptors.ProcessingUnitDescriptor:
			processingUnit := &ProcessingUnit{
				handle:         d.handle,
				ifaceNum:       videoInterface.AltSettings[0].InterfaceNumber,
				UnitDescriptor: ci,
			}
			info.ControlInterfaces = append(info.ControlInterfaces, &ControlInterface{ProcessingUnit: processingUnit, Descriptor: ci})
		case *descriptors.InputTerminalDescriptor:
			it, err := descriptors.UnmarshalInputTerminal(block)
			if err != nil {
				return nil, err
			}

			switch descriptor := it.(type) {
			case *descriptors.CameraTerminalDescriptor:
				camera := &CameraTerminal{
					handle:           d.handle,
					ifaceNum:         videoInterface.AltSettings[0].InterfaceNumber,
					CameraDescriptor: descriptor,
				}
				info.ControlInterfaces = append(info.ControlInterfaces, &ControlInterface{CameraTerminal: camera, Descriptor: descriptor})
			}
		case *descriptors.HeaderDescriptor:
			info.bcdUVC = ci.UVC
			// pull the streaming interfaces too
			for _, ifaceIdx := range ci.VideoStreamingInterfaceIndexes {
				if int(ifaceIdx) >= len(configDesc.Interfaces) {
					continue
				}
				streamIface := &configDesc.Interfaces[ifaceIdx]
				if len(streamIface.AltSettings) == 0 {
					continue
				}
				vsbuf := streamIface.AltSettings[0].Extra
				asi := transfers.NewStreamingInterface(d.handle, streamIface, ci.UVC)
				for j := 0; j != len(vsbuf); j += int(vsbuf[j]) {
					block := vsbuf[j : j+int(vsbuf[j])]
					// Only parse CS_INTERFACE (0x24) descriptors
					if block[1] != 0x24 {
						continue
					}
					si, err := descriptors.UnmarshalStreamingInterface(block)
					if err != nil {
						return nil, err
					}
					asi.Descriptors = append(asi.Descriptors, si)
				}
				info.StreamingInterfaces = append(info.StreamingInterfaces, asi)
			}
		default:
			// This is an interface that we have not yet parsed
			info.ControlInterfaces = append(info.ControlInterfaces, &ControlInterface{Descriptor: ci})
		}
	}

	return info, nil
}

func (d *DeviceInfo) Close() error {
	return nil
}

func (d *UVCDevice) ControlTransfer(requestType, request uint8, value, index uint16, data []byte, timeout time.Duration) (int, error) {
	return d.handle.ControlTransfer(requestType, request, value, index, data, timeout)
}
