package transfers

import "io"

type Reader interface {
	io.Closer
	ReadFrame() ([]byte, error)
}
