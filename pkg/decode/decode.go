package decode

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"unsafe"

	"github.com/kevmo314/go-uvc/pkg/descriptors"
	"github.com/kevmo314/go-uvc/pkg/transfers"
)

/*
#cgo LDFLAGS: -lavcodec -lavutil
#include <libavcodec/avcodec.h>
#include <libavutil/avutil.h>
#include <libavutil/pixdesc.h>
*/
import "C"

const averror_eagain = -C.EAGAIN

var ErrEAGAIN = fmt.Errorf("EAGAIN")

type VideoDecoder interface {
	ReadFrame() (image.Image, error)
	Write(pkt []byte) (int, error)
	WriteUSBFrame(fr *transfers.Frame) error
	Close() error
}

type LibAVCodecDecoder struct {
	ctx   *C.AVCodecContext
	pkt   *C.AVPacket
	frame *C.AVFrame
}

func newDecoder(codecID uint32) (*LibAVCodecDecoder, error) {
	codec := C.avcodec_find_decoder(codecID)
	if codec == nil {
		return nil, fmt.Errorf("avcodec_find_decoder() failed")
	}

	ctx := C.avcodec_alloc_context3(codec)
	if ctx == nil {
		return nil, fmt.Errorf("avcodec_alloc_context3() failed")
	}

	if res := C.avcodec_open2(ctx, codec, nil); res < 0 {
		C.avcodec_free_context(&ctx)
		return nil, fmt.Errorf("avcodec_open2() failed")
	}

	pkt := C.av_packet_alloc()
	if pkt == nil {
		C.avcodec_free_context(&ctx)
		return nil, fmt.Errorf("av_packet_alloc() failed")
	}

	frame := C.av_frame_alloc()
	if frame == nil {
		C.av_packet_free(&pkt)
		C.avcodec_free_context(&ctx)
		return nil, fmt.Errorf("av_frame_alloc() failed")
	}

	return &LibAVCodecDecoder{ctx: ctx, pkt: pkt, frame: frame}, nil
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
	}
	return nil, fmt.Errorf("unsupported frame descriptor: %#v", fd)
}

func NewH264Decoder() (*LibAVCodecDecoder, error) {
	return newDecoder(C.AV_CODEC_ID_H264)
}

func NewVP8Decoder() (*LibAVCodecDecoder, error) {
	return newDecoder(C.AV_CODEC_ID_VP8)
}

func (d *LibAVCodecDecoder) Close() error {
	C.av_frame_free(&d.frame)
	C.av_packet_free(&d.pkt)
	C.avcodec_free_context(&d.ctx)
	return nil
}

func (d *LibAVCodecDecoder) Write(pkt []byte) (int, error) {
	d.pkt.data = (*C.uint8_t)(C.CBytes(pkt))
	d.pkt.size = C.int(len(pkt))

	if res := C.avcodec_send_packet(d.ctx, d.pkt); res < 0 {
		return 0, fmt.Errorf("avcodec_send_packet() failed: %d", res)
	}
	return len(pkt), nil
}

func (d *LibAVCodecDecoder) WriteUSBFrame(fr *transfers.Frame) error {
	for _, p := range fr.Payloads {
		d.pkt.data = (*C.uint8_t)(C.CBytes(p.Data))
		d.pkt.size = C.int(len(p.Data))

		if res := C.avcodec_send_packet(d.ctx, d.pkt); res < 0 {
			return fmt.Errorf("avcodec_send_packet() failed: %d", res)
		}
	}
	return nil
}

func (d *LibAVCodecDecoder) ReadFrame() (image.Image, error) {
	if res := C.avcodec_receive_frame(d.ctx, d.frame); res < 0 {
		if res == averror_eagain {
			return nil, ErrEAGAIN
		}
		return nil, fmt.Errorf("avcodec_receive_frame() failed: %d", res)
	}
	switch d.frame.format {
	case C.AV_PIX_FMT_YUV420P, C.AV_PIX_FMT_YUV422P, C.AV_PIX_FMT_YUV444P, C.AV_PIX_FMT_YUV410P, C.AV_PIX_FMT_YUV411P, C.AV_PIX_FMT_YUVJ420P, C.AV_PIX_FMT_YUVJ422P, C.AV_PIX_FMT_YUVJ444P:
		img := &image.YCbCr{
			Y:       (*[1 << 30]uint8)(unsafe.Pointer(d.frame.data[0]))[:d.frame.height*d.frame.linesize[0]],
			Cb:      (*[1 << 30]uint8)(unsafe.Pointer(d.frame.data[1]))[:d.frame.height*d.frame.linesize[1]],
			Cr:      (*[1 << 30]uint8)(unsafe.Pointer(d.frame.data[2]))[:d.frame.height*d.frame.linesize[2]],
			Rect:    image.Rect(0, 0, int(d.frame.width), int(d.frame.height)),
			YStride: int(d.frame.linesize[0]),
			CStride: int(d.frame.linesize[1]),
		}
		switch d.frame.format {
		case C.AV_PIX_FMT_YUV420P, C.AV_PIX_FMT_YUVJ420P:
			img.SubsampleRatio = image.YCbCrSubsampleRatio420
		case C.AV_PIX_FMT_YUV422P, C.AV_PIX_FMT_YUVJ422P:
			img.SubsampleRatio = image.YCbCrSubsampleRatio422
		case C.AV_PIX_FMT_YUV444P, C.AV_PIX_FMT_YUVJ444P:
			img.SubsampleRatio = image.YCbCrSubsampleRatio444
		case C.AV_PIX_FMT_YUV410P:
			img.SubsampleRatio = image.YCbCrSubsampleRatio410
		case C.AV_PIX_FMT_YUV411P:
			img.SubsampleRatio = image.YCbCrSubsampleRatio411
		}
		return img, nil
	case C.AV_PIX_FMT_RGB24:
		return &RGB{
			Pix:    (*[1 << 30]uint8)(unsafe.Pointer(d.frame.data[0]))[:d.frame.height*d.frame.linesize[0]],
			Stride: int(d.frame.linesize[0]),
			Rect:   image.Rect(0, 0, int(d.frame.width), int(d.frame.height)),
		}, nil
	case C.AV_PIX_FMT_BGR24:
		return &BGR{
			Pix:    (*[1 << 30]uint8)(unsafe.Pointer(d.frame.data[0]))[:d.frame.height*d.frame.linesize[0]],
			Stride: int(d.frame.linesize[0]),
			Rect:   image.Rect(0, 0, int(d.frame.width), int(d.frame.height)),
		}, nil
	case C.AV_PIX_FMT_GRAY8:
		return &image.Gray{
			Pix:    (*[1 << 30]uint8)(unsafe.Pointer(d.frame.data[0]))[:d.frame.height*d.frame.linesize[0]],
			Stride: int(d.frame.linesize[0]),
			Rect:   image.Rect(0, 0, int(d.frame.width), int(d.frame.height)),
		}, nil
	case C.AV_PIX_FMT_GRAY16BE:
		return &image.Gray16{
			Pix:    (*[1 << 30]uint8)(unsafe.Pointer(d.frame.data[0]))[:d.frame.height*d.frame.linesize[0]],
			Stride: int(d.frame.linesize[0]),
			Rect:   image.Rect(0, 0, int(d.frame.width), int(d.frame.height)),
		}, nil
	}
	return nil, fmt.Errorf("unsupported pixel format: %s", C.GoString(C.av_get_pix_fmt_name(int32(d.frame.format))))
}

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
