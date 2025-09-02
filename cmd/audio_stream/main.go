package main

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	uvc "github.com/kevmo314/go-uvc"
	"github.com/kevmo314/go-uvc/pkg/transfers"
)

type AudioFormat = transfers.AudioFormat

func main() {
	var (
		devicePath   = flag.String("device", "/dev/bus/usb/001/003", "USB device path")
		outputFile   = flag.String("output", "audio.pcm", "Output PCM file")
		duration     = flag.Duration("duration", 10*time.Second, "Recording duration")
		samplingFreq = flag.Uint("freq", 48000, "Sampling frequency in Hz")
		probeOnly    = flag.Bool("probe", false, "Probe device formats and exit")
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

	fmt.Printf("Found %d audio streaming interfaces\n", len(info.StreamingInterfaces))

	// Debug: print all interfaces found with format details
	for i, si := range info.StreamingInterfaces {
		fmt.Printf("\nInterface %d: num=%d, alt=%d, endpoint=0x%02x\n",
			i, si.InterfaceNumber(), si.AlternateSetting(), si.EndpointAddress)
		
		// Print format type and tag
		formatTypeName := "Unknown"
		switch si.FormatType {
		case 0x01:
			formatTypeName = "Type I (PCM/Linear)"
		case 0x02:
			formatTypeName = "Type II (Dynamic/MPEG/AC3)"
		case 0x03:
			formatTypeName = "Type III (Format Specific)"
		}
		
		formatTagName := ""
		switch si.FormatTag {
		case 0x0001:
			formatTagName = "PCM"
		case 0x0002:
			formatTagName = "PCM8"
		case 0x0003:
			formatTagName = "IEEE_FLOAT"
		case 0x0004:
			formatTagName = "ALAW"
		case 0x0005:
			formatTagName = "MULAW"
		case 0x1001:
			formatTagName = "MPEG"
		case 0x1002:
			formatTagName = "AC3"
		default:
			formatTagName = fmt.Sprintf("0x%04x", si.FormatTag)
		}
		
		fmt.Printf("  Format: %s, Tag: %s\n", formatTypeName, formatTagName)
		fmt.Printf("  Audio: %d channels, %d bits, subframe: %d bytes\n", 
			si.NrChannels, si.BitResolution, si.SubframeSize)
		fmt.Printf("  Sampling frequencies: %v Hz\n", si.SamplingFreqs)
		
		// Type II specific info
		if si.FormatType == 0x02 {
			fmt.Printf("  Max bitrate: %d kbps, Samples per frame: %d\n", 
				si.MaxBitRate, si.SamplesPerFrame)
		}
		
		// Type III specific info
		if si.FormatType == 0x03 && len(si.FormatSpecific) > 0 {
			fmt.Printf("  Format specific data: %x\n", si.FormatSpecific)
		}
	}

	if len(info.StreamingInterfaces) == 0 {
		log.Fatal("No audio streaming interfaces found")
	}
	
	// Also check for MIDI interfaces
	if len(info.MIDIInterfaces) > 0 {
		fmt.Printf("\nFound %d MIDI streaming interfaces\n", len(info.MIDIInterfaces))
		for i, mi := range info.MIDIInterfaces {
			fmt.Printf("\nMIDI Interface %d: num=%d\n", i, mi.InterfaceNumber())
			fmt.Printf("  IN Jacks: %d, OUT Jacks: %d\n", mi.NumInJacks, mi.NumOutJacks)
			if mi.EndpointIn != 0 {
				fmt.Printf("  Input Endpoint: 0x%02x\n", mi.EndpointIn)
			}
			if mi.EndpointOut != 0 {
				fmt.Printf("  Output Endpoint: 0x%02x\n", mi.EndpointOut)
			}
			if mi.NumCables > 0 {
				fmt.Printf("  Number of cables: %d\n", mi.NumCables)
			}
		}
	}
	
	// If probe-only mode, exit after displaying info
	if *probeOnly {
		fmt.Println("\nProbe complete.")
		return
	}

	// Find the streaming interface that best matches the requested sampling frequency
	var streamInterface *transfers.AudioStreamingInterface
	var exactMatch bool

	// First, try to find an exact match for the requested frequency
	for _, si := range info.StreamingInterfaces {
		if si.NrChannels > 0 && len(si.SamplingFreqs) > 0 {
			for _, freq := range si.SamplingFreqs {
				if freq == uint32(*samplingFreq) {
					streamInterface = si
					exactMatch = true
					break
				}
			}
			if exactMatch {
				break
			}
		}
	}

	// If no exact match, find the closest frequency
	if streamInterface == nil {
		var closestDiff uint32 = ^uint32(0) // Max uint32
		for _, si := range info.StreamingInterfaces {
			if si.NrChannels > 0 && len(si.SamplingFreqs) > 0 {
				for _, freq := range si.SamplingFreqs {
					diff := freq - uint32(*samplingFreq)
					if uint32(*samplingFreq) > freq {
						diff = uint32(*samplingFreq) - freq
					}
					if diff < closestDiff {
						closestDiff = diff
						streamInterface = si
					}
				}
			}
		}
	}

	if streamInterface == nil {
		log.Fatal("No valid audio streaming interface found")
	}

	fmt.Printf("Using interface %d, alternate setting %d\n",
		streamInterface.InterfaceNumber(),
		streamInterface.AlternateSetting())
	fmt.Printf("Audio format: %d channels, %d-bit, supported frequencies: %v Hz\n",
		streamInterface.NrChannels,
		streamInterface.BitResolution,
		streamInterface.SamplingFreqs)

	// Check if requested frequency is supported
	freqSupported := false
	for _, freq := range streamInterface.SamplingFreqs {
		if freq == uint32(*samplingFreq) {
			freqSupported = true
			break
		}
	}
	if !freqSupported && len(streamInterface.SamplingFreqs) > 0 {
		fmt.Printf("Warning: Requested frequency %d Hz not supported, using %d Hz\n",
			*samplingFreq, streamInterface.SamplingFreqs[0])
		*samplingFreq = uint(streamInterface.SamplingFreqs[0])
	}

	// Claim the audio reader
	reader, err := streamInterface.ClaimAudioReader(uint32(*samplingFreq))
	if err != nil {
		log.Fatalf("Failed to claim audio reader: %v", err)
	}
	defer reader.Stop()

	// Create output file
	outFile, err := os.Create(*outputFile)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer outFile.Close()

	// Write WAV header (optional, for easier playback)
	format := reader.GetAudioFormat()
	if err := writeWAVHeader(outFile, &format, *duration); err != nil {
		log.Printf("Warning: Failed to write WAV header: %v", err)
	}

	// Start audio streaming
	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	// Handle interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nStopping audio capture...")
		cancel()
	}()

	// Start the reader
	if err := reader.Start(ctx); err != nil {
		log.Fatalf("Failed to start audio reader: %v", err)
	}

	// Get the actual audio format from the device
	actualFormat, err := streamInterface.GetActualAudioFormat()
	if err != nil {
		log.Printf("Warning: Could not read actual audio format: %v", err)
	} else {
		// Check if any parameters differ from expected
		if actualFormat.SampleRate != uint32(*samplingFreq) {
			fmt.Printf("WARNING: Device is using %d Hz instead of requested %d Hz\n",
				actualFormat.SampleRate, *samplingFreq)
		}
		if actualFormat.Channels != format.Channels {
			fmt.Printf("WARNING: Device is using %d channels instead of expected %d\n",
				actualFormat.Channels, format.Channels)
		}
		if actualFormat.BitsPerSample != format.BitsPerSample {
			fmt.Printf("WARNING: Device is using %d bits instead of expected %d\n",
				actualFormat.BitsPerSample, format.BitsPerSample)
		}

		// Use the actual format for the WAV file
		format = actualFormat

		// Rewrite WAV header with actual format
		outFile.Seek(0, 0)
		if err := writeWAVHeader(outFile, &format, *duration); err != nil {
			log.Printf("Warning: Failed to rewrite WAV header: %v", err)
		}
	}

	fmt.Printf("Recording audio for %v...\n", *duration)
	fmt.Printf("Audio format: %d channels, %d Hz, %d bits\n",
		format.Channels, format.SampleRate, format.BitsPerSample)

	totalBytes := 0
	startTime := time.Now()

	// Read audio data
	for {
		select {
		case <-ctx.Done():
			elapsed := time.Since(startTime)
			fmt.Printf("\nRecording complete. Captured %d bytes in %v\n", totalBytes, elapsed)

			// Update WAV header with actual data size
			if err := updateWAVHeader(outFile, totalBytes); err != nil {
				log.Printf("Warning: Failed to update WAV header: %v", err)
			}
			return

		case audioData := <-reader.Read():
			if len(audioData) > 0 {
				n, err := outFile.Write(audioData)
				if err != nil {
					log.Printf("Error writing audio data: %v", err)
					continue
				}
				totalBytes += n

				// Print progress
				if totalBytes%(format.Channels*format.BitsPerSample/8*int(format.SampleRate)/10) == 0 {
					fmt.Printf(".")
				}
			}
		}
	}
}

