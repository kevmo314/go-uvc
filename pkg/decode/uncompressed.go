package decode

import (
	"fmt"
	"image"
	"io"

	"github.com/kevmo314/go-uvc/pkg/transfers"
)

type UncompressedDecoder struct {
	images        []image.Image
	fourcc        [4]byte
	width, height int
}

func NewUncompressedDecoder(fourcc [4]byte, width, height int) (*UncompressedDecoder, error) {
	return &UncompressedDecoder{fourcc: fourcc, width: width, height: height}, nil
}

func (d *UncompressedDecoder) ReadFrame() (image.Image, error) {
	if len(d.images) == 0 {
		return nil, ErrEAGAIN
	}
	img := d.images[0]
	d.images = d.images[1:]
	return img, nil
}

func (d *UncompressedDecoder) Write(pkt []byte) (int, error) {
	switch d.fourcc {
	case [4]byte{'I', '4', '2', '0'}:
		img := image.NewYCbCr(image.Rect(0, 0, d.width, d.height), image.YCbCrSubsampleRatio420)
		img.Y = pkt[:d.width*d.height]
		img.Cb = pkt[d.width*d.height : d.width*d.height*2]
		img.Cr = pkt[d.width*d.height*2 : d.width*d.height*3]
		return len(pkt), nil
	case [4]byte{'Y', 'U', 'Y', '2'}:
		img := image.NewYCbCr(image.Rect(0, 0, d.width, d.height), image.YCbCrSubsampleRatio422)
		img.Y = pkt[:d.width*d.height]
		img.Cb = pkt[d.width*d.height : d.width*d.height*2]
		img.Cr = pkt[d.width*d.height*2 : d.width*d.height*3]
		return len(pkt), nil
	}
	return 0, fmt.Errorf("unknown FourCC for GUID %s", d.fourcc)
}

func (d *UncompressedDecoder) WriteUSBFrame(fr *transfers.Frame) error {
	buf, err := io.ReadAll(fr)
	if err != nil {
		return err
	}
	if _, err := d.Write(buf); err != nil {
		return err
	}
	return nil
}

func (d *UncompressedDecoder) Close() error {
	return nil
}
