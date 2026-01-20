package main

import (
	"flag"
	"fmt"
	"image"
	"io"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/kevmo314/go-uvc"
	"github.com/kevmo314/go-uvc/pkg/decode"
	"github.com/kevmo314/go-uvc/pkg/descriptors"
	"github.com/kevmo314/go-uvc/pkg/transfers"
)

var sps = []byte{
	0x00, 0x00, 0x00, 0x01, 0x67, 0x64, 0x00, 0x34,
	0xAC, 0x4D, 0x00, 0xF0, 0x04, 0x4F, 0xCB, 0x35,
	0x01, 0x01, 0x01, 0x40, 0x00, 0x00, 0xFA, 0x00,
	0x00, 0x3A, 0x98, 0x03, 0xC7, 0x0C, 0xA8,
}
var pps = []byte{
	0x00, 0x00, 0x00, 0x01, 0x68, 0xEE, 0xBC, 0xB0,
}

func main() {
	path := flag.String("path", "/dev/bus/usb/001/046", "path")
	frames := flag.Int("frames", 100, "frames to profile")
	flag.Parse()

	fd, _ := os.OpenFile(*path, os.O_RDWR, 0)
	defer fd.Close()
	dev, _ := uvc.NewUVCDevice(fd.Fd())
	defer dev.Close()
	info, _ := dev.DeviceInfo()

	var si *transfers.StreamingInterface
	var sf descriptors.FormatDescriptor
	var fr descriptors.FrameDescriptor

	for _, s := range info.StreamingInterfaces {
		for _, desc := range s.Descriptors {
			if f, ok := desc.(*descriptors.FrameBasedFormatDescriptor); ok {
				for _, dd := range s.Descriptors {
					if ff, ok := dd.(*descriptors.FrameBasedFrameDescriptor); ok {
						if ff.Width == 1920 && ff.Height == 1080 {
							si, sf, fr = s, f, ff
							goto found
						}
					}
				}
			}
		}
	}

found:
	reader, _ := si.ClaimFrameReader(sf.Index(), fr.Index())
	defer reader.Close()

	decoder, _ := decode.NewH264Decoder()
	defer decoder.Close()
	decoder.SetSPSPPS(sps, pps)

	log.Printf("Profiling %d frames...\n", *frames)

	var totalRead, totalReadAll, totalDecode, totalConvert time.Duration
	var readCount, decodeCount, convertCount int

	start := time.Now()

	for i := 0; i < *frames; i++ {
		// Time USB read
		t0 := time.Now()
		frame, err := reader.ReadFrame()
		readTime := time.Since(t0)
		if err != nil {
			continue
		}
		totalRead += readTime
		readCount++

		// Time io.ReadAll
		t1 := time.Now()
		data, _ := io.ReadAll(frame)
		readAllTime := time.Since(t1)
		totalReadAll += readAllTime

		if len(data) == 0 {
			continue
		}

		// Reconstruct frame for decoder
		frame.Payloads = []*transfers.Payload{{Data: data}}

		// Time decode
		t2 := time.Now()
		err = decoder.WriteUSBFrame(frame)
		if err != nil {
			continue
		}
		img, err := decoder.ReadFrame()
		decodeTime := time.Since(t2)
		if err != nil {
			if err != decode.ErrEAGAIN {
				continue
			}
			continue
		}
		totalDecode += decodeTime
		decodeCount++

		// Time RGBA conversion (using optimized parallel conversion)
		t3 := time.Now()
		bounds := img.Bounds()
		rgba := image.NewRGBA(bounds)
		if ycbcr, ok := img.(*image.YCbCr); ok {
			convertYCbCrToRGBAParallel(ycbcr, rgba)
		}
		convertTime := time.Since(t3)
		totalConvert += convertTime
		convertCount++
	}

	elapsed := time.Since(start)

	fmt.Println("\n=== Profile Results ===")
	fmt.Printf("Total time: %v for %d frames\n", elapsed, *frames)
	fmt.Printf("Effective FPS: %.2f\n\n", float64(*frames)/elapsed.Seconds())

	if readCount > 0 {
		fmt.Printf("USB ReadFrame:\n")
		fmt.Printf("  Total: %v, Count: %d, Avg: %v\n", totalRead, readCount, totalRead/time.Duration(readCount))
		fmt.Printf("  io.ReadAll: Total: %v, Avg: %v\n", totalReadAll, totalReadAll/time.Duration(readCount))
	}

	if decodeCount > 0 {
		fmt.Printf("\nH264 Decode:\n")
		fmt.Printf("  Total: %v, Count: %d, Avg: %v\n", totalDecode, decodeCount, totalDecode/time.Duration(decodeCount))
	}

	if convertCount > 0 {
		fmt.Printf("\nRGBA Conversion:\n")
		fmt.Printf("  Total: %v, Count: %d, Avg: %v\n", totalConvert, convertCount, totalConvert/time.Duration(convertCount))
	}

	fmt.Println("\n=== Breakdown ===")
	total := totalRead + totalReadAll + totalDecode + totalConvert
	fmt.Printf("USB Read:     %.1f%%\n", float64(totalRead)/float64(total)*100)
	fmt.Printf("io.ReadAll:   %.1f%%\n", float64(totalReadAll)/float64(total)*100)
	fmt.Printf("H264 Decode:  %.1f%%\n", float64(totalDecode)/float64(total)*100)
	fmt.Printf("RGBA Convert: %.1f%%\n", float64(totalConvert)/float64(total)*100)
}

