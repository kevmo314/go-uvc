package decode

import (
	"bytes"
	"image"
	"image/jpeg"

	"github.com/kevmo314/go-uvc/pkg/transfers"
)

type MJPEGDecoder struct {
	imagesBuf []image.Image
}

func NewMJPEGDecoder() (*MJPEGDecoder, error) {
	return &MJPEGDecoder{}, nil
}

func (d *MJPEGDecoder) Write(pkt []byte) (int, error) {
	img, err := jpeg.Decode(bytes.NewReader(pkt))
	if err != nil {
		return 0, err
	}
	d.imagesBuf = append(d.imagesBuf, img)
	return len(pkt), nil
}

func (d *MJPEGDecoder) WriteUSBFrame(fr *transfers.Frame) error {
	img, err := jpeg.Decode(fr)
	if err != nil {
		return err
	}
	d.imagesBuf = append(d.imagesBuf, img)
	return nil
}

func (d *MJPEGDecoder) ReadFrame() (image.Image, error) {
	if len(d.imagesBuf) == 0 {
		return nil, ErrEAGAIN
	}
	img := d.imagesBuf[0]
	d.imagesBuf = d.imagesBuf[1:]
	return img, nil
}

func (d *MJPEGDecoder) Close() error {
	return nil
}
