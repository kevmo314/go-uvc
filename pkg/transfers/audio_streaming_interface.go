package transfers

import (
	"fmt"
	"time"

	usb "github.com/kevmo314/go-usb"
	"github.com/kevmo314/go-uvc/pkg/descriptors"
)

type AudioStreamingInterfaceControlSelector int

const (
	AudioStreamingInterfaceControlSelectorUndefined    AudioStreamingInterfaceControlSelector = 0x00
	AudioStreamingInterfaceControlSelectorSamplingFreq                                        = 0x01
	AudioStreamingInterfaceControlSelectorPitch                                               = 0x02
)

type AudioFormat struct {
	Channels      int
	SampleRate    uint32
	BitsPerSample int
	FormatType    uint8  // 1=Type I (PCM), 2=Type II (dynamic), 3=Type III (specific)
	FormatTag     uint16 // Format tag (PCM=1, MPEG=0x1001, AC3=0x1002, etc.)
}

type AudioStreamingInterface struct {
	bcdADC      uint16 // Audio Device Class version
	handle      *usb.DeviceHandle
	iface       *usb.Interface
	Descriptors []descriptors.AudioStreamingDescriptor

	// Audio specific fields
	TerminalLink    uint8
	Delay           uint8
	FormatType      uint8
	FormatTag       uint16 // Format tag from AS_GENERAL descriptor
	NrChannels      uint8
	SubframeSize    uint8
	BitResolution   uint8
	SamplingFreqs   []uint32
	EndpointAddress uint8
	MaxPacketSize   uint16

	// Format Type II specific (for MPEG, AAC, etc.)
	MaxBitRate      uint16
	SamplesPerFrame uint16

	// Format Type III specific
	FormatSpecific []byte
}

func NewAudioStreamingInterface(handle *usb.DeviceHandle, iface *usb.Interface, bcdADC uint16) *AudioStreamingInterface {
	return &AudioStreamingInterface{handle: handle, iface: iface, bcdADC: bcdADC}
}

func (asi *AudioStreamingInterface) InterfaceNumber() uint8 {
	if len(asi.iface.AltSettings) == 0 {
		return 0
	}
	return asi.iface.AltSettings[0].InterfaceNumber
}

func (asi *AudioStreamingInterface) AlternateSetting() uint8 {
	if len(asi.iface.AltSettings) == 0 {
		return 0
	}
	return asi.iface.AltSettings[0].AlternateSetting
}

