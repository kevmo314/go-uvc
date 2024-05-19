package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"os"
	"sync/atomic"
	"time"

	"golang.org/x/image/draw"

	"github.com/kevmo314/go-uvc"
	"github.com/kevmo314/go-uvc/pkg/descriptors"
	"github.com/rivo/tview"
)

func main() {
	if len(os.Args) < 2 {
		panic("usage: ./inspect <usb device path>")
	}
	path := os.Args[1]

	fd, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		panic(err)
	}
	defer fd.Close()

	dev, err := uvc.NewUVCDevice(fd.Fd())
	if err != nil {
		panic(err)
	}

	go dev.EventLoop()

	info, err := dev.DeviceInfo()
	if err != nil {
		panic(err)
	}

	app := tview.NewApplication()

	streamingIfaces := tview.NewList()
	streamingIfaces.SetBorder(true).SetTitle("Streaming Interfaces")

	controlIfaces := tview.NewList().ShowSecondaryText(false)
	controlIfaces.SetBorder(true).SetTitle("Control Interfaces")

	ifaces := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(streamingIfaces, 0, 1, true).AddItem(controlIfaces, 0, 1, false)

	formats := tview.NewList()
	formats.SetBorder(true).SetTitle("Formats")

	frames := tview.NewList()
	frames.SetBorder(true).SetTitle("Frames")

	preview := tview.NewImage()
	preview.SetColors(256).SetDithering(tview.DitheringNone).SetBorder(true).SetTitle("Preview")

	logText := tview.NewTextView()
	logText.SetMaxLines(3).SetBorder(true).SetTitle("Log")

	log.SetOutput(logText)

	active := &atomic.Uint32{}

	for _, si := range info.StreamingInterfaces {
		streamingIfaces.AddItem(fmt.Sprintf("Interface %d", si.InterfaceNumber()), fmt.Sprintf("v%s", si.UVCVersionString()), 0, func() {
			for fdIndex, d := range si.Descriptors {
				if fd, ok := d.(descriptors.FormatDescriptor); ok {
					formats.AddItem(formatDescriptorTitle(fd), formatDescriptorSubtitle(fd), 0, func() {
						frs := si.Descriptors[fdIndex+1 : fdIndex+int(NumFrameDescriptors(fd))+1]
						for _, fr := range frs {
							if fr, ok := fr.(descriptors.FrameDescriptor); ok {
								frames.AddItem(frameDescriptorTitle(fr), frameDescriptorSubtitle(fr), 0, func() {
									track := active.Add(1)
									app.SetFocus(preview)
									reader, err := si.ClaimFrameReader(fd.Index(), fr.Index())
									if err != nil {
										panic(err)
									}
									go func() {
										log.Printf("starting frame reader %d", track)
										defer reader.Close()
										t0 := time.Now().Add(-1 * time.Second)
										for i := 0; active.Load() == track; i++ {
											fr, err := reader.ReadFrame()
											if err != nil {
												panic(err)
											}
											t1 := time.Now()
											if t1.Sub(t0) < 50*time.Millisecond {
												continue
											}
											t0 = t1
											img, err := jpeg.Decode(bytes.NewReader(fr.Data))
											if err != nil {
												continue
											}
											w := 64
											h := img.Bounds().Dy() * w / img.Bounds().Dx()
											preview.SetImage(resize(img, w, h))
											app.ForceDraw()
										}
									}()
								})
							}
						}
						app.SetFocus(frames)
					})
				}
			}
			app.SetFocus(formats)
		})
	}

	for _, ci := range info.ControlInterfaces {
		controlIfaces.AddItem(controlInterfaceTitle(ci), "", 0, func() {
		})
	}

	// Create the layout.

	flex := tview.NewFlex().
		AddItem(ifaces, 0, 1, true).
		AddItem(formats, 0, 1, false).
		AddItem(frames, 0, 1, false).
		AddItem(preview, 0, 3, false)

	if err := app.SetRoot(tview.NewFlex().SetDirection(tview.FlexRow).AddItem(flex, 0, 1, true).AddItem(logText, 5, 0, false), true).Run(); err != nil {
		panic(err)
	}
}

func resize(img image.Image, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.NearestNeighbor.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
	return dst
}

func NumFrameDescriptors(fd descriptors.FormatDescriptor) uint8 {
	// darn you golang and your lack of structural typing.
	switch fd := fd.(type) {
	case *descriptors.MJPEGFormatDescriptor:
		return fd.NumFrameDescriptors
	case *descriptors.H264FormatDescriptor:
		return fd.NumFrameDescriptors
	case *descriptors.VP8FormatDescriptor:
		return fd.NumFrameDescriptors
	case *descriptors.UncompressedFormatDescriptor:
		return fd.NumFrameDescriptors
	case *descriptors.FrameBasedFormatDescriptor:
		return fd.NumFrameDescriptors
	default:
		return 0
	}
}

