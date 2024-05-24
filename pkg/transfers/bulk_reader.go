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
	ctx    *C.libusb_context
	txReqs []*C.struct_libusb_transfer

	// circular buffer of completed transfers
	completedTxReqs []*C.struct_libusb_transfer
	head, size      int
}

//export bulkReaderTransferCallback
func bulkReaderTransferCallback(transfer *C.struct_libusb_transfer) {
	r := pointer.Restore(transfer.user_data).(*BulkReader)

	r.completedTxReqs[r.head] = transfer
	r.head = (r.head + 1) % len(r.completedTxReqs)
	if r.size < len(r.completedTxReqs) {
		r.size++
	} else {
		panic("illegal state")
	}
}

func (si *StreamingInterface) NewBulkReader(endpointAddress uint8, mtu uint32) (*BulkReader, error) {
	// the libusb sync api seems to result in some partial reads on some devices so we use the async api
	r := &BulkReader{
		ctx:    si.ctx,
		txReqs: make([]*C.struct_libusb_transfer, 0, 100),
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
			(*C.struct_libusb_device_handle)(si.handle),
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
	r.completedTxReqs = make([]*C.struct_libusb_transfer, len(r.txReqs))
	return r, nil
}

func (r *BulkReader) Read(buf []byte) (int, error) {
	for r.size == 0 {
		// pump events.
		C.libusb_handle_events(r.ctx)
	}

	activeTx := r.completedTxReqs[(r.head-r.size+len(r.completedTxReqs))%len(r.completedTxReqs)]
	r.size--
	n := copy(buf, (*[1 << 30]byte)(unsafe.Pointer(activeTx.buffer))[:activeTx.actual_length])
	if ret := C.libusb_submit_transfer(activeTx); ret < 0 {
		return 0, fmt.Errorf("libusb_submit_transfer failed: %s", C.GoString(C.libusb_error_name(ret)))
	}
	return n, nil
}

func (r *BulkReader) Close() error {
	for _, t := range r.txReqs {
		C.free(unsafe.Pointer(t.buffer))
		C.libusb_free_transfer(t)
	}
	return nil
}