func openUSBDevice(path string) (int, error) {
	fd, err := syscall.Open(path, syscall.O_RDWR, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to open device %s: %w", path, err)
	}
	return fd, nil
}

func writeWAVHeader(file *os.File, format *AudioFormat, duration time.Duration) error {
	// Estimate data size
	dataSize := uint32(float64(format.SampleRate) * duration.Seconds() * float64(format.Channels) * float64(format.BitsPerSample/8))

	// RIFF header
	file.Write([]byte("RIFF"))
	binary.Write(file, binary.LittleEndian, uint32(dataSize+36)) // File size - 8
	file.Write([]byte("WAVE"))

	// fmt chunk
	file.Write([]byte("fmt "))
	binary.Write(file, binary.LittleEndian, uint32(16)) // Chunk size
	binary.Write(file, binary.LittleEndian, uint16(1))  // Audio format (PCM)
	binary.Write(file, binary.LittleEndian, uint16(format.Channels))
	binary.Write(file, binary.LittleEndian, uint32(format.SampleRate))
	byteRate := uint32(format.SampleRate) * uint32(format.Channels) * uint32(format.BitsPerSample/8)
	binary.Write(file, binary.LittleEndian, byteRate)
	blockAlign := uint16(format.Channels) * uint16(format.BitsPerSample/8)
	binary.Write(file, binary.LittleEndian, blockAlign)
	binary.Write(file, binary.LittleEndian, uint16(format.BitsPerSample))

	// data chunk
	file.Write([]byte("data"))
	binary.Write(file, binary.LittleEndian, dataSize)

	return nil
}

func updateWAVHeader(file *os.File, actualDataSize int) error {
	// Update RIFF chunk size
	file.Seek(4, 0)
	binary.Write(file, binary.LittleEndian, uint32(actualDataSize+36))

	// Update data chunk size
	file.Seek(40, 0)
	binary.Write(file, binary.LittleEndian, uint32(actualDataSize))

	return nil
}
