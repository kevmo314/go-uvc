package main

import (
	"flag"
	"image"
	"log"
	"os"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/kevmo314/go-uvc"
	"github.com/kevmo314/go-uvc/pkg/decode"
	"github.com/kevmo314/go-uvc/pkg/descriptors"
	"github.com/kevmo314/go-uvc/pkg/transfers"
)

// SPS/PPS for 1920x1080 H264
var sps1080p = []byte{
	0x00, 0x00, 0x00, 0x01, 0x67, 0x64, 0x00, 0x34,
	0xAC, 0x4D, 0x00, 0xF0, 0x04, 0x4F, 0xCB, 0x35,
	0x01, 0x01, 0x01, 0x40, 0x00, 0x00, 0xFA, 0x00,
	0x00, 0x3A, 0x98, 0x03, 0xC7, 0x0C, 0xA8,
}
var pps1080p = []byte{
	0x00, 0x00, 0x00, 0x01, 0x68, 0xEE, 0xBC, 0xB0,
}

var sps720p = []byte{
	0x00, 0x00, 0x00, 0x01, 0x67, 0x64, 0x00, 0x28,
	0xAC, 0x4D, 0x00, 0xA0, 0x02, 0xCF, 0x96, 0x6E,
	0x02, 0x02, 0x02, 0x80, 0x00, 0x01, 0xF4, 0x00,
	0x00, 0x75, 0x30, 0x07, 0x8C, 0x18, 0x50,
}
var pps720p = []byte{
	0x00, 0x00, 0x00, 0x01, 0x68, 0xEE, 0xBC, 0xB0,
}

func main() {
	runtime.LockOSThread() // SDL requires main thread

	path := flag.String("path", "/dev/bus/usb/001/052", "path to USB device")
	width := flag.Int("width", 1920, "video width")
	height := flag.Int("height", 1080, "video height")
	flag.Parse()

	// Initialize SDL
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		log.Fatalf("Failed to init SDL: %v", err)
	}
	defer sdl.Quit()

	// Open USB device
	fd, err := os.OpenFile(*path, os.O_RDWR, 0)
	if err != nil {
		log.Fatalf("Failed to open device: %v", err)
	}
	defer fd.Close()

	dev, err := uvc.NewUVCDevice(fd.Fd())
	if err != nil {
		log.Fatalf("Failed to create UVC device: %v", err)
	}
	defer dev.Close()

	info, err := dev.DeviceInfo()
	if err != nil {
		log.Fatalf("Failed to get device info: %v", err)
	}

	// Find H264 format
	var selectedInterface *transfers.StreamingInterface
	var selectedFormat descriptors.FormatDescriptor
	var selectedFrame descriptors.FrameDescriptor

	for _, si := range info.StreamingInterfaces {
		for _, desc := range si.Descriptors {
			if fd, ok := desc.(*descriptors.FrameBasedFormatDescriptor); ok {
				for _, d := range si.Descriptors {
					if fr, ok := d.(*descriptors.FrameBasedFrameDescriptor); ok {
						if fr.Width == uint16(*width) && fr.Height == uint16(*height) {
							selectedInterface = si
							selectedFormat = fd
							selectedFrame = fr
							goto found
						}
					}
				}
			}
		}
	}

found:
	if selectedInterface == nil {
		log.Fatal("No matching format found")
	}

	// Create decoder
	h264Decoder, err := decode.NewH264Decoder()
	if err != nil {
		log.Fatalf("Failed to create decoder: %v", err)
	}
	defer h264Decoder.Close()

	// Set SPS/PPS
	fbFrame := selectedFrame.(*descriptors.FrameBasedFrameDescriptor)
	if fbFrame.Width == 1920 {
		h264Decoder.SetSPSPPS(sps1080p, pps1080p)
	} else {
		h264Decoder.SetSPSPPS(sps720p, pps720p)
	}

	// Claim reader
	reader, err := selectedInterface.ClaimFrameReader(selectedFormat.Index(), selectedFrame.Index())
	if err != nil {
		log.Fatalf("Failed to claim reader: %v", err)
	}
	defer reader.Close()

	// Create SDL window
	window, err := sdl.CreateWindow("Osmo Action 4",
		sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		int32(*width), int32(*height), sdl.WINDOW_SHOWN)
	if err != nil {
		log.Fatalf("Failed to create window: %v", err)
	}
	defer window.Destroy()

	// Create renderer with vsync disabled
	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		log.Fatalf("Failed to create renderer: %v", err)
	}
	defer renderer.Destroy()

	// Create texture for YUV420 (direct from decoder, no conversion needed!)
	texture, err := renderer.CreateTexture(sdl.PIXELFORMAT_IYUV,
		sdl.TEXTUREACCESS_STREAMING, int32(*width), int32(*height))
	if err != nil {
		log.Fatalf("Failed to create texture: %v", err)
	}
	defer texture.Destroy()

	// Frame channel
	type yuvFrame struct {
		y, u, v    []byte
		yStride    int
		uvStride   int
	}
	frameChan := make(chan *yuvFrame, 2)

	// Reader goroutine
	go func() {
		var lastLog time.Time
		var frameCount int

		for {
			frame, err := reader.ReadFrame()
			if err != nil {
				continue
			}

			if err := h264Decoder.WriteUSBFrame(frame); err != nil {
				continue
			}

			img, err := h264Decoder.ReadFrame()
			if err != nil {
				continue
			}

			// Get YCbCr directly (no RGBA conversion!)
			if ycbcr, ok := img.(*image.YCbCr); ok {
				yf := &yuvFrame{
					y:        ycbcr.Y,
					u:        ycbcr.Cb,
					v:        ycbcr.Cr,
					yStride:  ycbcr.YStride,
					uvStride: ycbcr.CStride,
				}
				select {
				case frameChan <- yf:
				default:
				}
			}

			frameCount++
			if time.Since(lastLog) >= time.Second {
				log.Printf("Decode FPS: %d", frameCount)
				frameCount = 0
				lastLog = time.Now()
			}
		}
	}()

	// Main loop
	var displayCount int
	var lastFPS time.Time
	var mu sync.Mutex
	var latestFrame *yuvFrame

	// Frame receiver
	go func() {
		for f := range frameChan {
			mu.Lock()
			latestFrame = f
			mu.Unlock()
		}
	}()

	running := true
	for running {
		// Handle events
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch event.(type) {
			case *sdl.QuitEvent:
				running = false
			}
		}

		// Get latest frame
		mu.Lock()
		frame := latestFrame
		latestFrame = nil
		mu.Unlock()

		if frame != nil {
			// Update texture directly with YUV data (GPU does conversion!)
			texture.UpdateYUV(nil,
				frame.y, frame.yStride,
				frame.u, frame.uvStride,
				frame.v, frame.uvStride)

			displayCount++
		}

		// Render
		renderer.Clear()
		renderer.Copy(texture, nil, nil)
		renderer.Present()

		// FPS logging
		if time.Since(lastFPS) >= time.Second {
			log.Printf("Display FPS: %d", displayCount)
			displayCount = 0
			lastFPS = time.Now()
		}

		// Small sleep to not spin CPU (targeting ~60fps render loop)
		sdl.Delay(1)
	}
}

// For unsafe pointer conversion
var _ = unsafe.Pointer(nil)
