package transfers

import (
	"fmt"
	"io"

	usb "github.com/kevmo314/go-usb"
	"github.com/kevmo314/go-uvc/pkg/descriptors"
)

type FrameReader struct {
	handle *usb.DeviceHandle
	iface  *usb.Interface
	vpcc   *descriptors.VideoProbeCommitControl
	pr     io.Reader

	fid         *bool
	buffer      []byte
	size, patch int
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

func (si *StreamingInterface) NewFrameReader(endpointAddress uint8, vpcc *descriptors.VideoProbeCommitControl) (*FrameReader, error) {
	useIsochronous := len(si.iface.AltSettings) > 1
	if useIsochronous {
		altsetting, packetSize, err := findIsochronousAltSetting(si.iface, endpointAddress, vpcc.MaxPayloadTransferSize)
		if err != nil {
			return nil, err
		}
		if err := si.handle.SetInterfaceAltSetting(altsetting.InterfaceNumber, altsetting.AlternateSetting); err != nil {
			return nil, fmt.Errorf("set_interface_alt_setting failed: %w", err)
		}
		packets := min((vpcc.MaxVideoFrameSize+packetSize-1)/packetSize, 128)
		ir, err := si.NewIsochronousReader(endpointAddress, packets, packetSize)
		if err != nil {
			return nil, err
		}
		return &FrameReader{
			handle: si.handle,
			iface:  si.iface,
			vpcc:   vpcc,
			pr:     ir,
			buffer: make([]byte, vpcc.MaxVideoFrameSize),
		}, nil
	} else {
		// Use async bulk reader for better throughput with queued URBs
		br, err := si.NewAsyncBulkReader(endpointAddress, vpcc.MaxPayloadTransferSize)
		if err != nil {
			return nil, err
		}
		return &FrameReader{
			handle: si.handle,
			iface:  si.iface,
			vpcc:   vpcc,
			pr:     br,
			buffer: make([]byte, vpcc.MaxVideoFrameSize),
		}, nil
	}
}

func findAltEndpoint(endpoints []usb.Endpoint, endpointAddress uint8) (int, error) {
	for i, endpoint := range endpoints {
		if endpoint.EndpointAddr == endpointAddress {
			return i, nil
		}
	}
	return 0, fmt.Errorf("endpoint not found")
}

func getEndpointMaxPacketSize(endpoint usb.Endpoint) uint32 {
	// For SuperSpeed devices, check companion descriptor
	if endpoint.SSCompanion != nil {
		return uint32(endpoint.SSCompanion.BytesPerInterval)
	}
	val := uint32(endpoint.MaxPacketSize & 0x07ff)
	endpointType := usb.TransferType(endpoint.Attributes & 0x03)
	if endpointType == usb.TransferTypeIsochronous || endpointType == usb.TransferTypeInterrupt {
		val *= 1 + ((val >> 1) & 3)
	}
	return val
}

// findIsochronousAltSetting sets the isochronous alternate setting for the given interface and endpoint address to the
// first alternate setting that has a max packet size of at least mtu.
//
// UVC spec 1.5, section 2.4.3: A typical use of alternate settings is to provide a way to change the bandwidth requirements an active
// isochronous pipe imposes on the USB.
func findIsochronousAltSetting(iface *usb.Interface, endpointAddress uint8, payloadSize uint32) (*usb.InterfaceAltSetting, uint32, error) {
	for i, altsetting := range iface.AltSettings {
		if altsetting.NumEndpoints == 0 {
			// UVC spec 1.5, section 2.4.3: All devices that transfer isochronous video data must
			// incorporate a zero-bandwidth alternate setting for each VideoStreaming interface that has an
			// isochronous video endpoint, and it must be the default alternate setting (alternate setting zero).
			//
			// in other words, if there aren't any endpoints on this alternate setting it's reserved for a zero-bandwidth
			// alternate setting so we can't use it and should skip it.
			continue
		}

		j, err := findAltEndpoint(altsetting.Endpoints, endpointAddress)
		if err != nil {
			return nil, 0, err
		}

		packetSize := getEndpointMaxPacketSize(altsetting.Endpoints[j])
		if packetSize >= payloadSize || i == len(iface.AltSettings)-1 {
			return &altsetting, packetSize, nil
		}
	}
	return nil, 0, fmt.Errorf("no suitable isochronous alternate setting found for payload size %d", payloadSize)
}

// ReadFrame reads individual payloads from the USB device and returns a constructed frame.
func (r *FrameReader) ReadFrame() (*Frame, error) {
	var f *Frame
	for {
		p := &Payload{}
		n := 0
		if r.patch == 0 {
			m, err := r.pr.Read(r.buffer[r.size:])
			if err != nil {
				return nil, err
			}
			n = m
			if err := p.UnmarshalBinary(r.buffer[r.size : r.size+n]); err != nil {
				return nil, err
			}
		} else {
			if copy(r.buffer, r.buffer[r.size:r.size+r.patch]) != r.patch {
				return nil, fmt.Errorf("copy failed")
			}
			r.size = r.patch
			n = r.patch
			r.patch = 0
			if err := p.UnmarshalBinary(r.buffer[:r.size]); err != nil {
				return nil, err
			}
		}
		if r.fid == nil || p.FrameID() != *r.fid {
			// frame id bit flipped, this is a new frame
			if f != nil {
				// set the patch to the size of the payload to indicate that
				// the next payload should read from the existing buffer.
				r.patch = n
				return f, nil
			}
			f = &Frame{}
			fid := p.FrameID()
			r.fid = &fid
		}
		if f == nil {
			// if there's no frame, ignore this payload.
			// this can happen if the device sends frames after an end of frame bit.
			continue
		}
		r.size += n
		f.Payloads = append(f.Payloads, p)
		if p.EndOfFrame() {
			// reset the buffer
			r.size = 0
			return f, nil
		}
	}
}

func (r *FrameReader) Close() error {
	if len(r.iface.AltSettings) == 0 {
		return nil
	}
	ifnum := r.iface.AltSettings[0].InterfaceNumber
	return r.handle.ReleaseInterface(ifnum)
}
