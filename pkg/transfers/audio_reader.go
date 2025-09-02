package transfers

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

/*
#cgo LDFLAGS: -lusb-1.0
#include <libusb-1.0/libusb.h>
#include <stdlib.h>
#include <stdint.h>

// Forward declaration
void audio_transfer_callback(struct libusb_transfer *transfer);

static struct libusb_transfer* alloc_audio_transfer(int num_iso_packets) {
	return libusb_alloc_transfer(num_iso_packets);
}

static void submit_audio_transfer(struct libusb_transfer *transfer) {
	libusb_submit_transfer(transfer);
}

static void setup_audio_iso_transfer(
	struct libusb_transfer *transfer,
	struct libusb_device_handle *dev_handle,
	unsigned char endpoint,
	unsigned char *buffer,
	int length,
	int num_iso_packets,
	uintptr_t user_data,
	unsigned int timeout
) {
	libusb_fill_iso_transfer(transfer, dev_handle, endpoint, buffer, length,
		num_iso_packets, (libusb_transfer_cb_fn)audio_transfer_callback, (void*)user_data, timeout);
	libusb_set_iso_packet_lengths(transfer, length / num_iso_packets);
}
*/
import "C"

var (
	audioReaderRegistry = make(map[uintptr]*AudioReader)
	audioReaderMutex    sync.Mutex
	audioReaderCounter  uintptr
)

type AudioReader struct {
	id             uintptr
	asi            *AudioStreamingInterface
	transfers      []*C.struct_libusb_transfer
	buffers        []unsafe.Pointer // C-allocated buffers
	bufferSizes    []int
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	mu             sync.Mutex
	audioBuffer    chan []byte
	running        bool
	transferCount  int64 // Debug: count callbacks
	bytesReceived  int64 // Debug: total bytes received
	packetsSuccess int64 // Debug: successful packets
	packetsError   int64 // Debug: error packets
	packetsEmpty   int64 // Debug: empty packets
}

const (
	numAudioTransfers = 8
	numIsoPackets     = 8    // Reduced from 10 to match USB frame timing better
	audioTimeout      = 5000 // Increased timeout
)

func NewAudioReader(asi *AudioStreamingInterface) *AudioReader {
	audioReaderMutex.Lock()
	defer audioReaderMutex.Unlock()

	id := atomic.AddUintptr(&audioReaderCounter, 1)
	reader := &AudioReader{
		id:          id,
		asi:         asi,
		audioBuffer: make(chan []byte, 1000), // Increased buffer size
	}
	audioReaderRegistry[id] = reader
	return reader
}

//export audio_transfer_callback
func audio_transfer_callback(transfer *C.struct_libusb_transfer) {
	id := uintptr(transfer.user_data)
	audioReaderMutex.Lock()
	reader, ok := audioReaderRegistry[id]
	audioReaderMutex.Unlock()
	if ok {
		reader.handleTransfer(transfer)
	} else {
		fmt.Printf("Warning: transfer callback for unknown reader id %d\n", id)
	}
}

func (ar *AudioReader) handleTransfer(transfer *C.struct_libusb_transfer) {
	atomic.AddInt64(&ar.transferCount, 1)

	if transfer.status != C.LIBUSB_TRANSFER_COMPLETED {
		if transfer.status != C.LIBUSB_TRANSFER_CANCELLED {
			// Resubmit transfer if not cancelled
			C.libusb_submit_transfer(transfer)
		}
		return
	}

	// Process ISO packets
	numPackets := int(transfer.num_iso_packets)
	if numPackets > 0 {
		totalBytes := 0
		emptyPackets := 0
		errorPackets := 0

		// Access the iso_packet_desc array using the same method as UVC
		descs := unsafe.Slice(unsafe.SliceData(transfer.iso_packet_desc[:]), transfer.num_iso_packets)

		for i := 0; i < numPackets; i++ {
			packet := descs[i]

			// Check transfer status
			if packet.status != C.LIBUSB_TRANSFER_COMPLETED {
				// Skip packets with errors
				errorPackets++
				atomic.AddInt64(&ar.packetsError, 1)
				continue
			}

			// Skip empty packets
			if packet.actual_length == 0 {
				emptyPackets++
				atomic.AddInt64(&ar.packetsEmpty, 1)
				continue
			}

			atomic.AddInt64(&ar.packetsSuccess, 1)

			// Get the packet buffer using libusb function (like UVC does)
			pktbuf := C.libusb_get_iso_packet_buffer_simple(transfer, C.uint(i))
			if pktbuf == nil {
				continue
			}

			// Extract audio data from the packet
			data := C.GoBytes(unsafe.Pointer(pktbuf), C.int(packet.actual_length))
			totalBytes += len(data)
			atomic.AddInt64(&ar.bytesReceived, int64(len(data)))

			// Send to audio buffer (non-blocking)
			select {
			case ar.audioBuffer <- data:
			default:
				// Drop frame if buffer is full (silently)
			}
		}

		// No need for debug output anymore
	}

	// Resubmit the transfer
	if ar.running {
		if ret := C.libusb_submit_transfer(transfer); ret < 0 {
			fmt.Printf("Error resubmitting transfer: %s\n", C.GoString(C.libusb_error_name(ret)))
		}
	}
}

