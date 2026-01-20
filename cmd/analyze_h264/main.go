package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/kevmo314/go-uvc"
	"github.com/kevmo314/go-uvc/pkg/descriptors"
	"github.com/kevmo314/go-uvc/pkg/transfers"
)

func main() {
	path := flag.String("path", "/dev/bus/usb/001/046", "path to the usb device")
	frames := flag.Int("frames", 60, "number of frames to analyze")
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

	var selectedInterface *transfers.StreamingInterface
	var selectedFormat descriptors.FormatDescriptor
	var selectedFrame descriptors.FrameDescriptor

	for _, si := range info.StreamingInterfaces {
		for _, desc := range si.Descriptors {
			if fd, ok := desc.(*descriptors.FrameBasedFormatDescriptor); ok {
				var frames []*descriptors.FrameBasedFrameDescriptor
				for _, d := range si.Descriptors {
					if fr, ok := d.(*descriptors.FrameBasedFrameDescriptor); ok {
						frames = append(frames, fr)
					}
				}
				if len(frames) > 0 {
					selectedInterface = si
					selectedFormat = fd
					selectedFrame = frames[0]
					goto found
				}
			}
		}
	}

found:
	if selectedInterface == nil {
		log.Fatal("No Frame-Based format found")
	}

	reader, err := selectedInterface.ClaimFrameReader(selectedFormat.Index(), selectedFrame.Index())
	if err != nil {
		log.Fatalf("Failed to claim frame reader: %v", err)
	}
	defer reader.Close()

	log.Printf("Reading %d frames...\n", *frames)

	nalTypeCounts := make(map[int]int)
	var firstIDRFrame int = -1

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

		// Find all NAL units
		nalUnits := findNALUnits(data)

		for _, unit := range nalUnits {
			nalType := unit[2]
			nalTypeCounts[nalType]++

			nalData := data[unit[0]:unit[1]]

			// Print details for interesting NAL types
			switch nalType {
			case 7: // SPS
				log.Printf("Frame %d: SPS found (%d bytes)", i, len(nalData))
				log.Printf("  Hex: %s", hex.EncodeToString(nalData))
				parseSPS(nalData)
			case 8: // PPS
				log.Printf("Frame %d: PPS found (%d bytes)", i, len(nalData))
				log.Printf("  Hex: %s", hex.EncodeToString(nalData))
				parsePPS(nalData)
			case 5: // IDR
				if firstIDRFrame < 0 {
					firstIDRFrame = i
				}
				log.Printf("Frame %d: IDR slice (%d bytes)", i, len(nalData))
				// Show first 32 bytes
				showLen := 32
				if len(nalData) < showLen {
					showLen = len(nalData)
				}
				log.Printf("  First %d bytes: %s", showLen, hex.EncodeToString(nalData[:showLen]))
			case 1: // Non-IDR slice
				// Only print first occurrence
				if nalTypeCounts[1] <= 3 {
					log.Printf("Frame %d: Non-IDR slice (%d bytes)", i, len(nalData))
				}
			case 6: // SEI
				log.Printf("Frame %d: SEI (%d bytes)", i, len(nalData))
			}
		}
	}

	log.Println("\n=== Summary ===")
	log.Printf("First IDR frame: %d", firstIDRFrame)
	log.Printf("NAL type distribution:")
	for nalType, count := range nalTypeCounts {
		name := nalTypeName(nalType)
		log.Printf("  Type %d (%s): %d", nalType, name, count)
	}
}

func nalTypeName(t int) string {
	names := map[int]string{
		0:  "Unspecified",
		1:  "Non-IDR slice",
		2:  "Slice partition A",
		3:  "Slice partition B",
		4:  "Slice partition C",
		5:  "IDR slice",
		6:  "SEI",
		7:  "SPS",
		8:  "PPS",
		9:  "Access unit delimiter",
		10: "End of sequence",
		11: "End of stream",
		12: "Filler data",
	}
	if name, ok := names[t]; ok {
		return name
	}
	return fmt.Sprintf("Unknown(%d)", t)
}

func findNALUnits(data []byte) [][3]int {
	var units [][3]int
	i := 0
	for i < len(data)-4 {
		if data[i] == 0 && data[i+1] == 0 {
			startCodeLen := 0
			if data[i+2] == 0 && data[i+3] == 1 {
				startCodeLen = 4
			} else if data[i+2] == 1 {
				startCodeLen = 3
			}
			if startCodeLen > 0 {
				nalStart := i
				nalType := int(data[i+startCodeLen] & 0x1F)

				j := i + startCodeLen + 1
				for j < len(data)-3 {
					if data[j] == 0 && data[j+1] == 0 {
						if (j+3 < len(data) && data[j+2] == 0 && data[j+3] == 1) || data[j+2] == 1 {
							break
						}
					}
					j++
				}
				if j >= len(data)-3 {
					j = len(data)
				}
				units = append(units, [3]int{nalStart, j, nalType})
				i = j
				continue
			}
		}
		i++
	}
	return units
}

// parseSPS prints basic SPS information
func parseSPS(data []byte) {
	// Skip start code
	offset := 0
	if data[0] == 0 && data[1] == 0 && data[2] == 0 && data[3] == 1 {
		offset = 4
	} else if data[0] == 0 && data[1] == 0 && data[2] == 1 {
		offset = 3
	}

	if len(data) <= offset {
		return
	}

	nalHeader := data[offset]
	log.Printf("  NAL header: 0x%02x (type=%d)", nalHeader, nalHeader&0x1F)

	if len(data) > offset+1 {
		profileIdc := data[offset+1]
		log.Printf("  profile_idc: %d", profileIdc)

		switch profileIdc {
		case 66:
			log.Printf("    -> Baseline Profile")
		case 77:
			log.Printf("    -> Main Profile")
		case 88:
			log.Printf("    -> Extended Profile")
		case 100:
			log.Printf("    -> High Profile")
		case 110:
			log.Printf("    -> High 10 Profile")
		case 122:
			log.Printf("    -> High 4:2:2 Profile")
		case 244:
			log.Printf("    -> High 4:4:4 Predictive Profile")
		}
	}

	if len(data) > offset+2 {
		constraintFlags := data[offset+2]
		log.Printf("  constraint_set_flags: 0x%02x", constraintFlags)
	}

	if len(data) > offset+3 {
		levelIdc := data[offset+3]
		log.Printf("  level_idc: %d (Level %.1f)", levelIdc, float64(levelIdc)/10.0)
	}
}

// parsePPS prints basic PPS information
func parsePPS(data []byte) {
	// Skip start code
	offset := 0
	if data[0] == 0 && data[1] == 0 && data[2] == 0 && data[3] == 1 {
		offset = 4
	} else if data[0] == 0 && data[1] == 0 && data[2] == 1 {
		offset = 3
	}

	if len(data) <= offset {
		return
	}

	nalHeader := data[offset]
	log.Printf("  NAL header: 0x%02x (type=%d)", nalHeader, nalHeader&0x1F)

	// PPS is Exp-Golomb encoded, so we can't easily parse it without a proper parser
	log.Printf("  (PPS requires Exp-Golomb decoding for detailed parsing)")
}
