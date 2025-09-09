package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/cmplx"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gdamore/tcell/v2"
	uvc "github.com/kevmo314/go-uvc"
	"github.com/kevmo314/go-uvc/pkg/transfers"
	"github.com/mjibson/go-dsp/fft"
	"github.com/rivo/tview"
)

type WaveformDisplay struct {
	samples       []float32
	sampleRate    int
	channels      int
	maxSamples    int
	writeIndex    int
	isRecording   atomic.Bool
	peakLevel     float32
	rmsLevel      float32
	totalBytes    int64
	startTime     time.Time
	fftSize       int
	frequencyBins []float64
}

func NewWaveformDisplay(sampleRate, channels int) *WaveformDisplay {
	maxSamples := sampleRate * 2 // 2 seconds of samples
	fftSize := 2048              // Power of 2 for FFT
	return &WaveformDisplay{
		samples:       make([]float32, maxSamples),
		sampleRate:    sampleRate,
		channels:      channels,
		maxSamples:    maxSamples,
		startTime:     time.Now(),
		fftSize:       fftSize,
		frequencyBins: make([]float64, fftSize/2),
	}
}

func (w *WaveformDisplay) AddSamples(data []byte, bitsPerSample int) {
	if !w.isRecording.Load() {
		return
	}

	w.totalBytes += int64(len(data))
	bytesPerSample := bitsPerSample / 8
	numSamples := len(data) / (bytesPerSample * w.channels)

	var rmsSum float32
	var peakMax float32

	for i := 0; i < numSamples && w.writeIndex < w.maxSamples; i++ {
		var sample float32

		// Convert sample to float32 based on bit depth
		switch bitsPerSample {
		case 16:
			// 16-bit signed samples
			sampleIdx := i * w.channels * bytesPerSample
			if sampleIdx+1 < len(data) {
				s16 := int16(data[sampleIdx]) | (int16(data[sampleIdx+1]) << 8)
				sample = float32(s16) / 32768.0
			}
		case 24:
			// 24-bit signed samples
			sampleIdx := i * w.channels * bytesPerSample
			if sampleIdx+2 < len(data) {
				s24 := int32(data[sampleIdx]) | (int32(data[sampleIdx+1]) << 8) | (int32(data[sampleIdx+2]) << 16)
				if s24&0x800000 != 0 { // Sign extend
					s24 |= ^0xFFFFFF // Sign extend for 24-bit
				}
				sample = float32(s24) / 8388608.0
			}
		case 32:
			// 32-bit signed samples
			sampleIdx := i * w.channels * bytesPerSample
			if sampleIdx+3 < len(data) {
				s32 := int32(data[sampleIdx]) | (int32(data[sampleIdx+1]) << 8) |
					(int32(data[sampleIdx+2]) << 16) | (int32(data[sampleIdx+3]) << 24)
				sample = float32(s32) / 2147483648.0
			}
		default:
			sample = 0
		}

		// Update statistics
		absVal := float32(math.Abs(float64(sample)))
		if absVal > peakMax {
			peakMax = absVal
		}
		rmsSum += sample * sample

		w.samples[w.writeIndex] = sample
		w.writeIndex = (w.writeIndex + 1) % w.maxSamples
	}

	// Update running statistics
	if numSamples > 0 {
		currentRMS := float32(math.Sqrt(float64(rmsSum / float32(numSamples))))
		// Exponential moving average
		w.peakLevel = w.peakLevel*0.95 + peakMax*0.05
		w.rmsLevel = w.rmsLevel*0.95 + currentRMS*0.05
	}
}

