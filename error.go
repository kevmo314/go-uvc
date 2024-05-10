package uvc

/*
#cgo LDFLAGS: -lusb-1.0
#include <libusb-1.0/libusb.h>
*/
import "C"
import "errors"

var (
	ErrInvalidDescriptor = errors.New("invalid descriptor")
)

func libusberror(err C.int) error {
	return errors.New(C.GoString(C.libusb_error_name(err)))
}
