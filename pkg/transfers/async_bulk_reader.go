package transfers

import (
	"fmt"
	"sync"

	usb "github.com/kevmo314/go-usb"
)

const (
	// DefaultNumTransfers is the number of queued transfers for async bulk reads.
	// More transfers = better pipelining but more memory usage.
	// With 16KB URB buffers, 64 URBs = 1MB total, well under kernel limits.
	DefaultNumTransfers = 64

	// MaxURBBufferSize is the maximum buffer size per URB to avoid ENOMEM.
	// This matches libusb's MAX_BULK_BUFFER_LENGTH and the kernel's MAX_USBFS_BUFFER_SIZE.
	MaxURBBufferSize = 16384
)

// AsyncBulkReader implements io.Reader with queued USB bulk transfers.
// It keeps multiple URBs in flight to maximize USB throughput.
// For UVC bulk transfers, it reassembles data from multiple small URBs
// into complete payloads (delimited by short transfers).
type AsyncBulkReader struct {
	handle    *usb.DeviceHandle
	endpoint  uint8
	urbSize   int // Actual URB buffer size (capped at MaxURBBufferSize)
	transfers []*usb.AsyncBulkTransfer

	mu       sync.Mutex
	nextRead int // Index of next transfer to read from
	closed   bool
}

// NewAsyncBulkReader creates a new async bulk reader with queued transfers.
func (si *StreamingInterface) NewAsyncBulkReader(endpointAddress uint8, mtu uint32) (*AsyncBulkReader, error) {
	return NewAsyncBulkReaderWithCount(si.handle, endpointAddress, mtu, DefaultNumTransfers)
}

// NewAsyncBulkReaderWithCount creates an async bulk reader with a specific number of queued transfers.
func NewAsyncBulkReaderWithCount(handle *usb.DeviceHandle, endpointAddress uint8, mtu uint32, numTransfers int) (*AsyncBulkReader, error) {
	if numTransfers < 1 {
		numTransfers = 1
	}

	// Use smaller URB buffers to avoid ENOMEM, but cap at mtu if mtu is smaller
	urbSize := MaxURBBufferSize
	if int(mtu) < urbSize {
		urbSize = int(mtu)
	}

	r := &AsyncBulkReader{
		handle:    handle,
		endpoint:  endpointAddress,
		urbSize:   urbSize,
		transfers: make([]*usb.AsyncBulkTransfer, numTransfers),
	}

	// Create all transfers with small URB buffers
	for i := 0; i < numTransfers; i++ {
		t, err := handle.NewAsyncBulkTransfer(endpointAddress, urbSize)
		if err != nil {
			// Clean up already created transfers
			for j := 0; j < i; j++ {
				r.transfers[j].Cancel()
			}
			return nil, fmt.Errorf("failed to create async transfer %d: %w", i, err)
		}
		r.transfers[i] = t
	}

	// Submit all transfers to fill the pipeline
	for i := 0; i < numTransfers; i++ {
		if err := r.transfers[i].Submit(); err != nil {
			// Cancel all submitted transfers
			for j := 0; j < i; j++ {
				r.transfers[j].Cancel()
			}
			return nil, fmt.Errorf("failed to submit initial transfer %d: %w", i, err)
		}
	}

	return r, nil
}

// Read implements io.Reader. It returns complete UVC payloads by reassembling
// data from multiple small URBs. A payload is complete when we receive a short
// transfer (actual_length < urbSize) which signals end of USB transfer.
func (r *AsyncBulkReader) Read(buf []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return 0, fmt.Errorf("reader closed")
	}

	// Accumulate URB data directly into buf until short transfer
	written := 0
	for {
		t := r.transfers[r.nextRead]
		data, err := t.Wait()
		if err != nil {
			return 0, fmt.Errorf("async bulk read failed: %w", err)
		}

		// Check buffer has enough space
		if len(buf)-written < len(data) {
			return 0, fmt.Errorf("buffer too small: need %d bytes, have %d", len(data), len(buf)-written)
		}

		// Copy BEFORE resubmitting to avoid race with kernel
		copy(buf[written:], data)
		written += len(data)

		// Now safe to resubmit
		t.Submit()
		r.nextRead = (r.nextRead + 1) % len(r.transfers)

		// Short transfer (including ZLP) signals end of payload
		if len(data) < r.urbSize {
			return written, nil
		}
	}
}

// Close cancels all pending transfers and releases resources.
func (r *AsyncBulkReader) Close() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true
	r.mu.Unlock()

	// Cancel all transfers
	for _, t := range r.transfers {
		t.Cancel()
	}
	// Wait for all cancellations to complete
	for _, t := range r.transfers {
		t.Wait() // Ignore error - we're closing
	}

	return nil
}
