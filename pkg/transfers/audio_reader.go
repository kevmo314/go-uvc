package transfers

import (
	"fmt"
	"sync"
	"sync/atomic"

	usb "github.com/kevmo314/go-usb"
)

type AudioReader struct {
	asi       *AudioStreamingInterface
	handle    *usb.DeviceHandle
	transfers []*usb.IsochronousTransfer
	mu        sync.Mutex

	// Current state
	currentTx int
	packetIdx int

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
	reader := &AudioReader{
		asi:    asi,
		handle: asi.handle,
	}

	// Initialize transfers and buffers
	if err := reader.initialize(); err != nil {
		return nil, err
	}

	return reader, nil
}

// initialize sets up the transfers (called from constructor)
func (ar *AudioReader) initialize() error {
	// Calculate packet size based on audio format
	packetSize := int(ar.asi.MaxPacketSize)

	// Allocate and submit transfers
	for i := 0; i < numAudioTransfers; i++ {
		tx, err := ar.handle.NewIsochronousTransfer(ar.asi.EndpointAddress, numIsoPackets, packetSize)
		if err != nil {
			ar.cleanup()
			return fmt.Errorf("failed to create isochronous transfer: %w", err)
		}
		ar.transfers = append(ar.transfers, tx)

		// Submit transfer
		if err := tx.Submit(); err != nil {
			ar.cleanup()
			return fmt.Errorf("failed to submit transfer: %w", err)
		}
	}

	return nil
}

// ReadAudio reads audio data synchronously
func (ar *AudioReader) ReadAudio(buf []byte) (int, error) {
	for {
		tx := ar.transfers[ar.currentTx]

		// Wait for the current transfer to complete
		if err := tx.Wait(); err != nil {
			return 0, fmt.Errorf("isochronous transfer failed: %w", err)
		}

		packets := tx.Packets()

		// If we've processed all packets in this transfer, resubmit it
		if ar.packetIdx >= len(packets) {
			// Resubmit the transfer
			if err := tx.Submit(); err != nil {
				return 0, fmt.Errorf("failed to resubmit transfer: %w", err)
			}
			ar.packetIdx = 0
			ar.currentTx = (ar.currentTx + 1) % len(ar.transfers)
			atomic.AddInt64(&ar.transferCount, 1)
			continue
		}

		// Process current packet
		packet := packets[ar.packetIdx]

		// Skip packets with errors
		if packet.Status != 0 {
			atomic.AddInt64(&ar.packetsError, 1)
			ar.packetIdx++
			continue
		}

		// Skip empty packets
		if packet.ActualLength == 0 {
			atomic.AddInt64(&ar.packetsEmpty, 1)
			ar.packetIdx++
			continue
		}

		// Make sure buffer is large enough
		if len(buf) < int(packet.ActualLength) {
			return 0, fmt.Errorf("buffer too small: need %d, have %d", packet.ActualLength, len(buf))
		}

		// Get packet buffer and copy data
		data, err := tx.IsoPacketBuffer(ar.packetIdx)
		if err != nil {
			ar.packetIdx++
			continue
		}

		// Copy the audio data
		n := copy(buf, data)

		atomic.AddInt64(&ar.packetsSuccess, 1)
		atomic.AddInt64(&ar.bytesReceived, int64(n))

		ar.packetIdx++

		return n, nil
	}
}

func (ar *AudioReader) Close() error {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	// Cancel all transfers
	for _, tx := range ar.transfers {
		tx.Cancel()
	}

	ar.cleanup()

	return nil
}

func (ar *AudioReader) cleanup() {
	ar.transfers = nil

	// Release the interface
	ifnum := ar.asi.InterfaceNumber()
	ar.handle.ReleaseInterface(ifnum)
}

func (ar *AudioReader) GetAudioFormat() AudioFormat {
	return AudioFormat{
		Channels:      int(ar.asi.NrChannels),
		SampleRate:    ar.asi.SamplingFreqs[0], // Use first available
		BitsPerSample: int(ar.asi.BitResolution),
	}
}
