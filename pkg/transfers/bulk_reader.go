package transfers

/*
#cgo LDFLAGS: -lusb-1.0
#include <libusb-1.0/libusb.h>
#include <stdlib.h>

void bulkReaderTransferCallback(struct libusb_transfer *transfer);
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/mattn/go-pointer"
)

type BulkReader struct {
	txReqs []*C.struct_libusb_transfer
	readCh chan []byte
	errCh  chan error
}

//export bulkReaderTransferCallback
func bulkReaderTransferCallback(transfer *C.struct_libusb_transfer) {
	r := pointer.Restore(transfer.user_data).(*BulkReader)
	if transfer.status == C.LIBUSB_TRANSFER_COMPLETED {
		if transfer.actual_length > 0 {
			r.readCh <- C.GoBytes(unsafe.Pointer(transfer.buffer), C.int(transfer.actual_length))
		}
		if ret := C.libusb_submit_transfer(transfer); ret < 0 {
			r.errCh <- fmt.Errorf("libusb_submit_transfer failed: %s", C.GoString(C.libusb_error_name(ret)))
		}
	} else {
		r.errCh <- fmt.Errorf("libusb_bulk_transfer failed: %d", transfer.status)
	}
}

func NewBulkReader(deviceHandle unsafe.Pointer, endpointAddress uint8, mtu uint32) (*BulkReader, error) {
	// the libusb sync api seems to result in some partial reads on some devices so we use the async api
	r := &BulkReader{
		txReqs: make([]*C.struct_libusb_transfer, 0, 100),
		errCh:  make(chan error),
	}
	for i := 0; i < 100; i++ {
		tx := C.libusb_alloc_transfer(0)
		if tx == nil {
			return nil, fmt.Errorf("libusb_alloc_transfer failed")
		}
		buf := C.malloc(C.ulong(mtu))
		if buf == nil {
			return nil, fmt.Errorf("malloc failed")
		}
		C.libusb_fill_bulk_transfer(
			tx,
			(*C.struct_libusb_device_handle)(deviceHandle),
			C.uchar(endpointAddress),
			(*C.uchar)(buf),
			C.int(mtu),
			(*[0]byte)(C.libusb_transfer_cb_fn(C.bulkReaderTransferCallback)),
			pointer.Save(r),
			0)
		if ret := C.libusb_submit_transfer(tx); ret < 0 {
			if i == 0 {
				return nil, fmt.Errorf("libusb_submit_transfer failed: %s", C.GoString(C.libusb_error_name(ret)))
			}
			break
		}
		r.txReqs = append(r.txReqs, tx)
	}
	r.readCh = make(chan []byte, len(r.txReqs))
	return r, nil
}

func (r *BulkReader) ReadPayload() (*Payload, error) {
	select {
	case <-r.errCh:
		return nil, <-r.errCh
	case b := <-r.readCh:
		p := &Payload{}
		return p, p.UnmarshalBinary(b)
	}
}

func (r *BulkReader) Close() error {
	for _, t := range r.txReqs {
		C.free(unsafe.Pointer(t.buffer))
		C.libusb_free_transfer(t)
	}
	return nil
}
