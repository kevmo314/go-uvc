package decode

import (
	"fmt"
	"image"

	"github.com/kevmo314/go-uvc/pkg/descriptors"
	"github.com/kevmo314/go-uvc/pkg/transfers"
)

type VideoDecoder interface {
	ReadFrame() (image.Image, error)
	Write(pkt []byte) (int, error)
	WriteUSBFrame(fr *transfers.Frame) error
	Close() error
}

func NewDescriptorDecoder(fd descriptors.FormatDescriptor, fr descriptors.FrameDescriptor) (VideoDecoder, error) {
	switch fd := fd.(type) {
	case *descriptors.MJPEGFormatDescriptor:
		return NewMJPEGDecoder()
	case *descriptors.H264FormatDescriptor:
		return NewH264Decoder()
	case *descriptors.VP8FormatDescriptor:
		return NewVP8Decoder()
	case *descriptors.FrameBasedFormatDescriptor:
		fcc, err := fd.FourCC()
		if err != nil {
			return nil, err
		}
		switch fcc {
		case [4]byte{'h', '2', '6', '4'}, [4]byte{'H', '2', '6', '4'}:
			return NewH264Decoder()
		case [4]byte{'v', 'p', '8', '0'}:
			return NewVP8Decoder()
		case [4]byte{'m', 'j', 'p', 'g'}:
			return NewMJPEGDecoder()
		}
	case *descriptors.UncompressedFormatDescriptor:
		fcc, err := fd.FourCC()
		if err != nil {
			return nil, err
		}
		fr := fr.(*descriptors.UncompressedFrameDescriptor)
		return NewUncompressedDecoder(fcc, int(fr.Width), int(fr.Height))
	}
	return nil, fmt.Errorf("unsupported frame descriptor: %#v", fd)
}

type FrameReaderDecoder struct {
	reader *transfers.FrameReader
	dec    VideoDecoder
}

func NewFrameReaderDecoder(reader *transfers.FrameReader, fd descriptors.FormatDescriptor, fr descriptors.FrameDescriptor) (*FrameReaderDecoder, error) {
	dec, err := NewDescriptorDecoder(fd, fr)
	if err != nil {
		return nil, err
	}
	return &FrameReaderDecoder{reader: reader, dec: dec}, nil
}

func (d *FrameReaderDecoder) ReadFrame() (image.Image, error) {
	for {
		img, err := d.dec.ReadFrame()
		if err == nil {
			return img, nil
		}
		if err != ErrEAGAIN {
			return nil, err
		}
		fr, err := d.reader.ReadFrame()
		if err != nil {
			return nil, err
		}
		if err := d.dec.WriteUSBFrame(fr); err != nil {
			return nil, err
		}
	}
}
