//go:build !ffmpeg

package decode

import (
	"fmt"
	"image"

	"github.com/kevmo314/go-uvc/pkg/transfers"
)

var ErrEAGAIN = fmt.Errorf("EAGAIN")

type LibAVCodecDecoder struct{}

func newDecoder(codecID uint32) (*LibAVCodecDecoder, error) {
	return nil, fmt.Errorf("ffmpeg support not enabled; rebuild with -tags ffmpeg")
}

func NewH264Decoder() (*LibAVCodecDecoder, error) {
	return newDecoder(0)
}

func NewVP8Decoder() (*LibAVCodecDecoder, error) {
	return newDecoder(0)
}

func (d *LibAVCodecDecoder) Close() error {
	return nil
}

func (d *LibAVCodecDecoder) SetSPSPPS(sps, pps []byte) {}

func (d *LibAVCodecDecoder) Write(pkt []byte) (int, error) {
	return 0, fmt.Errorf("ffmpeg support not enabled; rebuild with -tags ffmpeg")
}

func (d *LibAVCodecDecoder) WriteUSBFrame(fr *transfers.Frame) error {
	return fmt.Errorf("ffmpeg support not enabled; rebuild with -tags ffmpeg")
}

func (d *LibAVCodecDecoder) ReadFrame() (image.Image, error) {
	return nil, fmt.Errorf("ffmpeg support not enabled; rebuild with -tags ffmpeg")
}
