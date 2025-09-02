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
	"io"
	"runtime/cgo"
	"unsafe"
)

//export isochronousReaderTransferCallback
func isochronousReaderTransferCallback(transfer *C.struct_libusb_transfer) {
	r := cgo.Handle(transfer.user_data).Value().(*IsochronousReader)

	r.completedTxReqs[r.head] = transfer
	r.head = (r.head + 1) % len(r.completedTxReqs)
	if r.size < len(r.completedTxReqs) {
		r.size++
	} else {
		panic("illegal state")
	}
}

type IsochronousReader struct {
	ctx *C.libusb_context
	// reference to all the transfers
	txReqs []*C.struct_libusb_transfer

	// circular buffer of completed transfers
	completedTxReqs []*C.struct_libusb_transfer
	head, size      int
	index           int

	// store handles for cleanup
	handles []cgo.Handle
}

func (si *StreamingInterface) NewIsochronousReader(endpointAddress uint8, packets, packetSize uint32) (*IsochronousReader, error) {
	r := &IsochronousReader{
		ctx: si.ctx,
	}
	handle := cgo.NewHandle(r)
	r.handles = []cgo.Handle{handle}

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
			unsafe.Pointer(handle),
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
	r.completedTxReqs = make([]*C.struct_libusb_transfer, len(r.txReqs))
	return r, nil
}

func (r *IsochronousReader) Read(buf []byte) (int, error) {
	for {
		for r.size == 0 {
			// pump events.
			if ret := C.libusb_handle_events(r.ctx); ret < 0 {
				return 0, fmt.Errorf("libusb_handle_events failed: %d", ret)
			}
		}

		activeTx := r.completedTxReqs[(r.head-r.size+len(r.completedTxReqs))%len(r.completedTxReqs)]
		descs := unsafe.Slice(unsafe.SliceData(activeTx.iso_packet_desc[:]), activeTx.num_iso_packets)
		if r.index == len(descs) {
			// this tx is done, get the next one.
			r.size--
			r.index = 0
			if ret := C.libusb_submit_transfer(activeTx); ret < 0 {
				return 0, fmt.Errorf("libusb_submit_transfer failed: %s", C.GoString(C.libusb_error_name(ret)))
			}
			continue
		}
		pkt := descs[r.index]
		if pkt.status != C.LIBUSB_TRANSFER_COMPLETED {
			return 0, fmt.Errorf("libusb_iso_transfer failed: %d", pkt.status)
		}
		if pkt.actual_length == 0 {
			r.index++
			continue
		}
		if len(buf) < int(pkt.actual_length) {
			return 0, io.ErrShortBuffer
		}
		pktbuf := C.libusb_get_iso_packet_buffer_simple(activeTx, C.uint(r.index))
		r.index++
		return copy(buf, unsafe.Slice((*byte)(pktbuf), int(pkt.actual_length))), nil
	}
}

func (r *IsochronousReader) Close() error {
	for _, t := range r.txReqs {
		C.free(unsafe.Pointer(t.buffer))
		C.libusb_free_transfer(t)
	}
	for _, h := range r.handles {
		h.Delete()
	}
	return nil
}
