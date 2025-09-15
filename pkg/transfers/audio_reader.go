package transfers

import (
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"
)

/*
#cgo LDFLAGS: -lusb-1.0
#include <libusb-1.0/libusb.h>
#include <stdlib.h>
#include <stdint.h>
#include <stddef.h>

// Forward declaration
void audio_transfer_callback(struct libusb_transfer *transfer);

// Helper to get iso_packet_desc pointer, works around flexible array member issues
static inline struct libusb_iso_packet_descriptor* get_iso_packet_desc_audio(struct libusb_transfer *transfer) {
    return (struct libusb_iso_packet_descriptor*)((char*)transfer + offsetof(struct libusb_transfer, iso_packet_desc));
}

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
	id          uintptr
	asi         *AudioStreamingInterface
	ctx         *C.libusb_context
	transfers   []*C.struct_libusb_transfer
	buffers     []unsafe.Pointer // C-allocated buffers
	bufferSizes []int
	mu          sync.Mutex

	// Circular buffer of completed transfers (like IsochronousReader)
	completedTxReqs []*C.struct_libusb_transfer
	head, size      int
	index           int // Current packet index within active transfer

	// Statistics (kept for debugging)
	transferCount  int64
	bytesReceived  int64
	packetsSuccess int64
	packetsError   int64
	packetsEmpty   int64
}

const (
	numAudioTransfers = 8
	numIsoPackets     = 8
	audioTimeout      = 5000
)

func NewAudioReader(asi *AudioStreamingInterface) (*AudioReader, error) {
	audioReaderMutex.Lock()
	defer audioReaderMutex.Unlock()

	id := atomic.AddUintptr(&audioReaderCounter, 1)
	reader := &AudioReader{
		id:  id,
		asi: asi,
		ctx: asi.ctx,
	}
	audioReaderRegistry[id] = reader

	// Initialize transfers and buffers
	if err := reader.initialize(); err != nil {
		delete(audioReaderRegistry, id)
		return nil, err
	}

	return reader, nil
}

//export audio_transfer_callback
func audio_transfer_callback(transfer *C.struct_libusb_transfer) {
	id := uintptr(transfer.user_data)
	audioReaderMutex.Lock()
	reader, ok := audioReaderRegistry[id]
	audioReaderMutex.Unlock()
	if ok {
		reader.handleTransferComplete(transfer)
	}
}

func (ar *AudioReader) handleTransferComplete(transfer *C.struct_libusb_transfer) {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	atomic.AddInt64(&ar.transferCount, 1)

	// Add to circular buffer
	ar.completedTxReqs[ar.head] = transfer
	ar.head = (ar.head + 1) % len(ar.completedTxReqs)
	if ar.size < len(ar.completedTxReqs) {
		ar.size++
	} else {
		panic("audio reader: circular buffer overflow")
	}
}

// initialize sets up the transfers (called from constructor)
func (ar *AudioReader) initialize() error {

	// Calculate packet size based on audio format
	packetSize := int(ar.asi.MaxPacketSize)
	bufferSize := packetSize * numIsoPackets

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

	// Initialize circular buffer
	ar.completedTxReqs = make([]*C.struct_libusb_transfer, len(ar.transfers))

	return nil
}

// ReadAudio reads audio data synchronously by polling libusb
func (ar *AudioReader) ReadAudio(buf []byte) (int, error) {
	for {
		// Poll for completed transfers if none available
		for ar.size == 0 {
			// Handle USB events with timeout
			tv := C.struct_timeval{tv_sec: 0, tv_usec: 10000} // 10ms timeout
			if ret := C.libusb_handle_events_timeout(ar.ctx, &tv); ret < 0 {
				return 0, fmt.Errorf("libusb_handle_events_timeout failed: %s", C.GoString(C.libusb_error_name(ret)))
			}
		}

		ar.mu.Lock()
		// Get the oldest completed transfer
		activeTx := ar.completedTxReqs[(ar.head-ar.size+len(ar.completedTxReqs))%len(ar.completedTxReqs)]

		// Access ISO packet descriptors using C helper function
		descsPtr := C.get_iso_packet_desc_audio(activeTx)
		descs := (*[1 << 16]C.struct_libusb_iso_packet_descriptor)(unsafe.Pointer(descsPtr))[:activeTx.num_iso_packets:activeTx.num_iso_packets]

		// If we've processed all packets in this transfer, resubmit it
		if ar.index >= len(descs) {
			ar.size--
			ar.index = 0

			// Resubmit the transfer
			if ret := C.libusb_submit_transfer(activeTx); ret < 0 {
				ar.mu.Unlock()
				return 0, fmt.Errorf("failed to resubmit transfer: %s", C.GoString(C.libusb_error_name(ret)))
			}
			ar.mu.Unlock()
			continue
		}

		// Process current packet
		packet := descs[ar.index]

		// Skip packets with errors
		if packet.status != C.LIBUSB_TRANSFER_COMPLETED {
			atomic.AddInt64(&ar.packetsError, 1)
			ar.index++
			ar.mu.Unlock()
			continue
		}

		// Skip empty packets
		if packet.actual_length == 0 {
			atomic.AddInt64(&ar.packetsEmpty, 1)
			ar.index++
			ar.mu.Unlock()
			continue
		}

		// Make sure buffer is large enough
		if len(buf) < int(packet.actual_length) {
			ar.mu.Unlock()
			return 0, fmt.Errorf("buffer too small: need %d, have %d", packet.actual_length, len(buf))
		}

		// Get packet buffer and copy data
		pktbuf := C.libusb_get_iso_packet_buffer_simple(activeTx, C.uint(ar.index))
		if pktbuf == nil {
			ar.index++
			ar.mu.Unlock()
			continue
		}

		// Copy the audio data
		n := copy(buf, unsafe.Slice((*byte)(pktbuf), int(packet.actual_length)))

		atomic.AddInt64(&ar.packetsSuccess, 1)
		atomic.AddInt64(&ar.bytesReceived, int64(n))

		ar.index++
		ar.mu.Unlock()

		return n, nil
	}
}

func (ar *AudioReader) Close() error {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	// Cancel all transfers
	for _, transfer := range ar.transfers {
		C.libusb_cancel_transfer(transfer)
	}

	// Wait for all transfers to complete
	// Pump events until all transfers are cancelled
	for ar.size < len(ar.transfers) {
		tv := C.struct_timeval{tv_sec: 0, tv_usec: 10000}
		C.libusb_handle_events_timeout(ar.ctx, &tv)
	}

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
