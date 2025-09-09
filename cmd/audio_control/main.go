package main

import (
	"flag"
	"fmt"
	"log"
	"syscall"

	uvc "github.com/kevmo314/go-uvc"
	"github.com/kevmo314/go-uvc/pkg/transfers"
)

func main() {
	var (
		devicePath = flag.String("device", "/dev/bus/usb/001/003", "USB device path")
		action     = flag.String("action", "info", "Action: info, mute, unmute, volume, agc")
		unitID     = flag.Uint("unit", 2, "Feature unit ID")
		channel    = flag.Uint("channel", 0, "Channel number (0=master)")
		value      = flag.Int("value", 0, "Value for set operations")
	)
	flag.Parse()

	// Open the USB device
	fd, err := openUSBDevice(*devicePath)
	if err != nil {
		log.Fatalf("Failed to open USB device: %v", err)
	}
	defer syscall.Close(fd)

	// Create UAC device
	device, err := uvc.NewUACDevice(uintptr(fd))
	if err != nil {
		log.Fatalf("Failed to create UAC device: %v", err)
	}
	defer device.Close()

	// Get device info
	info, err := device.DeviceInfo()
	if err != nil {
		log.Fatalf("Failed to get device info: %v", err)
	}

	if len(info.StreamingInterfaces) == 0 {
		log.Fatal("No audio interfaces found")
	}

	// Use the first interface for control
	iface := info.StreamingInterfaces[0]
	control := transfers.NewUACControl(
		info.GetHandle(),
		iface.InterfaceNumber(),
	)

	switch *action {
	case "info":
		showControlInfo(control, uint8(*unitID), uint8(*channel))

	case "mute":
		err := control.SetMute(uint8(*unitID), uint8(*channel), true)
		if err != nil {
			log.Printf("Failed to mute: %v", err)
		} else {
			fmt.Println("Muted successfully")
		}

	case "unmute":
		err := control.SetMute(uint8(*unitID), uint8(*channel), false)
		if err != nil {
			log.Printf("Failed to unmute: %v", err)
		} else {
			fmt.Println("Unmuted successfully")
		}

	case "volume":
		// Value is in dB * 256
		volumeDB := int16(*value * 256)
		err := control.SetVolume(uint8(*unitID), uint8(*channel), volumeDB)
		if err != nil {
			log.Printf("Failed to set volume: %v", err)
		} else {
			fmt.Printf("Volume set to %d dB\n", *value)
		}

	case "agc":
		enable := *value != 0
		err := control.SetAGC(uint8(*unitID), uint8(*channel), enable)
		if err != nil {
			log.Printf("Failed to set AGC: %v", err)
		} else {
			fmt.Printf("AGC %s\n", map[bool]string{true: "enabled", false: "disabled"}[enable])
		}

	case "freq":
		// For endpoint control
		if len(info.StreamingInterfaces) > 0 {
			endpoint := info.StreamingInterfaces[0].EndpointAddress
			freq := uint32(*value)
			err := control.SetSamplingFrequency(endpoint, freq)
			if err != nil {
				log.Printf("Failed to set sampling frequency: %v", err)
			} else {
				fmt.Printf("Sampling frequency set to %d Hz\n", freq)
			}

			// Read it back
			actualFreq, err := control.GetSamplingFrequency(endpoint)
			if err == nil {
				fmt.Printf("Device reports: %d Hz\n", actualFreq)
			}
		}

	default:
		log.Fatalf("Unknown action: %s", *action)
	}
}

func showControlInfo(control *transfers.UACControl, unitID, channel uint8) {
	fmt.Println("Audio Control Information:")
	fmt.Printf("Unit ID: %d, Channel: %d\n\n", unitID, channel)

	// Try to get mute status
	muted, err := control.GetMute(unitID, channel)
	if err == nil {
		fmt.Printf("Mute: %v\n", muted)
	} else {
		fmt.Printf("Mute: not supported (%v)\n", err)
	}

	// Try to get volume
	volume, err := control.GetVolume(unitID, channel)
	if err == nil {
		fmt.Printf("Current Volume: %.2f dB\n", float32(volume)/256.0)

		// Get volume range
		min, max, res, err := control.GetVolumeRange(unitID, channel)
		if err == nil {
			fmt.Printf("Volume Range: %.2f to %.2f dB (step: %.2f dB)\n",
				float32(min)/256.0, float32(max)/256.0, float32(res)/256.0)
		}
	} else {
		fmt.Printf("Volume: not supported (%v)\n", err)
	}

	// Try to get status
	status, err := control.GetStatus(unitID, transfers.FU_VOLUME_CONTROL)
	if err == nil && len(status) >= 2 {
		fmt.Printf("Volume Status: 0x%02x%02x\n", status[1], status[0])
	}
}

func openUSBDevice(path string) (int, error) {
	fd, err := syscall.Open(path, syscall.O_RDWR, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to open device %s: %w", path, err)
	}
	return fd, nil
}