func (w *WaveformDisplay) GetWaveform(width, height int) []string {
	if len(w.samples) == 0 || width == 0 || height == 0 {
		return make([]string, height)
	}

	// Create a 2D grid to build the display
	grid := make([][]rune, height)
	for i := range grid {
		grid[i] = make([]rune, width)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	// Draw center line (zero level)
	centerY := height / 2
	if centerY >= 0 && centerY < height {
		for x := 0; x < width; x++ {
			grid[centerY][x] = '─'
		}
	}

	samplesPerCol := w.maxSamples / width
	if samplesPerCol == 0 {
		samplesPerCol = 1
	}

	for x := 0; x < width; x++ {
		startIdx := x * samplesPerCol
		if startIdx >= len(w.samples) {
			break
		}

		endIdx := startIdx + samplesPerCol
		if endIdx > len(w.samples) {
			endIdx = len(w.samples)
		}

		// Find min and max in this column
		var minVal, maxVal float32 = 0, 0
		for i := startIdx; i < endIdx; i++ {
			val := w.samples[i]
			if val < minVal {
				minVal = val
			}
			if val > maxVal {
				maxVal = val
			}
		}

		// Convert to screen coordinates (flip Y so 0 is at top)
		minY := height - 1 - int((minVal+1)*float32(height-1)/2)
		maxY := height - 1 - int((maxVal+1)*float32(height-1)/2)

		// Clamp values
		if minY < 0 {
			minY = 0
		}
		if minY >= height {
			minY = height - 1
		}
		if maxY < 0 {
			maxY = 0
		}
		if maxY >= height {
			maxY = height - 1
		}

		// Swap if needed (maxY should be smaller index since Y is flipped)
		if maxY > minY {
			maxY, minY = minY, maxY
		}

		// Draw vertical line from maxY to minY
		for y := maxY; y <= minY; y++ {
			if y >= 0 && y < height && x >= 0 && x < width {
				// Determine which character to use
				if math.Abs(float64(maxVal-minVal)) > 0.5 {
					grid[y][x] = '║'
				} else if y != centerY || grid[y][x] == ' ' {
					// Don't overwrite center line unless it's empty
					grid[y][x] = '│'
				}
			}
		}
	}

	// Convert grid to strings, ensuring proper width
	lines := make([]string, height)
	for i := range grid {
		// Ensure line doesn't exceed width
		if len(grid[i]) > width {
			lines[i] = string(grid[i][:width])
		} else {
			lines[i] = string(grid[i])
		}
	}

	return lines
}

func (w *WaveformDisplay) GetStatistics() string {
	if !w.isRecording.Load() {
		return "Not recording"
	}

	duration := time.Since(w.startTime)
	bitRate := float64(w.totalBytes*8) / duration.Seconds()

	// Convert peak and RMS to dB
	peakDB := float64(-120)
	rmsDB := float64(-120)
	if w.peakLevel > 0 {
		peakDB = 20 * math.Log10(float64(w.peakLevel))
	}
	if w.rmsLevel > 0 {
		rmsDB = 20 * math.Log10(float64(w.rmsLevel))
	}

	return fmt.Sprintf(`Duration: %v
Data Rate: %.1f kbps
Peak Level: %.1f dB
RMS Level: %.1f dB
Buffer: %d/%d samples`,
		duration.Round(time.Second),
		bitRate/1000,
		peakDB,
		rmsDB,
		w.writeIndex,
		w.maxSamples)
}

func (w *WaveformDisplay) UpdateFrequencySpectrum() {
	if w.writeIndex < w.fftSize {
		return // Not enough samples yet
	}

	// Get the most recent samples for FFT
	startIdx := w.writeIndex - w.fftSize
	if startIdx < 0 {
		startIdx = 0
	}

	// Convert samples to complex numbers for FFT
	fftInput := make([]complex128, w.fftSize)
	for i := 0; i < w.fftSize && startIdx+i < len(w.samples); i++ {
		// Apply Hamming window to reduce spectral leakage
		window := 0.54 - 0.46*math.Cos(2*math.Pi*float64(i)/float64(w.fftSize-1))
		fftInput[i] = complex(float64(w.samples[startIdx+i])*window, 0)
	}

	// Perform FFT
	fftOutput := fft.FFT(fftInput)

	// Calculate magnitude spectrum (only first half, as it's symmetric)
	for i := 0; i < len(w.frequencyBins); i++ {
		magnitude := cmplx.Abs(fftOutput[i])
		// Convert to dB scale
		if magnitude > 0 {
			w.frequencyBins[i] = 20 * math.Log10(magnitude)
		} else {
			w.frequencyBins[i] = -120
		}
	}
}

func (w *WaveformDisplay) GetFrequencySpectrum(width, height int) []string {
	if len(w.frequencyBins) == 0 || width == 0 || height == 0 {
		return make([]string, height)
	}

	// Update spectrum
	w.UpdateFrequencySpectrum()

	// Create a 2D grid for the spectrum display
	grid := make([][]rune, height)
	for i := range grid {
		grid[i] = make([]rune, width)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	// Number of frequency bins to display
	binsToShow := len(w.frequencyBins) / 4 // Show lower frequencies (up to Nyquist/4)
	if binsToShow > width {
		binsToShow = width
	}

	// Bin width for display
	binsPerColumn := binsToShow / width
	if binsPerColumn == 0 {
		binsPerColumn = 1
	}

	// Find min and max for scaling
	minDB := -60.0
	maxDB := 0.0

	for x := 0; x < width && x*binsPerColumn < binsToShow; x++ {
		// Average bins for this column
		sum := 0.0
		count := 0
		for i := 0; i < binsPerColumn && x*binsPerColumn+i < binsToShow; i++ {
			sum += w.frequencyBins[x*binsPerColumn+i]
			count++
		}

		if count > 0 {
			avgDB := sum / float64(count)

			// Scale to display height
			normalized := (avgDB - minDB) / (maxDB - minDB)
			if normalized < 0 {
				normalized = 0
			}
			if normalized > 1 {
				normalized = 1
			}

			barHeight := int(normalized * float64(height-1))

			// Draw vertical bar from bottom
			for y := 0; y <= barHeight; y++ {
				displayY := height - 1 - y
				if displayY >= 0 && displayY < height {
					if y == barHeight {
						grid[displayY][x] = '█'
					} else if y > barHeight*2/3 {
						grid[displayY][x] = '▓'
					} else if y > barHeight/3 {
						grid[displayY][x] = '▒'
					} else {
						grid[displayY][x] = '░'
					}
				}
			}
		}
	}

	// Add frequency labels at the bottom
	if height > 1 {
		// Add Hz labels at key positions
		labelPositions := []struct {
			freq int
			pos  int
		}{
			{100, width * 100 * 2 / w.sampleRate},
			{1000, width * 1000 * 2 / w.sampleRate},
			{5000, width * 5000 * 2 / w.sampleRate},
			{10000, width * 10000 * 2 / w.sampleRate},
		}

		for _, label := range labelPositions {
			if label.pos >= 0 && label.pos < width {
				grid[height-1][label.pos] = '|'
			}
		}
	}

	// Convert grid to strings
	lines := make([]string, height)
	for i := range grid {
		lines[i] = string(grid[i])
	}

	return lines
}

func main() {
	path := flag.String("path", "", "path to the usb device")
	flag.Parse()

	if *path == "" {
		fmt.Println("Error: Please specify a USB device path with -path flag")
		fmt.Println("Example: uac_inspect -path /dev/bus/usb/001/007")
		fmt.Println("\nAvailable USB devices:")
		fmt.Println("Run: lsusb")
		os.Exit(1)
	}

	fd, err := os.OpenFile(*path, os.O_RDWR, 0)
	if err != nil {
		fmt.Printf("Error opening device %s: %v\n", *path, err)
		fmt.Println("Make sure to run with sudo if permission denied")
		os.Exit(1)
	}
	defer fd.Close()

	dev, err := uvc.NewUACDevice(fd.Fd())
	if err != nil {
		fmt.Printf("Error creating UAC device: %v\n", err)
		os.Exit(1)
	}

	info, err := dev.DeviceInfo()
	if err != nil {
		fmt.Printf("Error getting device info: %v\n", err)
		os.Exit(1)
	}

	// Debug output
	fmt.Printf("Found %d audio streaming interfaces\n", len(info.StreamingInterfaces))
	fmt.Printf("Found %d MIDI interfaces\n", len(info.MIDIInterfaces))

	if len(info.StreamingInterfaces) == 0 && len(info.MIDIInterfaces) == 0 {
		fmt.Println("\nNo audio or MIDI interfaces found on this device.")
		fmt.Println("This might not be an audio device, or it might not be UAC-compliant.")
		fmt.Println("\nTry testing with a known USB audio device like:")
		fmt.Println("  - USB headset")
		fmt.Println("  - Webcam with built-in microphone")
		fmt.Println("  - USB sound card")
		os.Exit(1)
	}

	app := tview.NewApplication()

	streamingIfaces := tview.NewList()
	streamingIfaces.SetBorder(true).SetTitle("Audio Streaming Interfaces")

	midiIfaces := tview.NewList()
	midiIfaces.SetBorder(true).SetTitle("MIDI Interfaces")

	controlRequests := tview.NewList().ShowSecondaryText(false)
	controlRequests.SetBorder(true).SetTitle("Audio Controls")
	controlRequests.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			controlRequests.Clear()
			app.SetFocus(streamingIfaces)
			return nil
		}
		return event
	})

	leftColumn := tview.NewFlex().SetDirection(tview.FlexRow)
	leftColumn.AddItem(streamingIfaces, 0, 2, true)
	if len(info.MIDIInterfaces) > 0 {
		leftColumn.AddItem(midiIfaces, 0, 1, false)
	}

	formats := tview.NewList()
	formats.SetBorder(true).SetTitle("Audio Formats")

	waveformView := tview.NewTextView()
	waveformView.SetBorder(true).SetTitle("Audio Waveform")
	waveformView.SetDynamicColors(true)

	// Frequency spectrum view
	spectrumView := tview.NewTextView()
	spectrumView.SetBorder(true).SetTitle("Frequency Spectrum (Hz)")
	spectrumView.SetDynamicColors(true)

	// Audio stats view
	statsView := tview.NewTextView()
	statsView.SetBorder(true).SetTitle("Audio Statistics")
	statsView.SetDynamicColors(true)

	logText := tview.NewTextView()
	logText.SetMaxLines(10).SetBorder(true).SetTitle("Log")
	log.SetOutput(logText)

	// Log initial detection
	log.Printf("Detected %d audio interfaces, %d MIDI interfaces",
		len(info.StreamingInterfaces), len(info.MIDIInterfaces))

	// Create a flex for waveform and spectrum side by side
	visualizationFlex := tview.NewFlex().SetDirection(tview.FlexColumn)
	visualizationFlex.AddItem(waveformView, 0, 1, false)
	visualizationFlex.AddItem(spectrumView, 0, 1, false)

	rightColumn := tview.NewFlex().SetDirection(tview.FlexRow)
	rightColumn.AddItem(formats, 0, 1, false)
	rightColumn.AddItem(controlRequests, 0, 1, false)
	rightColumn.AddItem(visualizationFlex, 0, 2, false)
	rightColumn.AddItem(statsView, 4, 0, false)

	var currentWaveform *WaveformDisplay
	var currentReader *transfers.AudioReader

	active := &atomic.Uint32{}

	// Populate streaming interfaces
	for i, si := range info.StreamingInterfaces {
		interfaceTitle := fmt.Sprintf("Interface %d (Alt %d)", si.InterfaceNumber(), si.AlternateSetting())
		interfaceSubtitle := fmt.Sprintf("%d channels, %d-bit, %s",
			si.NrChannels, si.BitResolution, formatAudioType(si))

		streamingIfaces.AddItem(interfaceTitle, interfaceSubtitle, 0, func() {
			currentInterface := info.StreamingInterfaces[i]

			// Clear and populate formats
			formats.Clear()

			// Group sampling frequencies for better display
			freqGroups := make(map[string][]uint32)
			for _, freq := range currentInterface.SamplingFreqs {
				key := fmt.Sprintf("%dch %dbit", currentInterface.NrChannels, currentInterface.BitResolution)
				freqGroups[key] = append(freqGroups[key], freq)
			}

			// Sort frequencies
			for _, freqs := range freqGroups {
				sort.Slice(freqs, func(i, j int) bool {
					return freqs[i] < freqs[j]
				})
			}

			// Add format entries
			for key, freqs := range freqGroups {
				freqList := make([]string, len(freqs))
				for i, f := range freqs {
					if f >= 1000 {
						freqList[i] = fmt.Sprintf("%.1fkHz", float32(f)/1000.0)
					} else {
						freqList[i] = fmt.Sprintf("%dHz", f)
					}
				}

				formatTitle := fmt.Sprintf("%s - %s", key, formatTagName(currentInterface.FormatTag))
				formatSubtitle := strings.Join(freqList, ", ")

				formats.AddItem(formatTitle, formatSubtitle, 0, func() {
					// Stop any current recording
					if currentReader != nil {
						currentReader.Close()
						currentReader = nil
					}
					if currentWaveform != nil {
						currentWaveform.isRecording.Store(false)
					}

					// Pick a good default frequency (48kHz if available, otherwise highest)
					selectedFreq := freqs[len(freqs)-1] // Default to highest
					for _, f := range freqs {
						if f == 48000 {
							selectedFreq = f
							break
						}
					}

					track := active.Add(1)

					// Start new recording
					reader, err := currentInterface.ClaimAudioReader(selectedFreq)
					if err != nil {
						log.Printf("Error claiming audio reader: %v", err)
						return
					}

					currentReader = reader
					currentWaveform = NewWaveformDisplay(int(selectedFreq), int(currentInterface.NrChannels))
					currentWaveform.isRecording.Store(true)

					// Get actual format
					actualFormat, err := currentInterface.GetActualAudioFormat()
					if err != nil {
						log.Printf("Warning: Could not read actual format: %v", err)
						actualFormat = reader.GetAudioFormat()
					}

					log.Printf("Started recording: %dch, %dHz, %dbit",
						actualFormat.Channels, actualFormat.SampleRate, actualFormat.BitsPerSample)

					// Start reading in background
					go func() {
						defer reader.Close()

						// Create a buffer for reading audio
						buf := make([]byte, 8192)

						// Update waveform display
						go func() {
							ticker := time.NewTicker(50 * time.Millisecond)
							defer ticker.Stop()

							for active.Load() == track {
								select {
								case <-ticker.C:
									if currentWaveform != nil && currentWaveform.isRecording.Load() {
										// Update waveform
										x, y, width, height := waveformView.GetInnerRect()
										_ = x
										_ = y
										if width > 0 && height > 0 {
											lines := currentWaveform.GetWaveform(width, height)
											waveformView.Clear()
											// Join all lines with newlines as a single string
											waveformText := strings.Join(lines, "\n")
											waveformView.SetText(waveformText)
										}

										// Update frequency spectrum
										sx, sy, swidth, sheight := spectrumView.GetInnerRect()
										_ = sx
										_ = sy
										if swidth > 0 && sheight > 0 {
											specLines := currentWaveform.GetFrequencySpectrum(swidth, sheight)
											spectrumView.Clear()
											spectrumText := strings.Join(specLines, "\n")
											spectrumView.SetText(spectrumText)
										}

										// Update statistics
										statsView.Clear()
										statsView.SetText(currentWaveform.GetStatistics())

										app.ForceDraw()
									}
								}
							}
						}()

						// Read audio data
						for active.Load() == track {
							n, err := reader.ReadAudio(buf)
							if err != nil {
								log.Printf("Error reading audio: %v", err)
								continue
							}
							if n > 0 && currentWaveform != nil {
								// Copy the data to avoid overwriting
								audioData := make([]byte, n)
								copy(audioData, buf[:n])
								currentWaveform.AddSamples(audioData, int(actualFormat.BitsPerSample))
							}
						}
					}()
				})
			}

			app.SetFocus(formats)
		})
	}

	// Populate MIDI interfaces
	for i, mi := range info.MIDIInterfaces {
		midiTitle := fmt.Sprintf("MIDI Interface %d", mi.InterfaceNumber())
		midiSubtitle := fmt.Sprintf("%d in, %d out jacks", mi.NumInJacks, mi.NumOutJacks)

		midiIfaces.AddItem(midiTitle, midiSubtitle, 0, func() {
			_ = i // Use MIDI interface if needed
			log.Printf("Selected MIDI interface %d", mi.InterfaceNumber())
		})
	}

	// Add audio control options
	if len(info.StreamingInterfaces) > 0 {
		// Use first interface for control examples
		iface := info.StreamingInterfaces[0]
		control := transfers.NewUACControl(
			info.GetHandle(),
			iface.InterfaceNumber(),
		)

		addControlOptions(controlRequests, control, app)
	}

	// Create the layout
	flex := tview.NewFlex().
		AddItem(leftColumn, 0, 1, true).
		AddItem(rightColumn, 0, 2, false)

	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(flex, 0, 1, true).
		AddItem(logText, 8, 0, false)

	if err := app.SetRoot(root, true).Run(); err != nil {
		panic(err)
	}
}