// Parallel YCbCr to RGBA conversion
func convertYCbCrToRGBAParallel(src *image.YCbCr, dst *image.RGBA) {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	numWorkers := runtime.NumCPU()
	rowsPerWorker := (h + numWorkers - 1) / numWorkers

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		startY := i * rowsPerWorker
		endY := startY + rowsPerWorker
		if endY > h {
			endY = h
		}
		if startY >= h {
			break
		}

		wg.Add(1)
		go func(startY, endY int) {
			defer wg.Done()
			convertYCbCrRows(src, dst, w, startY, endY)
		}(startY, endY)
	}
	wg.Wait()
}

func convertYCbCrRows(src *image.YCbCr, dst *image.RGBA, w, startY, endY int) {
	for y := startY; y < endY; y++ {
		yi := y * src.YStride
		var ci int
		switch src.SubsampleRatio {
		case image.YCbCrSubsampleRatio420:
			ci = (y / 2) * src.CStride
		case image.YCbCrSubsampleRatio422:
			ci = y * src.CStride
		case image.YCbCrSubsampleRatio444:
			ci = y * src.CStride
		default:
			ci = (y / 2) * src.CStride
		}

		for x := 0; x < w; x++ {
			yy := int(src.Y[yi+x])

			var cx int
			switch src.SubsampleRatio {
			case image.YCbCrSubsampleRatio420, image.YCbCrSubsampleRatio422:
				cx = x / 2
			default:
				cx = x
			}

			cb := int(src.Cb[ci+cx]) - 128
			cr := int(src.Cr[ci+cx]) - 128

			r := yy + 91881*cr/65536
			g := yy - 22554*cb/65536 - 46802*cr/65536
			b := yy + 116130*cb/65536

			if r < 0 {
				r = 0
			} else if r > 255 {
				r = 255
			}
			if g < 0 {
				g = 0
			} else if g > 255 {
				g = 255
			}
			if b < 0 {
				b = 0
			} else if b > 255 {
				b = 255
			}

			i := (y*w + x) * 4
			dst.Pix[i] = uint8(r)
			dst.Pix[i+1] = uint8(g)
			dst.Pix[i+2] = uint8(b)
			dst.Pix[i+3] = 255
		}
	}
}