func formatDescriptorTitle(fd descriptors.FormatDescriptor) string {
	switch fd := fd.(type) {
	case *descriptors.MJPEGFormatDescriptor:
		return fmt.Sprintf("MJPEG (%d formats)", fd.NumFrameDescriptors)
	case *descriptors.H264FormatDescriptor:
		return fmt.Sprintf("H264 (%d formats)", fd.NumFrameDescriptors)
	case *descriptors.VP8FormatDescriptor:
		return fmt.Sprintf("VP8 (%d formats)", fd.NumFrameDescriptors)
	case *descriptors.DVFormatDescriptor:
		return "DV"
	case *descriptors.UncompressedFormatDescriptor:
		return "Uncompressed"
	case *descriptors.MPEG2TSFormatDescriptor:
		return "MPEG2TS"
	case *descriptors.FrameBasedFormatDescriptor:
		fourcc, err := fd.FourCC()
		if err != nil {
			return fmt.Sprintf("Frame-Based (%d formats)", fd.NumFrameDescriptors)
		}
		return fmt.Sprintf("Frame-Based %s (%d formats)", fourcc, fd.NumFrameDescriptors)
	case *descriptors.StreamBasedFormatDescriptor:
		return "Stream-Based"
	default:
		return "Unknown"
	}
}

func formatDescriptorSubtitle(fd descriptors.FormatDescriptor) string {
	switch fd := fd.(type) {
	case *descriptors.MJPEGFormatDescriptor:
		return fmt.Sprintf("Aspect Ratio: %d:%d", fd.AspectRatioX, fd.AspectRatioY)
	case *descriptors.H264FormatDescriptor:
		return ""
	case *descriptors.VP8FormatDescriptor:
		return fmt.Sprintf("Max Mbps: %d", fd.MaxMBPerSec)
	case *descriptors.DVFormatDescriptor:
		return fmt.Sprintf("Format Type: %d", fd.FormatType)
	case *descriptors.UncompressedFormatDescriptor:
		return fd.GUIDFormat.String()
	case *descriptors.MPEG2TSFormatDescriptor:
		return fd.GUIDStrideFormat.String()
	case *descriptors.FrameBasedFormatDescriptor:
		return fmt.Sprintf("%s, Aspect Ratio: %d:%d, bpp: %d", fd.GUIDFormat, fd.AspectRatioX, fd.AspectRatioY, fd.BitsPerPixel)
	case *descriptors.StreamBasedFormatDescriptor:
		return fd.GUIDFormat.String()
	default:
		return "Unknown"
	}
}

func frameDescriptorTitle(fd descriptors.FrameDescriptor) string {
	switch fd := fd.(type) {
	case *descriptors.MJPEGFrameDescriptor:
		return fmt.Sprintf("MJPEG (%dx%d)", fd.Width, fd.Height)
	case *descriptors.H264FrameDescriptor:
		return fmt.Sprintf("H264 (%dx%d)", fd.Width, fd.Height)
	case *descriptors.VP8FrameDescriptor:
		return fmt.Sprintf("VP8 (%dx%d)", fd.Width, fd.Height)
	case *descriptors.UncompressedFrameDescriptor:
		return fmt.Sprintf("Uncompressed (%dx%d)", fd.Width, fd.Height)
	case *descriptors.FrameBasedFrameDescriptor:
		return fmt.Sprintf("Frame-Based (%dx%d)", fd.Width, fd.Height)
	default:
		return "Unknown"
	}
}

func frameDescriptorSubtitle(fd descriptors.FrameDescriptor) string {
	switch fd := fd.(type) {
	case *descriptors.MJPEGFrameDescriptor:
		return fmt.Sprintf("Bitrate: %d-%d Mbps", fd.MinBitRate, fd.MaxBitRate)
	case *descriptors.H264FrameDescriptor:
		return fmt.Sprintf("Level: %x, Profile: %x", fd.LevelIDC, fd.Profile)
	case *descriptors.VP8FrameDescriptor:
		return fmt.Sprintf("Bitrate: %d-%d Mbps", fd.MinBitRate, fd.MaxBitRate)
	case *descriptors.UncompressedFrameDescriptor:
		return fmt.Sprintf("Bitrate: %d-%d Mbps", fd.MinBitRate, fd.MaxBitRate)
	case *descriptors.FrameBasedFrameDescriptor:
		return fmt.Sprintf("Bitrate: %d-%d Mbps", fd.MinBitRate, fd.MaxBitRate)
	default:
		return "Unknown"
	}
}

func controlInterfaceTitle(ci descriptors.ControlInterface) string {
	switch ci.(type) {
	case *descriptors.HeaderDescriptor:
		return "Header"
	case *descriptors.InputTerminalDescriptor:
		return "Input Terminal"
	case *descriptors.OutputTerminalDescriptor:
		return "Output Terminal"
	case *descriptors.SelectorUnitDescriptor:
		return "Selector Unit"
	case *descriptors.ProcessingUnitDescriptor:
		return "Processing Unit"
	case *descriptors.EncodingUnitDescriptor:
		return "Encoding Unit"
	case *descriptors.ExtensionUnitDescriptor:
		return "Extension Unit"
	default:
		return "Unknown"
	}
}