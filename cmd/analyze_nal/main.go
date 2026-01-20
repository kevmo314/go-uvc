package main

import (
	"encoding/hex"
	"flag"
	"io"
	"log"
	"os"

	"github.com/kevmo314/go-uvc"
	"github.com/kevmo314/go-uvc/pkg/descriptors"
	"github.com/kevmo314/go-uvc/pkg/transfers"
)

func main() {
	path := flag.String("path", "/dev/bus/usb/001/046", "path to the usb device")
	frames := flag.Int("frames", 5, "frames to analyze")
	flag.Parse()

	fd, _ := os.OpenFile(*path, os.O_RDWR, 0)
	defer fd.Close()
	dev, _ := uvc.NewUVCDevice(fd.Fd())
	defer dev.Close()
	info, _ := dev.DeviceInfo()

	var si *transfers.StreamingInterface
	var sf descriptors.FormatDescriptor
	var fr descriptors.FrameDescriptor

	for _, s := range info.StreamingInterfaces {
		for _, desc := range s.Descriptors {
			if f, ok := desc.(*descriptors.FrameBasedFormatDescriptor); ok {
				for _, dd := range s.Descriptors {
					if ff, ok := dd.(*descriptors.FrameBasedFrameDescriptor); ok {
						if ff.Width == 1920 && ff.Height == 1080 {
							si, sf, fr = s, f, ff
							goto found
						}
					}
				}
			}
		}
	}

found:
	reader, _ := si.ClaimFrameReader(sf.Index(), fr.Index())
	defer reader.Close()

	for i := 0; i < *frames; i++ {
		frame, _ := reader.ReadFrame()
		data, _ := io.ReadAll(frame)

		log.Printf("\n=== Frame %d: %d bytes, %d payloads ===", i, len(data), len(frame.Payloads))

		// Show first 64 bytes
		showLen := 64
		if len(data) < showLen {
			showLen = len(data)
		}
		log.Printf("First %d bytes: %s", showLen, hex.EncodeToString(data[:showLen]))

		// Find and analyze all NAL units
		offset := 0
		nalCount := 0
		for offset < len(data)-4 {
			// Find start code
			if data[offset] == 0 && data[offset+1] == 0 {
				startCodeLen := 0
				if offset+3 < len(data) && data[offset+2] == 0 && data[offset+3] == 1 {
					startCodeLen = 4
				} else if offset+2 < len(data) && data[offset+2] == 1 {
					startCodeLen = 3
				}

				if startCodeLen > 0 {
					nalHeader := data[offset+startCodeLen]
					forbiddenBit := (nalHeader >> 7) & 1
					nalRefIdc := (nalHeader >> 5) & 3
					nalType := nalHeader & 0x1F

					// Find NAL length
					nextOffset := offset + startCodeLen + 1
					for nextOffset < len(data)-3 {
						if data[nextOffset] == 0 && data[nextOffset+1] == 0 {
							if (nextOffset+3 < len(data) && data[nextOffset+2] == 0 && data[nextOffset+3] == 1) ||
								data[nextOffset+2] == 1 {
								break
							}
						}
						nextOffset++
					}
					if nextOffset >= len(data)-3 {
						nextOffset = len(data)
					}

					nalLen := nextOffset - offset

					nalTypeName := ""
					switch nalType {
					case 1:
						nalTypeName = "Non-IDR slice"
					case 5:
						nalTypeName = "IDR slice"
					case 6:
						nalTypeName = "SEI"
					case 7:
						nalTypeName = "SPS"
					case 8:
						nalTypeName = "PPS"
					case 9:
						nalTypeName = "AUD"
					default:
						nalTypeName = "Unknown"
					}

					log.Printf("  NAL #%d at offset %d: type=%d (%s), ref_idc=%d, forbidden=%d, len=%d",
						nalCount, offset, nalType, nalTypeName, nalRefIdc, forbiddenBit, nalLen)

					// For slice NALs, decode slice header info
					if nalType == 1 || nalType == 5 {
						sliceData := data[offset+startCodeLen+1:]
						if len(sliceData) > 8 {
							log.Printf("    Slice header bytes: %s", hex.EncodeToString(sliceData[:8]))
						}
					}

					nalCount++
					offset = nextOffset
					continue
				}
			}
			offset++
		}
		log.Printf("  Total NAL units: %d", nalCount)
	}
}
