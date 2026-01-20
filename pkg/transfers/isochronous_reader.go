package transfers

import (
	"fmt"
	"io"

	usb "github.com/kevmo314/go-usb"
)

type IsochronousReader struct {
	handle     *usb.DeviceHandle
	transfers  []*usb.IsochronousTransfer
	currentTx  int
	packetIdx  int
	numPackets int
	packetSize int
}

func (si *StreamingInterface) NewIsochronousReader(endpointAddress uint8, packets, packetSize uint32) (*IsochronousReader, error) {
	r := &IsochronousReader{
		handle:     si.handle,
		numPackets: int(packets),
		packetSize: int(packetSize),
	}

	// Create and submit multiple transfers for continuous streaming
	// libuvc uses 100 by default, but Go's goroutines handle scheduling better,
	// so 8 is a reasonable middle ground.
	numTransfers := 8
	r.transfers = make([]*usb.IsochronousTransfer, numTransfers)

	for i := 0; i < numTransfers; i++ {
		tx, err := si.handle.NewIsochronousTransfer(endpointAddress, int(packets), int(packetSize))
		if err != nil {
			// Clean up any already created transfers
			for j := 0; j < i; j++ {
				r.transfers[j].Cancel()
			}
			return nil, fmt.Errorf("failed to create isochronous transfer: %w", err)
		}
		if err := tx.Submit(); err != nil {
			// Clean up any already created transfers
			for j := 0; j < i; j++ {
				r.transfers[j].Cancel()
			}
			return nil, fmt.Errorf("failed to submit isochronous transfer: %w", err)
		}
		r.transfers[i] = tx
	}

	return r, nil
}

func (r *IsochronousReader) Read(buf []byte) (int, error) {
	for {
		tx := r.transfers[r.currentTx]

		// Wait for the current transfer to complete
		if err := tx.Wait(); err != nil {
			return 0, fmt.Errorf("isochronous transfer failed: %w", err)
		}

		packets := tx.Packets()
		if r.packetIdx >= len(packets) {
			// Resubmit this transfer and move to the next one
			if err := tx.Submit(); err != nil {
				return 0, fmt.Errorf("failed to resubmit isochronous transfer: %w", err)
			}
			r.packetIdx = 0
			r.currentTx = (r.currentTx + 1) % len(r.transfers)
			continue
		}

		pkt := packets[r.packetIdx]
		if pkt.Status != 0 {
			r.packetIdx++
			continue
		}
		if pkt.ActualLength == 0 {
			r.packetIdx++
			continue
		}
		if len(buf) < int(pkt.ActualLength) {
			return 0, io.ErrShortBuffer
		}

		data, err := tx.IsoPacketBuffer(r.packetIdx)
		if err != nil {
			r.packetIdx++
			continue
		}
		r.packetIdx++
		return copy(buf, data), nil
	}
}

func (r *IsochronousReader) Close() error {
	for _, tx := range r.transfers {
		tx.Cancel()
	}
	// Wait for all cancellations to complete
	for _, tx := range r.transfers {
		tx.Wait() // Ignore error - we're closing
	}
	return nil
}
