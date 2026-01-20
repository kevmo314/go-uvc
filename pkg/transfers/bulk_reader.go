package transfers

import (
	"fmt"
	"time"

	usb "github.com/kevmo314/go-usb"
)

type BulkReader struct {
	handle   *usb.DeviceHandle
	endpoint uint8
	mtu      uint32
}

func (si *StreamingInterface) NewBulkReader(endpointAddress uint8, mtu uint32) (*BulkReader, error) {
	return &BulkReader{
		handle:   si.handle,
		endpoint: endpointAddress,
		mtu:      mtu,
	}, nil
}

func (r *BulkReader) Read(buf []byte) (int, error) {
	n, err := r.handle.BulkTransfer(r.endpoint, buf, 5*time.Second)
	if err != nil {
		return 0, fmt.Errorf("bulk_transfer failed: %w", err)
	}
	return n, nil
}

func (r *BulkReader) Close() error {
	return nil
}
