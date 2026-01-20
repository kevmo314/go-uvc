package main

import (
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"os"

	"github.com/kevmo314/go-uvc"
	"github.com/kevmo314/go-uvc/pkg/decode"
	"github.com/kevmo314/go-uvc/pkg/descriptors"
	"github.com/kevmo314/go-uvc/pkg/transfers"
)

func main() {
	path := flag.String("path", "/dev/bus/usb/001/007", "path to the usb device")
	output := flag.String("output", "frame", "output filename prefix (will save as frame_N.jpg)")
	count := flag.Int("count", 5, "number of frames to capture")
	flag.Parse()

	log.Printf("Opening USB device at %s", *path)

	// Open the USB device
	fd, err := os.OpenFile(*path, os.O_RDWR, 0)
	if err != nil {
		log.Fatalf("Failed to open device: %v", err)
	}
	defer fd.Close()

	// Create UVC device
	dev, err := uvc.NewUVCDevice(fd.Fd())
	if err != nil {
		log.Fatalf("Failed to create UVC device: %v", err)
	}

	// Get device info
	info, err := dev.DeviceInfo()
	if err != nil {
		log.Fatalf("Failed to get device info: %v", err)
	}

	log.Printf("Device has %d streaming interfaces", len(info.StreamingInterfaces))

	// Find first MJPEG format
	var selectedInterface *transfers.StreamingInterface
	var selectedFormat descriptors.FormatDescriptor
	var selectedFrame descriptors.FrameDescriptor

	for _, si := range info.StreamingInterfaces {
		log.Printf("Checking interface %d", si.InterfaceNumber())
		for _, desc := range si.Descriptors {
			if fd, ok := desc.(*descriptors.MJPEGFormatDescriptor); ok {
				log.Printf("Found MJPEG format with %d frame descriptors", fd.NumFrameDescriptors)

				// Find the first frame descriptor after this format
				for i, d := range si.Descriptors {
					if d == desc {
						// Get the first frame descriptor
						if i+1 < len(si.Descriptors) {
							if fr, ok := si.Descriptors[i+1].(*descriptors.MJPEGFrameDescriptor); ok {
								selectedInterface = si
								selectedFormat = fd
								selectedFrame = fr
								log.Printf("Selected MJPEG frame: %dx%d", fr.Width, fr.Height)
								goto found
							}
						}
					}
				}
			}
		}
	}

found:
	if selectedInterface == nil {
		log.Fatal("No MJPEG format found")
	}

	// Claim the frame reader
	log.Printf("Claiming frame reader for format index %d, frame index %d",
		selectedFormat.Index(), selectedFrame.Index())

	reader, err := selectedInterface.ClaimFrameReader(
		selectedFormat.Index(),
		selectedFrame.Index(),
	)
	if err != nil {
		log.Fatalf("Failed to claim frame reader: %v", err)
	}
	defer reader.Close()

	// Create decoder
	decoder, err := decode.NewFrameReaderDecoder(reader, selectedFormat, selectedFrame)
	if err != nil {
		log.Fatalf("Failed to create decoder: %v", err)
	}

	// Capture frames
	for i := 0; i < *count; i++ {
		log.Printf("Reading frame %d/%d...", i+1, *count)

		img, err := decoder.ReadFrame()
		if err != nil {
			log.Printf("Error reading frame %d: %v", i+1, err)
			continue
		}

		// Save the frame
		filename := fmt.Sprintf("%s_%d.jpg", *output, i+1)
		file, err := os.Create(filename)
		if err != nil {
			log.Printf("Failed to create file %s: %v", filename, err)
			continue
		}

		// Try to save as JPEG if it's already JPEG data, otherwise as PNG
		var encodeErr error
		switch img.(type) {
		case *image.YCbCr, *image.RGBA, *image.NRGBA:
			encodeErr = jpeg.Encode(file, img, nil)
		default:
			encodeErr = png.Encode(file, img)
		}

		file.Close()

		if encodeErr != nil {
			log.Printf("Failed to encode image %d: %v", i+1, encodeErr)
			os.Remove(filename)
		} else {
			bounds := img.Bounds()
			log.Printf("Successfully saved frame %d as %s (%dx%d)",
				i+1, filename, bounds.Dx(), bounds.Dy())
		}
	}

	log.Printf("Capture complete. Saved %d frames with prefix '%s'", *count, *output)
}
