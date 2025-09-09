package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	uvc "github.com/kevmo314/go-uvc"
)

func main() {
	path := flag.String("path", "/dev/bus/usb/001/007", "path to the usb device")
	duration := flag.Duration("duration", 5*time.Second, "recording duration")
	flag.Parse()

	fd, err := os.OpenFile(*path, os.O_RDWR, 0)
	if err != nil {
		panic(err)
	}
	defer fd.Close()

	dev, err := uvc.NewUACDevice(fd.Fd())
	if err != nil {
		panic(err)
	}

	info, err := dev.DeviceInfo()
	if err != nil {
		panic(err)
	}

	if len(info.StreamingInterfaces) == 0 {
		log.Fatal("No audio streaming interfaces found")
	}

	// Use the first interface
	iface := info.StreamingInterfaces[0]

	fmt.Printf("Using interface %d with %d channels, %d-bit audio\n",
		iface.InterfaceNumber(), iface.NrChannels, iface.BitResolution)
	fmt.Printf("Available sampling frequencies: %v Hz\n", iface.SamplingFreqs)

	// Pick the first available frequency
	if len(iface.SamplingFreqs) == 0 {
		log.Fatal("No sampling frequencies available")
	}
	freq := iface.SamplingFreqs[0]

	fmt.Printf("Starting audio capture at %d Hz...\n", freq)

	reader, err := iface.ClaimAudioReader(freq)
	if err != nil {
		log.Fatalf("Failed to claim audio reader: %v", err)
	}
	defer reader.Close()

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	format := reader.GetAudioFormat()
	fmt.Printf("Recording format: %d channels, %d Hz, %d bits\n",
		format.Channels, format.SampleRate, format.BitsPerSample)

	// Simple level meter
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	totalBytes := 0
	maxLevel := float32(0)

	fmt.Println("\nAudio Level Meter (# = signal level):")
	fmt.Println("0%                    50%                   100%")
	fmt.Println("|---------------------|---------------------|")

	// Create a buffer for reading audio data
	buf := make([]byte, 8192)

	// Create a channel for audio data processing
	audioChan := make(chan []byte, 10)

	// Start a goroutine to read audio data
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := reader.ReadAudio(buf)
				if err != nil {
					log.Printf("Error reading audio: %v", err)
					continue
				}
				if n > 0 {
					data := make([]byte, n)
					copy(data, buf[:n])
					select {
					case audioChan <- data:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("\n\nRecording complete. Total bytes: %d\n", totalBytes)
			return

		case audioData := <-audioChan:
			if len(audioData) > 0 {
				totalBytes += len(audioData)

				// Calculate peak level for this buffer
				bytesPerSample := format.BitsPerSample / 8
				numSamples := len(audioData) / (bytesPerSample * format.Channels)

				bufferMax := float32(0)
				for i := 0; i < numSamples; i++ {
					var sample float32

					if format.BitsPerSample == 16 {
						sampleIdx := i * format.Channels * bytesPerSample
						if sampleIdx+1 < len(audioData) {
							s16 := int16(audioData[sampleIdx]) | (int16(audioData[sampleIdx+1]) << 8)
							sample = float32(math.Abs(float64(s16))) / 32768.0
						}
					}

					if sample > bufferMax {
						bufferMax = sample
					}
				}

				// Update max level with decay
				maxLevel = maxLevel * 0.95
				if bufferMax > maxLevel {
					maxLevel = bufferMax
				}
			}

		case <-ticker.C:
			// Draw level meter
			barLength := int(maxLevel * 45)
			bar := ""
			for i := 0; i < 45; i++ {
				if i < barLength {
					bar += "#"
				} else {
					bar += " "
				}
			}

			// Calculate dB
			db := float64(-120)
			if maxLevel > 0 {
				db = 20 * math.Log10(float64(maxLevel))
			}

			fmt.Printf("\r|%s| %.1f dB", bar, db)
		}
	}
}
