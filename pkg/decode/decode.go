package decode

import (
	"fmt"
	"image"
	"unsafe"
)

/*
#cgo LDFLAGS: -lavcodec
#include <libavcodec/avcodec.h>
*/
import "C"

type VideoDecoder struct {
	ctx   *C.AVCodecContext
	pkt   *C.AVPacket
	frame *C.AVFrame
}

func newDecoder(codecID uint32) (*VideoDecoder, error) {
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

	return &VideoDecoder{ctx: ctx, pkt: pkt, frame: frame}, nil
}

func NewMJPEGDecoder() (*VideoDecoder, error) {
	return newDecoder(C.AV_CODEC_ID_MJPEG)
}

func NewH264Decoder() (*VideoDecoder, error) {
	return newDecoder(C.AV_CODEC_ID_H264)
}

func NewVP8Decoder() (*VideoDecoder, error) {
	return newDecoder(C.AV_CODEC_ID_VP8)
}

func (d *VideoDecoder) Close() error {
	C.av_frame_free(&d.frame)
	C.av_packet_free(&d.pkt)
	C.avcodec_free_context(&d.ctx)
	return nil
}

func (d *VideoDecoder) WriteNALU(nalu []byte) error {
	nalu = append([]uint8{0x00, 0x00, 0x00, 0x01}, []uint8(nalu)...)

	d.pkt.data = (*C.uint8_t)(C.CBytes(nalu))
	d.pkt.size = C.int(len(nalu))

	if res := C.avcodec_send_packet(d.ctx, d.pkt); res < 0 {
		return fmt.Errorf("avcodec_send_packet() failed: %d", res)
	}
	return nil
}

func (d *VideoDecoder) ReadFrame() (image.Image, error) {
	if res := C.avcodec_receive_frame(d.ctx, d.frame); res < 0 {
		return nil, fmt.Errorf("avcodec_receive_frame() failed: %d", res)
	}
	switch d.frame.format {
	case C.AV_PIX_FMT_YUV420P, C.AV_PIX_FMT_YUV422P, C.AV_PIX_FMT_YUV444P, C.AV_PIX_FMT_YUV410P, C.AV_PIX_FMT_YUV411P:
		img := &image.YCbCr{
			Y:       (*[1 << 30]uint8)(unsafe.Pointer(d.frame.data[0]))[:d.frame.height*d.frame.linesize[0]],
			Cb:      (*[1 << 30]uint8)(unsafe.Pointer(d.frame.data[1]))[:d.frame.height*d.frame.linesize[1]],
			Cr:      (*[1 << 30]uint8)(unsafe.Pointer(d.frame.data[2]))[:d.frame.height*d.frame.linesize[2]],
			Rect:    image.Rect(0, 0, int(d.frame.width), int(d.frame.height)),
			YStride: int(d.frame.linesize[0]),
			CStride: int(d.frame.linesize[1]),
		}
		switch d.frame.format {
		case C.AV_PIX_FMT_YUV420P:
			img.SubsampleRatio = image.YCbCrSubsampleRatio420
		case C.AV_PIX_FMT_YUV422P:
			img.SubsampleRatio = image.YCbCrSubsampleRatio422
		case C.AV_PIX_FMT_YUV444P:
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
	return nil, fmt.Errorf("unsupported pixel format: %d", d.frame.format)
}
