package main

import (
	"encoding/hex"
	"flag"
	"log"
	"os"

	"github.com/kevmo314/go-uvc"
	"github.com/kevmo314/go-uvc/pkg/decode"
	"github.com/kevmo314/go-uvc/pkg/descriptors"
	"github.com/kevmo314/go-uvc/pkg/transfers"
)

func main() {
	path := flag.String("path", "/dev/bus/usb/001/023", "path to the usb device")
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
	defer dev.Close()

	// Get device info
	info, err := dev.DeviceInfo()
	if err != nil {
		log.Fatalf("Failed to get device info: %v", err)
	}

	log.Printf("Device has %d streaming interfaces", len(info.StreamingInterfaces))

	// Find H264 format (frame-based)
	var selectedInterface *transfers.StreamingInterface
	var selectedFormat descriptors.FormatDescriptor
	var selectedFrame descriptors.FrameDescriptor

	for _, si := range info.StreamingInterfaces {
		log.Printf("Checking interface %d", si.InterfaceNumber())
		for _, desc := range si.Descriptors {
			// Check for frame-based format descriptor
			if fd, ok := desc.(*descriptors.FrameBasedFormatDescriptor); ok {
				log.Printf("Found Frame-Based format with GUID: %X, %d frame descriptors",
					fd.GUIDFormat, fd.NumFrameDescriptors)

				// Check if this is H264 (the GUID should indicate H264)
				// H264 GUID is typically: 48323634-0000-0010-8000-00AA00389B71
				// But let's log what we find
				log.Printf("  Format GUID bytes: %s", hex.EncodeToString(fd.GUIDFormat[:]))

				// Find frame descriptor - list all available frames first
				var frames []*descriptors.FrameBasedFrameDescriptor
				for _, d := range si.Descriptors {
					if fr, ok := d.(*descriptors.FrameBasedFrameDescriptor); ok {
						frames = append(frames, fr)
						log.Printf("  Available frame %d: %dx%d", fr.FrameIndex, fr.Width, fr.Height)
					}
				}

				// Select the first frame
				if len(frames) > 0 {
					selectedInterface = si
					selectedFormat = fd
					selectedFrame = frames[1]
					log.Printf("Selected Frame-Based frame index %d: %dx%d",
						selectedFrame.Index(), frames[1].Width, frames[1].Height)
					goto found
				}
			}
		}
	}

found:
	if selectedInterface == nil {
		log.Fatal("No Frame-Based H264 format found")
	}

	// Create an H264 decoder
	log.Printf("Creating H264 decoder...")
	h264Decoder, err := decode.NewH264Decoder()
	if err != nil {
		log.Fatalf("Failed to create H264 decoder: %v", err)
	}
	defer h264Decoder.Close()

	// Try different frame indices if the first one fails
	var reader *transfers.FrameReader
	frameIndices := []uint8{1, 2} // Try both available frames
	formatIndex := selectedFormat.Index()

	for _, frameIdx := range frameIndices {
		log.Printf("Attempting to claim frame reader for format index %d, frame index %d",
			formatIndex, frameIdx)

		reader, err = selectedInterface.ClaimFrameReader(formatIndex, frameIdx)
		if err != nil {
			log.Printf("Failed with frame index %d: %v", frameIdx, err)
			continue
		}
		log.Printf("Successfully claimed with frame index %d", frameIdx)
		break
	}

	if reader == nil {
		log.Fatalf("Failed to claim frame reader with any frame index")
	}
	defer reader.Close()

	// 00 00 00 01 67 64 00 34 ac 4d 00 f0 04 4f cb 35 01 01 01 40 00 00 fa 00 00 3a 98 03 c7 0c a8 00 00 00 01 68 ee 3c b0
	if _, err := h264Decoder.Write([]byte{
		0x00, 0x00, 0x00, 0x01, 0x67, 0x64, 0x00, 0x34, 0xac, 0x4d, 0x00, 0xf0, 0x04, 0x4f, 0xcb, 0x35, 0x01, 0x01, 0x01, 0x40, 0x00, 0x00, 0xfa,
		0x00, 0x00, 0x3a, 0x98, 0x03, 0xc7, 0x0c, 0xa8, 0x00, 0x00, 0x00, 0x01, 0x68, 0xee, 0x3c, 0xb0,
	}); err != nil {
		log.Printf("Failed to write raw data to H264 decoder: %v", err)
	}

	// Read and decode frames
	for i := 0; ; i++ {
		log.Printf("\n=== Reading frame %d ===", i+1)

		frame, err := reader.ReadFrame()
		if err != nil {
			log.Printf("Error reading frame: %v", err)
			continue
		}

		log.Printf("Frame has %d payloads", len(frame.Payloads))

		// Examine payloads
		totalSize := 0
		for j, p := range frame.Payloads {
			totalSize += len(p.Data)

			dataPreview := p.Data
			if len(dataPreview) > 32 {
				dataPreview = dataPreview[:32]
			}

			log.Printf("  Payload %d: Header=0x%02x (FID=%v, EOF=%v, PTS=%v, SCR=%v, ERR=%v), DataLen=%d",
				j, p.HeaderInfoBitmask, p.FrameID(), p.EndOfFrame(), p.HasPTS(), p.HasSCR(), p.Error(), len(p.Data))

			if j == 0 || j == len(frame.Payloads)-1 {
				log.Printf("    First %d bytes: %s", len(dataPreview), hex.EncodeToString(dataPreview))
				if len(p.Data) > 32 {
					log.Printf("    Last 32 bytes: %s", hex.EncodeToString(p.Data[len(p.Data)-32:]))
				}
			}

			// Check for H264 NAL unit start codes
			if j == 0 && len(p.Data) >= 4 {
				// Check for 0x00000001 or 0x000001 start codes
				if p.Data[0] == 0x00 && p.Data[1] == 0x00 {
					if p.Data[2] == 0x00 && p.Data[3] == 0x01 {
						log.Printf("    ✓ Found H264 start code (0x00000001)")
						if len(p.Data) > 4 {
							nalType := p.Data[4] & 0x1F
							log.Printf("    NAL unit type: %d", nalType)
						}
					} else if p.Data[2] == 0x01 {
						log.Printf("    ✓ Found H264 start code (0x000001)")
						if len(p.Data) > 3 {
							nalType := p.Data[3] & 0x1F
							log.Printf("    NAL unit type: %d", nalType)
						}
					}
				} else {
					log.Printf("    ✗ No H264 start code found (got %02X%02X%02X%02X)",
						p.Data[0], p.Data[1], p.Data[2], p.Data[3])
				}
			}
		}

		// Total size was already calculated above
		log.Printf("Total frame size: %d bytes", totalSize)

		// Try to decode the frame
		log.Printf("\n--- Attempting to decode frame %d with H264 decoder ---", i+1)

		err = h264Decoder.WriteUSBFrame(frame)
		if err != nil {
			log.Printf("❌ H264 decoder error: %v", err)
			log.Printf("  Error code -1094995529 is AVERROR_INVALIDDATA in ffmpeg")

			// Concatenate payload data for analysis
			var fullData []byte
			for _, p := range frame.Payloads {
				fullData = append(fullData, p.Data...)
			}

			// Search for NAL units in the data
			log.Printf("Searching for NAL unit start codes in frame data...")
			nalCount := 0
			for idx := 0; idx < len(fullData)-3; idx++ {
				if fullData[idx] == 0x00 && fullData[idx+1] == 0x00 {
					if idx+3 < len(fullData) && fullData[idx+2] == 0x00 && fullData[idx+3] == 0x01 {
						nalCount++
						nalType := 0
						if idx+4 < len(fullData) {
							nalType = int(fullData[idx+4] & 0x1F)
						}
						log.Printf("  Found 4-byte NAL start code #%d at position %d, type=%d", nalCount, idx, nalType)
						idx += 3 // Skip ahead
					} else if fullData[idx+2] == 0x01 {
						nalCount++
						nalType := 0
						if idx+3 < len(fullData) {
							nalType = int(fullData[idx+3] & 0x1F)
						}
						log.Printf("  Found 3-byte NAL start code #%d at position %d, type=%d", nalCount, idx, nalType)
						idx += 2 // Skip ahead
					}
				}
			}
			log.Printf("  Total NAL start codes found: %d", nalCount)

			// Show first 256 bytes of data to understand the structure
			showBytes := min(256, len(fullData))
			log.Printf("\nFirst %d bytes of frame data:", showBytes)
			for offset := 0; offset < showBytes; offset += 32 {
				end := min(offset+32, showBytes)
				log.Printf("  %04X: %s", offset, hex.EncodeToString(fullData[offset:end]))
			}

			// If this is the first frame with an error, stop to debug
			if i == 0 {
				log.Printf("\nStopping after first frame error for debugging")
				return
			}
		} else {
			log.Printf("✅ H264 decoder successfully processed the frame")

			// Try to read the decoded image
			img, err := h264Decoder.ReadFrame()
			if err != nil {
				log.Printf("No decoded frame available yet: %v", err)
			} else {
				bounds := img.Bounds()
				log.Printf("Decoded image dimensions: %dx%d", bounds.Dx(), bounds.Dy())
			}
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