func (ar *AudioReader) Start(ctx context.Context) error {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	if ar.running {
		return fmt.Errorf("audio reader already running")
	}

	ar.ctx, ar.cancel = context.WithCancel(ctx)
	ar.running = true

	// Calculate packet size based on audio format
	packetSize := int(ar.asi.MaxPacketSize)
	bufferSize := packetSize * numIsoPackets

	fmt.Printf("Starting audio reader: endpoint=0x%02x, packet_size=%d, buffer_size=%d\n",
		ar.asi.EndpointAddress, packetSize, bufferSize)

	// Allocate transfers and buffers
	for i := 0; i < numAudioTransfers; i++ {
		transfer := C.alloc_audio_transfer(C.int(numIsoPackets))
		if transfer == nil {
			ar.cleanup()
			return fmt.Errorf("failed to allocate transfer")
		}
		ar.transfers = append(ar.transfers, transfer)

		// Allocate buffer using C malloc
		buffer := C.malloc(C.size_t(bufferSize))
		if buffer == nil {
			ar.cleanup()
			return fmt.Errorf("failed to allocate buffer")
		}
		ar.buffers = append(ar.buffers, buffer)
		ar.bufferSizes = append(ar.bufferSizes, bufferSize)

		// Setup ISO transfer
		C.setup_audio_iso_transfer(
			transfer,
			ar.asi.handle,
			C.uchar(ar.asi.EndpointAddress),
			(*C.uchar)(buffer),
			C.int(bufferSize),
			C.int(numIsoPackets),
			C.uintptr_t(ar.id),
			C.uint(audioTimeout),
		)

		// Submit transfer
		if ret := C.libusb_submit_transfer(transfer); ret < 0 {
			ar.cleanup()
			return fmt.Errorf("failed to submit transfer: %s", C.GoString(C.libusb_error_name(ret)))
		}
	}

	// Start event handling loop
	ar.wg.Add(1)
	go ar.eventLoop()

	// Start the audio stream by sending a control message
	// UAC requires explicit start for some devices
	ar.startStream()

	return nil
}

func (ar *AudioReader) startStream() {
	// Try various methods to start the audio stream

	// Method 1: Send a SET_INTERFACE again (sometimes triggers stream start)
	ifnum := ar.asi.InterfaceNumber()
	altSetting := ar.asi.AlternateSetting()
	C.libusb_set_interface_alt_setting(ar.asi.handle, C.int(ifnum), C.int(altSetting))

	// Method 2: Try to set volume to max (Volume Control = 0x02)
	// Volume is typically a 16-bit signed value, 0x7FFF = max
	volumeData := []byte{0xFF, 0x7F} // Max volume in little-endian
	ret := C.libusb_control_transfer(
		ar.asi.handle,
		0x22,               // bmRequestType: Class specific, endpoint, host to device
		0x01,               // SET_CUR
		C.uint16_t(0x0200), // Volume control (0x02) in high byte
		C.uint16_t(ar.asi.EndpointAddress),
		(*C.uchar)(unsafe.Pointer(&volumeData[0])),
		2,
		100,
	)
	if ret >= 0 {
		fmt.Printf("Set volume to max on endpoint 0x%02x\n", ar.asi.EndpointAddress)
	}

	// Method 3: Try to unmute (Mute Control = 0x01)
	muteData := []byte{0x00} // 0x00 = unmute
	ret = C.libusb_control_transfer(
		ar.asi.handle,
		0x22,               // bmRequestType: Class specific, endpoint, host to device
		0x01,               // SET_CUR
		C.uint16_t(0x0100), // Mute control (0x01) in high byte
		C.uint16_t(ar.asi.EndpointAddress),
		(*C.uchar)(unsafe.Pointer(&muteData[0])),
		1,
		100,
	)
	if ret >= 0 {
		fmt.Printf("Unmuted endpoint 0x%02x\n", ar.asi.EndpointAddress)
	}

	// Method 4: Try interface-level unmute for feature unit
	// The feature unit is usually unit 2 for microphone
	ret = C.libusb_control_transfer(
		ar.asi.handle,
		0x21,               // bmRequestType: Class specific, interface, host to device
		0x01,               // SET_CUR
		C.uint16_t(0x0100), // Mute control, channel 0
		C.uint16_t(0x0200), // Feature unit 2, interface 0
		(*C.uchar)(unsafe.Pointer(&muteData[0])),
		1,
		100,
	)
	if ret >= 0 {
		fmt.Printf("Unmuted feature unit 2\n")
	}

	// Method 5: Clear halt again after transfers are submitted
	C.libusb_clear_halt(ar.asi.handle, C.uchar(ar.asi.EndpointAddress))

	// Give device time to respond
	time.Sleep(100 * time.Millisecond)
}

