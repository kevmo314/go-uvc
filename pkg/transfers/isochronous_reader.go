package transfers

/*
#cgo LDFLAGS: -lusb-1.0
#include <libusb-1.0/libusb.h>
#include <stdlib.h>

void isochronousReaderTransferCallback(struct libusb_transfer *transfer);
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/mattn/go-pointer"
)

//export isochronousReaderTransferCallback
func isochronousReaderTransferCallback(transfer *C.struct_libusb_transfer) {
	r := pointer.Restore(transfer.user_data).(*IsochronousReader)
	descs := (*[1 << 30]C.struct_libusb_iso_packet_descriptor)(unsafe.Pointer(&transfer.iso_packet_desc))[:transfer.num_iso_packets]
	for i, pkt := range descs {
		if pkt.status != C.LIBUSB_TRANSFER_COMPLETED {
			r.errCh <- fmt.Errorf("libusb_bulk_transfer failed: %d", pkt.status)
			return
		}
		if pkt.actual_length == 0 {
			continue
		}
		pktbuf := C.libusb_get_iso_packet_buffer_simple(transfer, C.uint(i))
		r.readCh <- C.GoBytes(unsafe.Pointer(pktbuf), C.int(pkt.actual_length))
	}
	if ret := C.libusb_submit_transfer(transfer); ret < 0 {
		r.errCh <- fmt.Errorf("libusb_submit_transfer failed: %s", C.GoString(C.libusb_error_name(ret)))
	}
}

type IsochronousReader struct {
	ctx    *C.libusb_context
	txReqs []*C.struct_libusb_transfer
	readCh chan []byte
	errCh  chan error
}

func (si *StreamingInterface) NewIsochronousReader(endpointAddress uint8, packets, packetSize uint32) (*IsochronousReader, error) {
	r := &IsochronousReader{
		ctx:    si.ctx,
		readCh: make(chan []byte, packets),
		errCh:  make(chan error),
	}
	for i := 0; i < 100; i++ {
		tx := C.libusb_alloc_transfer(C.int(packets))
		if tx == nil {
			return nil, fmt.Errorf("libusb_alloc_transfer failed")
		}
		buf := C.malloc(C.ulong(packets * packetSize))
		if buf == nil {
			return nil, fmt.Errorf("malloc failed")
		}
		C.libusb_fill_iso_transfer(
			tx,
			(*C.struct_libusb_device_handle)(si.handle),
			C.uchar(endpointAddress),
			(*C.uchar)(buf),
			C.int(packets*packetSize),
			C.int(packets),
			(*[0]byte)(C.libusb_transfer_cb_fn(C.isochronousReaderTransferCallback)),
			pointer.Save(r),
			0)
		C.libusb_set_iso_packet_lengths(tx, C.uint(packetSize))
		if ret := C.libusb_submit_transfer(tx); ret < 0 {
			if i == 0 {
				return nil, fmt.Errorf("libusb_submit_transfer failed: %s", C.GoString(C.libusb_error_name(ret)))
			}
			break
		}
		r.txReqs = append(r.txReqs, tx)
	}
	return r, nil
}

func (r *IsochronousReader) ReadPayload() (*Payload, error) {
	select {
	case <-r.errCh:
		return nil, <-r.errCh
	case b := <-r.readCh:
		p := &Payload{}
		return p, p.UnmarshalBinary(b)
	}
}

func (r *IsochronousReader) Close() error {
	for _, t := range r.txReqs {
		C.free(unsafe.Pointer(t.buffer))
		C.libusb_free_transfer(t)
	}
	return nil
}