func formatAudioType(si *transfers.AudioStreamingInterface) string {
	switch si.FormatType {
	case 0x01:
		return "PCM"
	case 0x02:
		return "Dynamic/MPEG"
	case 0x03:
		return "Format Specific"
	default:
		return "Unknown"
	}
}

func formatTagName(tag uint16) string {
	switch tag {
	case 0x0001:
		return "PCM"
	case 0x0002:
		return "PCM8"
	case 0x0003:
		return "IEEE_FLOAT"
	case 0x0004:
		return "ALAW"
	case 0x0005:
		return "MULAW"
	case 0x1001:
		return "MPEG"
	case 0x1002:
		return "AC3"
	default:
		return fmt.Sprintf("0x%04x", tag)
	}
}

func addControlOptions(controlRequests *tview.List, control *transfers.UACControl, app *tview.Application) {
	// Volume control
	controlRequests.AddItem("Get Volume", "", 0, func() {
		volume, err := control.GetVolume(2, 0) // Unit 2, Master channel
		if err != nil {
			log.Printf("Failed to get volume: %v", err)
		} else {
			log.Printf("Current volume: %.2f dB", float32(volume)/256.0)
		}
	})

	controlRequests.AddItem("Set Volume", "", 0, func() {
		// Simple volume setting - this could be enhanced with an input field
		volumeDB := int16(-10 * 256) // -10 dB
		err := control.SetVolume(2, 0, volumeDB)
		if err != nil {
			log.Printf("Failed to set volume: %v", err)
		} else {
			log.Printf("Volume set to -10 dB")
		}
	})

	controlRequests.AddItem("Get Mute Status", "", 0, func() {
		muted, err := control.GetMute(2, 0)
		if err != nil {
			log.Printf("Failed to get mute status: %v", err)
		} else {
			log.Printf("Mute status: %v", muted)
		}
	})

	controlRequests.AddItem("Toggle Mute", "", 0, func() {
		// Get current status and toggle
		muted, err := control.GetMute(2, 0)
		if err != nil {
			log.Printf("Failed to get mute status: %v", err)
			return
		}

		err = control.SetMute(2, 0, !muted)
		if err != nil {
			log.Printf("Failed to toggle mute: %v", err)
		} else {
			log.Printf("Mute toggled to: %v", !muted)
		}
	})

	controlRequests.AddItem("Get Volume Range", "", 0, func() {
		min, max, res, err := control.GetVolumeRange(2, 0)
		if err != nil {
			log.Printf("Failed to get volume range: %v", err)
		} else {
			log.Printf("Volume range: %.2f to %.2f dB (step: %.2f dB)",
				float32(min)/256.0, float32(max)/256.0, float32(res)/256.0)
		}
	})
}