func (ar *AudioReader) eventLoop() {
	defer ar.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ar.ctx.Done():
			fmt.Printf("Final stats: %d transfers, %d bytes received\n",
				atomic.LoadInt64(&ar.transferCount), atomic.LoadInt64(&ar.bytesReceived))
			return
		case <-ticker.C:
			fmt.Printf("Status: %d transfers, %d bytes received\n",
				atomic.LoadInt64(&ar.transferCount), atomic.LoadInt64(&ar.bytesReceived))
		default:
			tv := C.struct_timeval{tv_sec: 0, tv_usec: 10000} // 10ms timeout
			C.libusb_handle_events_timeout(ar.asi.ctx, &tv)
		}
	}
}

func (ar *AudioReader) Read() <-chan []byte {
	return ar.audioBuffer
}

func (ar *AudioReader) Stop() error {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	if !ar.running {
		return nil
	}

	ar.running = false
	ar.cancel()

	// Cancel all transfers
	for _, transfer := range ar.transfers {
		C.libusb_cancel_transfer(transfer)
	}

	// Wait for event loop to finish
	ar.wg.Wait()

	// Cleanup
	ar.cleanup()

	return nil
}

func (ar *AudioReader) cleanup() {
	for _, transfer := range ar.transfers {
		C.libusb_free_transfer(transfer)
	}
	ar.transfers = nil

	// Free C-allocated buffers
	for _, buffer := range ar.buffers {
		C.free(buffer)
	}
	ar.buffers = nil
	ar.bufferSizes = nil

	// Remove from registry
	audioReaderMutex.Lock()
	delete(audioReaderRegistry, ar.id)
	audioReaderMutex.Unlock()

	// Release the interface
	ifnum := ar.asi.InterfaceNumber()
	C.libusb_release_interface(ar.asi.handle, C.int(ifnum))
}

func (ar *AudioReader) GetAudioFormat() AudioFormat {
	return AudioFormat{
		Channels:      int(ar.asi.NrChannels),
		SampleRate:    ar.asi.SamplingFreqs[0], // Use first available
		BitsPerSample: int(ar.asi.BitResolution),
	}
}

func (ar *AudioReader) GetActualSampleRate(duration time.Duration) uint32 {
	// Calculate actual sample rate from bytes received
	bytesReceived := atomic.LoadInt64(&ar.bytesReceived)
	if bytesReceived == 0 || duration == 0 {
		return ar.asi.SamplingFreqs[0] // fallback to expected
	}

	// bytes_per_second = sample_rate * channels * bytes_per_sample
	// sample_rate = bytes_per_second / (channels * bytes_per_sample)
	bytesPerSecond := float64(bytesReceived) / duration.Seconds()
	samplesPerSecond := bytesPerSecond / float64(ar.asi.NrChannels*ar.asi.BitResolution/8)

	// Round to nearest common sample rate
	actualRate := uint32(samplesPerSecond + 0.5)
	commonRates := []uint32{8000, 16000, 24000, 32000, 44100, 48000}

	// Find closest common rate
	closest := actualRate
	minDiff := uint32(^uint32(0))
	for _, rate := range commonRates {
		diff := rate - actualRate
		if actualRate > rate {
			diff = actualRate - rate
		}
		if diff < minDiff {
			minDiff = diff
			closest = rate
		}
	}

	// If within 5% of a common rate, use that
	if float64(minDiff) < float64(closest)*0.05 {
		return closest
	}

	return actualRate
}
