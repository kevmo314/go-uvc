package main

import (
	"fmt"
	"log"

	usb "github.com/kevmo314/go-usb"
)

func main() {
	fmt.Println("Listing USB devices...")

	devices, err := usb.DeviceList()
	if err != nil {
		log.Fatalf("Failed to list devices: %v", err)
	}

	if len(devices) == 0 {
		fmt.Println("No USB devices found (WinUSB driver required)")
		fmt.Println("\nNote: On Windows, only devices with the WinUSB driver installed")
		fmt.Println("can be accessed. Webcams typically use the UVC driver.")
		fmt.Println("Use Zadig (https://zadig.akeo.ie/) to install WinUSB driver.")
		return
	}

	fmt.Printf("Found %d device(s):\n\n", len(devices))

	for i, dev := range devices {
		fmt.Printf("Device %d:\n", i+1)
		fmt.Printf("  Path: %s\n", dev.Path)
		fmt.Printf("  VID:PID: %04x:%04x\n", dev.Descriptor.VendorID, dev.Descriptor.ProductID)
		fmt.Printf("  USB Version: %d.%02d\n", dev.Descriptor.USBVersion>>8, dev.Descriptor.USBVersion&0xFF)
		fmt.Printf("  Class: %d, SubClass: %d, Protocol: %d\n",
			dev.Descriptor.DeviceClass, dev.Descriptor.DeviceSubClass, dev.Descriptor.DeviceProtocol)

		if dev.SysfsStrings != nil {
			if dev.SysfsStrings.Manufacturer != "" {
				fmt.Printf("  Manufacturer: %s\n", dev.SysfsStrings.Manufacturer)
			}
			if dev.SysfsStrings.Product != "" {
				fmt.Printf("  Product: %s\n", dev.SysfsStrings.Product)
			}
			if dev.SysfsStrings.Serial != "" {
				fmt.Printf("  Serial: %s\n", dev.SysfsStrings.Serial)
			}
		}

		// Check if it's a UVC device (class 14 = video, subclass 1 = video control)
		if dev.Descriptor.DeviceClass == 239 && dev.Descriptor.DeviceSubClass == 2 {
			fmt.Printf("  ** This appears to be a composite USB device (possibly webcam) **\n")
		}

		// Try to open and get more info
		handle, err := dev.Open()
		if err != nil {
			fmt.Printf("  (Could not open: %v)\n", err)
		} else {
			// Try to get config descriptor
			config, err := handle.GetActiveConfigDescriptor()
			if err == nil {
				fmt.Printf("  Active Config: %d, Interfaces: %d\n",
					config.ConfigurationValue, config.NumInterfaces)

				// Check interfaces for UVC
				for _, iface := range config.Interfaces {
					for _, alt := range iface.AltSettings {
						if alt.InterfaceClass == 14 { // Video class
							fmt.Printf("    Interface %d: Video Class (UVC)\n", alt.InterfaceNumber)
						}
					}
				}
			}
			handle.Close()
		}

		fmt.Println()
	}
}
