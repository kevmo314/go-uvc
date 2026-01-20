//go:build windows

package main

import (
	"fmt"
	"image/jpeg"
	"log"
	"os"

	"github.com/kevmo314/go-uvc"
)

func main() {
	fmt.Println("Enumerating cameras via Media Foundation...")

	devices, err := uvc.EnumerateUVCDevices()
	if err != nil {
		log.Fatalf("Failed to enumerate devices: %v", err)
	}

	if len(devices) == 0 {
		fmt.Println("No cameras found")
		return
	}

	fmt.Printf("Found %d camera(s):\n\n", len(devices))
	for i, dev := range devices {
		fmt.Printf("%d: %s\n", i, dev.FriendlyName)
		fmt.Printf("   Path: %s\n", dev.SymbolicLink)
	}

	// Open the first camera
	fmt.Println("\nOpening first camera...")
	cam, err := uvc.NewUVCDeviceByIndex(0)
	if err != nil {
		log.Fatalf("Failed to open camera: %v", err)
	}
	defer cam.Close()

	// Get device info
	info, err := cam.DeviceInfo()
	if err != nil {
		log.Fatalf("Failed to get device info: %v", err)
	}

	fmt.Printf("\nCamera: %s\n", info.FriendlyName())
	fmt.Printf("Supported formats:\n")
	for _, f := range info.Formats {
		fmt.Printf("  %s %dx%d @ %.1f fps\n", f.FourCC, f.Width, f.Height, f.FrameRate)
	}

	// Find MJPEG format
	var selectedFormat *uvc.MFMediaFormat
	for _, f := range info.Formats {
		if f.FourCC == "MJPG" && f.Width >= 640 {
			selectedFormat = f
			break
		}
	}

	if selectedFormat == nil {
		// Fallback to any format
		if len(info.Formats) > 0 {
			selectedFormat = info.Formats[0]
		} else {
			log.Fatal("No video formats available")
		}
	}

	fmt.Printf("\nUsing format: %s %dx%d\n", selectedFormat.FourCC, selectedFormat.Width, selectedFormat.Height)

	// Claim frame reader
	reader, err := cam.ClaimFrameReader(selectedFormat.Width, selectedFormat.Height, selectedFormat.FourCC)
	if err != nil {
		log.Fatalf("Failed to claim frame reader: %v", err)
	}
	defer reader.Close()

	// Read a few frames
	fmt.Println("\nCapturing frames...")
	for i := 0; i < 5; i++ {
		img, err := reader.ReadFrameImage()
		if err != nil {
			log.Printf("Error reading frame %d: %v", i+1, err)
			continue
		}

		filename := fmt.Sprintf("frame_%d.jpg", i+1)
		file, err := os.Create(filename)
		if err != nil {
			log.Printf("Error creating file: %v", err)
			continue
		}

		if err := jpeg.Encode(file, img, nil); err != nil {
			log.Printf("Error encoding JPEG: %v", err)
		}
		file.Close()

		bounds := img.Bounds()
		fmt.Printf("Saved %s (%dx%d)\n", filename, bounds.Dx(), bounds.Dy())
	}

	fmt.Println("\nDone!")
}
