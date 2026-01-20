package transfers

import (
	"fmt"
	"sync"

	usb "github.com/kevmo314/go-usb"
)

const (
	// DefaultNumTransfers is the number of queued transfers for async bulk reads.
	// More transfers = better pipelining but more memory usage.
	// Linux kernel typically limits bulk URBs per endpoint to 4-16.
	// libuvc uses 100 by default, but Go's goroutines handle scheduling better,
	// so 8 is a reasonable middle ground.
	DefaultNumTransfers = 8
)

// AsyncBulkReader implements io.Reader with queued USB bulk transfers.
// It keeps multiple URBs in flight to maximize USB throughput.
type AsyncBulkReader struct {
	handle    *usb.DeviceHandle
	endpoint  uint8
	mtu       uint32
	transfers []*usb.AsyncBulkTransfer

	mu       sync.Mutex
	nextRead int  // Index of next transfer to read from
	closed   bool

	// Buffered data from a previous read that wasn't fully consumed
	pending    []byte
	pendingOff int
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

	r := &AsyncBulkReader{
		handle:    handle,
		endpoint:  endpointAddress,
		mtu:       mtu,
		transfers: make([]*usb.AsyncBulkTransfer, numTransfers),
	}

	// Create all transfers
	for i := 0; i < numTransfers; i++ {
		t, err := handle.NewAsyncBulkTransfer(endpointAddress, int(mtu))
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

// Read implements io.Reader. It reads data from the next completed USB transfer.
func (r *AsyncBulkReader) Read(buf []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return 0, fmt.Errorf("reader closed")
	}

	// First, drain any pending data from a previous read
	if len(r.pending) > r.pendingOff {
		n := copy(buf, r.pending[r.pendingOff:])
		r.pendingOff += n
		if r.pendingOff >= len(r.pending) {
			r.pending = nil
			r.pendingOff = 0
		}
		return n, nil
	}

	// Wait for the next transfer to complete
	t := r.transfers[r.nextRead]
	data, err := t.Wait()
	if err != nil {
		return 0, fmt.Errorf("async bulk read failed: %w", err)
	}

	// Immediately re-submit this transfer to keep the pipeline full
	if err := t.Submit(); err != nil {
		// Don't return error - just log it and continue
		// The next read will fail if this is a persistent issue
	}

	// Advance to next transfer
	r.nextRead = (r.nextRead + 1) % len(r.transfers)

	// Copy data to caller's buffer
	n := copy(buf, data)
	if n < len(data) {
		// Save remaining data for next Read call
		r.pending = data
		r.pendingOff = n
	}

	return n, nil
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
