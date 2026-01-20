package decode

import (
	"fmt"
	"image"
	"unsafe"

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

type LibAVCodecDecoder struct {
	ctx   *C.AVCodecContext
	pkt   *C.AVPacket
	frame *C.AVFrame

	// H264 SPS/PPS tracking
	sps    []byte
	pps    []byte
	hadIDR bool
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

// SetSPSPPS sets default SPS/PPS NAL units for H264 decoding.
// Some UVC devices (like DJI Osmo Action cameras) require manually configured
// SPS/PPS headers. The data should include the 4-byte start code (0x00000001).
func (d *LibAVCodecDecoder) SetSPSPPS(sps, pps []byte) {
	d.sps = make([]byte, len(sps))
	copy(d.sps, sps)
	d.pps = make([]byte, len(pps))
	copy(d.pps, pps)
}

func (d *LibAVCodecDecoder) Write(pkt []byte) (int, error) {
	cdata := C.CBytes(pkt)
	defer C.free(cdata)

	d.pkt.data = (*C.uint8_t)(cdata)
	d.pkt.size = C.int(len(pkt))

	if res := C.avcodec_send_packet(d.ctx, d.pkt); res < 0 {
		return 0, fmt.Errorf("avcodec_send_packet() failed: %d", res)
	}
	return len(pkt), nil
}

// findNALUnits finds all NAL unit boundaries in H264 data
// Returns slice of (start, end, nalType) for each NAL unit
func findNALUnits(data []byte) [][3]int {
	var units [][3]int
	i := 0
	for i < len(data)-4 {
		// Look for start code 0x00000001 or 0x000001
		if data[i] == 0 && data[i+1] == 0 {
			startCodeLen := 0
			if data[i+2] == 0 && data[i+3] == 1 {
				startCodeLen = 4
			} else if data[i+2] == 1 {
				startCodeLen = 3
			}
			if startCodeLen > 0 {
				nalStart := i
				nalType := int(data[i+startCodeLen] & 0x1F)

				// Find end of this NAL unit (start of next or end of data)
				j := i + startCodeLen + 1
				for j < len(data)-3 {
					if data[j] == 0 && data[j+1] == 0 {
						if (j+3 < len(data) && data[j+2] == 0 && data[j+3] == 1) || data[j+2] == 1 {
							break
						}
					}
					j++
				}
				if j >= len(data)-3 {
					j = len(data)
				}
				units = append(units, [3]int{nalStart, j, nalType})
				i = j
				continue
			}
		}
		i++
	}
	return units
}

func (d *LibAVCodecDecoder) WriteUSBFrame(fr *transfers.Frame) error {
	// Concatenate all payload data into a single buffer
	// H264 NAL units can span multiple UVC payloads
	totalSize := 0
	for _, p := range fr.Payloads {
		totalSize += len(p.Data)
	}

	if totalSize == 0 {
		return nil
	}

	buf := make([]byte, totalSize)
	offset := 0
	for _, p := range fr.Payloads {
		copy(buf[offset:], p.Data)
		offset += len(p.Data)
	}

	// Parse NAL units and extract SPS/PPS if present
	nalUnits := findNALUnits(buf)
	hasIDR := false
	for _, unit := range nalUnits {
		nalType := unit[2]
		nalData := buf[unit[0]:unit[1]]

		switch nalType {
		case 7: // SPS
			d.sps = make([]byte, len(nalData))
			copy(d.sps, nalData)
		case 8: // PPS
			d.pps = make([]byte, len(nalData))
			copy(d.pps, nalData)
		case 5: // IDR
			hasIDR = true
		}
	}

	// If this frame has an IDR and we have SPS/PPS, prepend them
	var sendBuf []byte
	if hasIDR && len(d.sps) > 0 && len(d.pps) > 0 {
		sendBuf = make([]byte, len(d.sps)+len(d.pps)+len(buf))
		copy(sendBuf, d.sps)
		copy(sendBuf[len(d.sps):], d.pps)
		copy(sendBuf[len(d.sps)+len(d.pps):], buf)
		d.hadIDR = true
	} else if d.hadIDR {
		// Already had IDR, just send the frame
		sendBuf = buf
	} else {
		// No IDR yet and no SPS/PPS - skip this frame
		return nil
	}

	cdata := C.CBytes(sendBuf)
	defer C.free(cdata)

	d.pkt.data = (*C.uint8_t)(cdata)
	d.pkt.size = C.int(len(sendBuf))

	if res := C.avcodec_send_packet(d.ctx, d.pkt); res < 0 {
		return fmt.Errorf("avcodec_send_packet() failed: %d", res)
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
			Y:       unsafe.Slice((*uint8)(d.frame.data[0]), d.frame.height*d.frame.linesize[0]),
			Cb:      unsafe.Slice((*uint8)(d.frame.data[1]), d.frame.height*d.frame.linesize[1]),
			Cr:      unsafe.Slice((*uint8)(d.frame.data[2]), d.frame.height*d.frame.linesize[2]),
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
			Pix:    unsafe.Slice((*uint8)(d.frame.data[0]), d.frame.height*d.frame.linesize[0]),
			Stride: int(d.frame.linesize[0]),
			Rect:   image.Rect(0, 0, int(d.frame.width), int(d.frame.height)),
		}, nil
	case C.AV_PIX_FMT_BGR24:
		return &BGR{
			Pix:    unsafe.Slice((*uint8)(d.frame.data[0]), d.frame.height*d.frame.linesize[0]),
			Stride: int(d.frame.linesize[0]),
			Rect:   image.Rect(0, 0, int(d.frame.width), int(d.frame.height)),
		}, nil
	case C.AV_PIX_FMT_GRAY8:
		return &image.Gray{
			Pix:    unsafe.Slice((*uint8)(d.frame.data[0]), d.frame.height*d.frame.linesize[0]),
			Stride: int(d.frame.linesize[0]),
			Rect:   image.Rect(0, 0, int(d.frame.width), int(d.frame.height)),
		}, nil
	case C.AV_PIX_FMT_GRAY16BE:
		return &image.Gray16{
			Pix:    unsafe.Slice((*uint8)(d.frame.data[0]), d.frame.height*d.frame.linesize[0]),
			Stride: int(d.frame.linesize[0]),
			Rect:   image.Rect(0, 0, int(d.frame.width), int(d.frame.height)),
		}, nil
	}
	return nil, fmt.Errorf("unsupported pixel format: %s", C.GoString(C.av_get_pix_fmt_name(int32(d.frame.format))))
}