func (asi *AudioStreamingInterface) ParseDescriptor(block []byte) error {
	if len(block) < 3 {
		return fmt.Errorf("descriptor too short")
	}

	subtype := block[2]
	switch subtype {
	case 0x01: // AS_GENERAL
		if len(block) >= 7 {
			asi.TerminalLink = block[3]
			asi.Delay = block[4]
			// Format tag is at block[5:6] for UAC1
			asi.FormatTag = uint16(block[5]) | (uint16(block[6]) << 8)
		}
	case 0x02: // FORMAT_TYPE
		if len(block) >= 4 {
			asi.FormatType = block[3]

			switch asi.FormatType {
			case 0x01: // FORMAT_TYPE_I (PCM, compressed)
				if len(block) >= 8 {
					asi.NrChannels = block[4]
					asi.SubframeSize = block[5]
					asi.BitResolution = block[6]
					samplingFreqType := block[7]

					// Parse sampling frequencies based on type
					if samplingFreqType == 0 {
						// Continuous sampling frequency range
						if len(block) >= 14 {
							minFreq := uint32(block[8]) | (uint32(block[9]) << 8) | (uint32(block[10]) << 16)
							maxFreq := uint32(block[11]) | (uint32(block[12]) << 8) | (uint32(block[13]) << 16)
							// Add common frequencies within range
							commonFreqs := []uint32{8000, 16000, 24000, 32000, 44100, 48000, 96000, 192000}
							for _, freq := range commonFreqs {
								if freq >= minFreq && freq <= maxFreq {
									asi.SamplingFreqs = append(asi.SamplingFreqs, freq)
								}
							}
						}
					} else {
						// Discrete sampling frequencies
						for i := uint8(0); i < samplingFreqType && 8+i*3 <= uint8(len(block)); i++ {
							freq := uint32(block[8+i*3]) |
								(uint32(block[9+i*3]) << 8) |
								(uint32(block[10+i*3]) << 16)
							asi.SamplingFreqs = append(asi.SamplingFreqs, freq)
						}
					}
				}

			case 0x02: // FORMAT_TYPE_II (MPEG, AC-3, etc.)
				if len(block) >= 9 {
					// MaxBitRate in kbps
					asi.MaxBitRate = uint16(block[4]) | (uint16(block[5]) << 8)
					// SamplesPerFrame
					asi.SamplesPerFrame = uint16(block[6]) | (uint16(block[7]) << 8)
					samplingFreqType := block[8]

					// Parse sampling frequencies (same as Type I)
					if samplingFreqType == 0 && len(block) >= 15 {
						minFreq := uint32(block[9]) | (uint32(block[10]) << 8) | (uint32(block[11]) << 16)
						maxFreq := uint32(block[12]) | (uint32(block[13]) << 8) | (uint32(block[14]) << 16)
						commonFreqs := []uint32{32000, 44100, 48000}
						for _, freq := range commonFreqs {
							if freq >= minFreq && freq <= maxFreq {
								asi.SamplingFreqs = append(asi.SamplingFreqs, freq)
							}
						}
					} else {
						for i := uint8(0); i < samplingFreqType && 9+i*3 <= uint8(len(block)); i++ {
							freq := uint32(block[9+i*3]) |
								(uint32(block[10+i*3]) << 8) |
								(uint32(block[11+i*3]) << 16)
							asi.SamplingFreqs = append(asi.SamplingFreqs, freq)
						}
					}
				}

			case 0x03: // FORMAT_TYPE_III (Format specific)
				if len(block) >= 8 {
					asi.NrChannels = block[4]
					asi.SubframeSize = block[5]
					asi.BitResolution = block[6]
					samplingFreqType := block[7]

					// Store format-specific data
					if len(block) > 8 {
						asi.FormatSpecific = block[8:]
					}

					// Parse sampling frequencies
					if samplingFreqType == 0 && len(block) >= 14 {
						minFreq := uint32(block[8]) | (uint32(block[9]) << 8) | (uint32(block[10]) << 16)
						maxFreq := uint32(block[11]) | (uint32(block[12]) << 8) | (uint32(block[13]) << 16)
						// For Type III, use the range as-is
						asi.SamplingFreqs = []uint32{minFreq, maxFreq}
					} else {
						for i := uint8(0); i < samplingFreqType && 8+i*3 <= uint8(len(block)); i++ {
							freq := uint32(block[8+i*3]) |
								(uint32(block[9+i*3]) << 8) |
								(uint32(block[10+i*3]) << 16)
							asi.SamplingFreqs = append(asi.SamplingFreqs, freq)
						}
					}
				}
			}
		}
	}

	return nil
}

func (asi *AudioStreamingInterface) ParseEndpointFromUSB(ep *usb.Endpoint) {
	asi.EndpointAddress = ep.EndpointAddr
	// Extract the actual packet size (bits 0-10)
	// Bits 11-12 are for additional transactions per microframe (USB 2.0 high-speed)
	asi.MaxPacketSize = ep.MaxPacketSize & 0x07FF
}

func (asi *AudioStreamingInterface) GetCurrentSamplingFreq() (uint32, error) {
	// Read the current sampling frequency from the endpoint
	freqData := make([]byte, 3)
	_, err := asi.handle.ControlTransfer(
		0xA2,   // bmRequestType: Class specific, endpoint, device to host
		0x81,   // GET_CUR
		0x0100, // Sampling Frequency Control
		uint16(asi.EndpointAddress),
		freqData,
		time.Second,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to get sampling frequency: %w", err)
	}

	actualFreq := uint32(freqData[0]) | (uint32(freqData[1]) << 8) | (uint32(freqData[2]) << 16)
	return actualFreq, nil
}

// GetCurrentAudioFormat reads the actual audio format parameters from the device
func (asi *AudioStreamingInterface) GetCurrentAudioFormat() (channels uint8, bitsPerSample uint8, err error) {
	// For UAC1, the format is typically fixed per alternate setting
	// But we can try to query the feature unit for channel config

	// The channels and bit resolution are usually static based on the format descriptor
	// Return what we parsed from the descriptors
	channels = asi.NrChannels
	bitsPerSample = asi.BitResolution

	// Try to verify with a GET_CUR on format-specific parameters
	// Note: This might not be supported by all devices
	ifnum := asi.InterfaceNumber()

	// Try to get channel config from feature unit (unit ID 2 is common for input)
	// Channel config control = 0x02
	channelData := make([]byte, 1)
	_, err = asi.handle.ControlTransfer(
		0xA1,                 // bmRequestType: Class specific, interface, device to host
		0x81,                 // GET_CUR
		0x0200,               // Channel Config control (0x02) in high byte
		(2<<8)|uint16(ifnum), // Feature unit 2, interface number
		channelData,
		100*time.Millisecond,
	)
	if err == nil {
		// If we got a response, it might indicate mono (1) or stereo (2)
		if channelData[0] > 0 && channelData[0] <= 8 {
			channels = channelData[0]
		}
	}

	return channels, bitsPerSample, nil
}

