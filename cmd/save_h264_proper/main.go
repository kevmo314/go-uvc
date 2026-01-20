package main

import (
	"flag"
	"io"
	"log"
	"os"

	"github.com/kevmo314/go-uvc"
	"github.com/kevmo314/go-uvc/pkg/descriptors"
	"github.com/kevmo314/go-uvc/pkg/transfers"
)

// SPS/PPS from BELABOX gstlibuvch264src
var sps = []byte{
	0x00, 0x00, 0x00, 0x01, 0x67, 0x64, 0x00, 0x34,
	0xAC, 0x4D, 0x00, 0xF0, 0x04, 0x4F, 0xCB, 0x35,
	0x01, 0x01, 0x01, 0x40, 0x00, 0x00, 0xFA, 0x00,
	0x00, 0x3A, 0x98, 0x03, 0xC7, 0x0C, 0xA8,
}
var pps = []byte{
	0x00, 0x00, 0x00, 0x01, 0x68, 0xEE, 0x3C, 0xB0,
}

func main() {
	path := flag.String("path", "/dev/bus/usb/001/046", "path to the usb device")
	output := flag.String("output", "/tmp/osmo_proper.h264", "output file")
	frames := flag.Int("frames", 120, "number of frames to save")
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

	var si *transfers.StreamingInterface
	var sf descriptors.FormatDescriptor
	var fr descriptors.FrameDescriptor

	// Find H264 format with 1920x1080 resolution
	for _, s := range info.StreamingInterfaces {
		for _, desc := range s.Descriptors {
			if f, ok := desc.(*descriptors.FrameBasedFormatDescriptor); ok {
				for _, dd := range s.Descriptors {
					if ff, ok := dd.(*descriptors.FrameBasedFrameDescriptor); ok {
						// Select 1080p (matching our SPS/PPS)
						if ff.Width == 1920 && ff.Height == 1080 {
							si, sf, fr = s, f, ff
							log.Printf("Selected H264 format: %dx%d", ff.Width, ff.Height)
							goto found
						}
					}
				}
			}
		}
	}

found:
	if si == nil {
		log.Fatal("No Frame-Based format found")
	}

	reader, err := si.ClaimFrameReader(sf.Index(), fr.Index())
	if err != nil {
		log.Fatalf("Failed to claim frame reader: %v", err)
	}
	defer reader.Close()

	out, err := os.Create(*output)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer out.Close()

	log.Printf("Saving %d frames to %s", *frames, *output)

	hadIDR := false
	idrCount := 0

	for i := 0; i < *frames; i++ {
		frame, err := reader.ReadFrame()
		if err != nil {
			log.Printf("Frame %d: read error: %v", i, err)
			continue
		}

		data, err := io.ReadAll(frame)
		if err != nil {
			log.Printf("Frame %d: ReadAll error: %v", i, err)
			continue
		}

		// Check for IDR
		hasIDR := false
		for j := 0; j < len(data)-4; j++ {
			if data[j] == 0 && data[j+1] == 0 && data[j+2] == 0 && data[j+3] == 1 {
				nalType := data[j+4] & 0x1F
				if nalType == 5 {
					hasIDR = true
					break
				}
			}
		}

		if hasIDR {
			idrCount++
			log.Printf("Frame %d: IDR #%d (%d bytes)", i, idrCount, len(data))
			// Prepend SPS/PPS before IDR
			out.Write(sps)
			out.Write(pps)
			hadIDR = true
		} else if !hadIDR {
			log.Printf("Frame %d: Skipping non-IDR before first IDR (%d bytes)", i, len(data))
			continue
		}

		out.Write(data)
	}

	log.Printf("Done. Saved %d frames (%d IDRs) to %s", *frames, idrCount, *output)
}
