//go:build !windows

package uvc

import (
	"fmt"
	"sync/atomic"

	usb "github.com/kevmo314/go-usb"
	"github.com/kevmo314/go-uvc/pkg/descriptors"
	"github.com/kevmo314/go-uvc/pkg/transfers"
)

type UACDevice struct {
	handle *usb.DeviceHandle
	closed *atomic.Bool
}

func (d *UACDevice) Close() error {
	d.closed.Store(true)
	return d.handle.Close()
}

type AudioControlInterface struct {
	Descriptor descriptors.AudioControlInterface
}

type AudioDeviceInfo struct {
	bcdADC              uint16
	handle              *usb.DeviceHandle
	configDesc          *usb.ConfigDescriptor
	ControlInterfaces   []*AudioControlInterface
	StreamingInterfaces []*transfers.AudioStreamingInterface
	MIDIInterfaces      []*transfers.MIDIStreamingInterface
}

func (info *AudioDeviceInfo) GetHandle() *usb.DeviceHandle {
	return info.handle
}

func (d *UACDevice) DeviceInfo() (*AudioDeviceInfo, error) {
	configDesc, err := d.handle.ConfigDescriptorByValue(0)
	if err != nil {
		return nil, fmt.Errorf("failed to get config descriptor: %w", err)
	}

	// scan audio control interfaces
	ifaceIdx := -1
	for i, iface := range configDesc.Interfaces {
		if len(iface.AltSettings) == 0 {
			continue
		}
		// UAC uses class 1 (Audio) and subclass 1 (Control)
		if iface.AltSettings[0].InterfaceClass == 1 && iface.AltSettings[0].InterfaceSubClass == 1 {
			ifaceIdx = i
			break
		}
	}
	if ifaceIdx == -1 {
		return nil, fmt.Errorf("audio control interface not found")
	}
	info := &AudioDeviceInfo{handle: d.handle, configDesc: configDesc}

	audioInterface := &configDesc.Interfaces[ifaceIdx]
	if len(audioInterface.AltSettings) == 0 {
		return nil, fmt.Errorf("no alt settings for audio control interface")
	}
	acbuf := audioInterface.AltSettings[0].Extra

	// Parse audio control interface descriptors
	for i := 0; i != len(acbuf); i += int(acbuf[i]) {
		block := acbuf[i : i+int(acbuf[i])]
		if len(block) < 3 {
			continue
		}

		// Check for audio class-specific interface descriptors (0x24)
		if block[1] == 0x24 {
			subtype := block[2]
			switch subtype {
			case 0x01: // HEADER
				if len(block) >= 9 {
					info.bcdADC = uint16(block[3]) | (uint16(block[4]) << 8)
				}
			}
		}
	}

	// Find and parse audio streaming and MIDI interfaces
	for _, iface := range configDesc.Interfaces {
		if len(iface.AltSettings) == 0 {
			continue
		}
		// Check for audio class
		if iface.AltSettings[0].InterfaceClass == 1 {
			// Check subclass
			if iface.AltSettings[0].InterfaceSubClass == 2 {
				// Audio Streaming interface
				// Check all alternate settings for this interface
				for _, altsetting := range iface.AltSettings {
					// Skip settings with no endpoints (zero-bandwidth)
					if altsetting.NumEndpoints == 0 {
						continue
					}

					streamingIface := transfers.NewAudioStreamingInterface(
						d.handle,
						&iface,
						info.bcdADC,
					)

					// Parse streaming interface descriptors
					asbuf := altsetting.Extra
					for j := 0; j != len(asbuf); j += int(asbuf[j]) {
						block := asbuf[j : j+int(asbuf[j])]
						if len(block) >= 3 && block[1] == 0x24 {
							// Parse audio streaming descriptors
							streamingIface.ParseDescriptor(block)
						}
					}

					// Parse endpoint descriptor if available
					if len(altsetting.Endpoints) > 0 {
						streamingIface.ParseEndpointFromUSB(&altsetting.Endpoints[0])
					}

					// Only add interfaces with valid audio data
					if streamingIface.NrChannels > 0 && len(streamingIface.SamplingFreqs) > 0 {
						info.StreamingInterfaces = append(info.StreamingInterfaces, streamingIface)
					}
				}
			} else if iface.AltSettings[0].InterfaceSubClass == 3 {
				// MIDI Streaming interface
				midiIface := transfers.NewMIDIStreamingInterface(
					d.handle,
					&iface,
				)

				// Parse MIDI descriptors
				midibuf := iface.AltSettings[0].Extra
				for j := 0; j != len(midibuf); j += int(midibuf[j]) {
					block := midibuf[j : j+int(midibuf[j])]
					if len(block) >= 3 && block[1] == 0x24 {
						// Parse MIDI streaming descriptors
						midiIface.ParseDescriptor(block)
					} else if len(block) >= 3 && block[1] == 0x25 {
						// Parse class-specific endpoint descriptor
						midiIface.ParseMIDIEndpoint(block)
					}
				}

				// Parse endpoints
				for _, ep := range iface.AltSettings[0].Endpoints {
					midiIface.ParseEndpointFromUSB(&ep)
				}

				// Add MIDI interface if it has jacks
				if midiIface.NumInJacks > 0 || midiIface.NumOutJacks > 0 {
					info.MIDIInterfaces = append(info.MIDIInterfaces, midiIface)
				}
			}
		}
	}

	return info, nil
}