// GetActualAudioFormat queries all current audio parameters from the device
func (asi *AudioStreamingInterface) GetActualAudioFormat() (AudioFormat, error) {
	format := AudioFormat{
		Channels:      int(asi.NrChannels),    // Default from descriptors
		BitsPerSample: int(asi.BitResolution), // Default from descriptors
		SampleRate:    asi.SamplingFreqs[0],   // Default from descriptors
		FormatType:    asi.FormatType,         // From descriptor
		FormatTag:     asi.FormatTag,          // From descriptor
	}

	// Get actual sampling frequency
	if freq, err := asi.GetCurrentSamplingFreq(); err == nil {
		format.SampleRate = freq
	}

	// Get actual channel and bit configuration
	if channels, bits, err := asi.GetCurrentAudioFormat(); err == nil {
		format.Channels = int(channels)
		format.BitsPerSample = int(bits)
	}

	return format, nil
}

func (asi *AudioStreamingInterface) ClaimAudioReader(samplingFreq uint32) (*AudioReader, error) {
	ifnum := asi.InterfaceNumber()
	altSetting := asi.AlternateSetting()

	// Detach kernel driver if attached
	asi.handle.DetachKernelDriver(ifnum)

	// Claim the interface
	if err := asi.handle.ClaimInterface(ifnum); err != nil {
		return nil, fmt.Errorf("claim_interface failed: %w", err)
	}

	// IMPORTANT: First set alternate setting to 0 (stop streaming)
	// This ensures the device properly resets the stream
	asi.handle.SetInterfaceAltSetting(ifnum, 0)
	time.Sleep(50 * time.Millisecond)

	// For UAC, the sampling rate might need to be set while in alt setting 0
	// Try setting the frequency BEFORE changing to the streaming alternate setting
	if samplingFreq > 0 && len(asi.SamplingFreqs) > 0 {
		freqData := make([]byte, 3)
		freqData[0] = byte(samplingFreq & 0xFF)
		freqData[1] = byte((samplingFreq >> 8) & 0xFF)
		freqData[2] = byte((samplingFreq >> 16) & 0xFF)

		// Try to set on the interface while in alt 0
		asi.handle.ControlTransfer(
			0x21,          // bmRequestType: Class specific, interface, host to device
			0x01,          // SET_CUR
			0x0100,        // Sampling Frequency Control
			uint16(ifnum), // Interface number
			freqData,
			time.Second,
		)
	}

	// Now set the alternate setting for audio streaming
	if err := asi.handle.SetInterfaceAltSetting(ifnum, altSetting); err != nil {
		asi.handle.ReleaseInterface(ifnum)
		return nil, fmt.Errorf("set_interface_alt_setting failed: %w", err)
	}

	// Clear any halt condition on the endpoint
	asi.handle.ClearHalt(asi.EndpointAddress)

	// Give the device time to stabilize after interface change
	time.Sleep(50 * time.Millisecond)

	// Set sampling frequency again after alternate setting if UAC supports it
	if samplingFreq > 0 && len(asi.SamplingFreqs) > 0 {
		// For UAC1, we need to send a SET_CUR request for sampling frequency to the endpoint
		freqData := make([]byte, 3)
		freqData[0] = byte(samplingFreq & 0xFF)
		freqData[1] = byte((samplingFreq >> 8) & 0xFF)
		freqData[2] = byte((samplingFreq >> 16) & 0xFF)

		// Try different approaches for setting sampling frequency
		// Method 1: To the endpoint directly (UAC1 standard)
		_, err := asi.handle.ControlTransfer(
			0x22,                        // bmRequestType: Class specific, endpoint, host to device
			0x01,                        // SET_CUR
			0x0100,                      // Sampling Frequency Control (0x01) in high byte
			uint16(asi.EndpointAddress), // Endpoint address
			freqData,
			time.Second,
		)
		if err != nil {
			// Method 2: Try interface-based control
			asi.handle.ControlTransfer(
				0x21,          // bmRequestType: Class specific, interface, host to device
				0x01,          // SET_CUR
				0x0100,        // Sampling Frequency Control
				uint16(ifnum), // Interface number
				freqData,
				time.Second,
			)
		}
	}

	return NewAudioReader(asi)
}
