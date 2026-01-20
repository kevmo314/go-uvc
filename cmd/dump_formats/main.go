package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/kevmo314/go-uvc"
	"github.com/kevmo314/go-uvc/pkg/descriptors"
)

func main() {
	path := flag.String("path", "/dev/bus/usb/001/046", "path to the usb device")
	flag.Parse()

	log.Printf("Opening USB device at %s", *path)

	fd, err := os.OpenFile(*path, os.O_RDWR, 0)
	if err != nil {
		log.Fatalf("Failed to open device: %v", err)
	}
	defer fd.Close()

	dev, err := uvc.NewUVCDevice(fd.Fd())
	if err != nil {
		log.Fatalf("Failed to create UVC device: %v", err)
	}
	defer dev.Close()

	info, err := dev.DeviceInfo()
	if err != nil {
		log.Fatalf("Failed to get device info: %v", err)
	}

	fmt.Println("=== Streaming Interfaces ===")
	for i, si := range info.StreamingInterfaces {
		fmt.Printf("\nStreaming Interface %d:\n", i)

		for _, desc := range si.Descriptors {
			switch d := desc.(type) {
			case *descriptors.FrameBasedFormatDescriptor:
				fourcc, _ := d.FourCC()
				fmt.Printf("  Format %d: Frame-Based (FourCC: %s, GUID: %s)\n",
					d.FormatIndex, string(fourcc[:]), d.GUIDFormat)
				fmt.Printf("    BitsPerPixel: %d, DefaultFrameIndex: %d\n",
					d.BitsPerPixel, d.DefaultFrameIndex)
			case *descriptors.FrameBasedFrameDescriptor:
				fmt.Printf("    Frame %d: %dx%d\n", d.FrameIndex, d.Width, d.Height)
				fmt.Printf("      BitRate: %d - %d bps\n", d.MinBitRate, d.MaxBitRate)
				fmt.Printf("      DefaultFrameInterval: %v (%.2f fps)\n",
					d.DefaultFrameInterval, 1e9/float64(d.DefaultFrameInterval.Nanoseconds()))
				if len(d.DiscreteFrameIntervals) > 0 {
					fmt.Printf("      Discrete intervals: ")
					for _, interval := range d.DiscreteFrameIntervals {
						fmt.Printf("%.2f fps ", 1e9/float64(interval.Nanoseconds()))
					}
					fmt.Println()
				}
			case *descriptors.MJPEGFormatDescriptor:
				fmt.Printf("  Format %d: MJPEG\n", d.FormatIndex)
			case *descriptors.MJPEGFrameDescriptor:
				fmt.Printf("    Frame %d: %dx%d\n", d.FrameIndex, d.Width, d.Height)
			case *descriptors.UncompressedFormatDescriptor:
				fourcc, _ := d.FourCC()
				fmt.Printf("  Format %d: Uncompressed (FourCC: %s)\n", d.FormatIndex, string(fourcc[:]))
			case *descriptors.UncompressedFrameDescriptor:
				fmt.Printf("    Frame %d: %dx%d\n", d.FrameIndex, d.Width, d.Height)
			}
		}
	}
}
