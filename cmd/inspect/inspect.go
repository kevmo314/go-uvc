package main

import (
	"flag"
	"fmt"
	"image"
	"log"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"golang.org/x/image/draw"

	"github.com/gdamore/tcell/v2"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kevmo314/go-uvc"
	"github.com/kevmo314/go-uvc/pkg/decode"
	"github.com/kevmo314/go-uvc/pkg/descriptors"
	"github.com/rivo/tview"
)

type Display struct {
	frame atomic.Value
}

func (g *Display) Update() error {
	return nil
}

func (g *Display) Draw(screen *ebiten.Image) {
	screen.DrawImage(g.frame.Load().(*ebiten.Image), &ebiten.DrawImageOptions{})
}

func (g *Display) Layout(outsideWidth, outsideHeight int) (int, int) {
	frame := g.frame.Load().(*ebiten.Image)
	return frame.Bounds().Dx(), frame.Bounds().Dy()
}

func main() {
	path := flag.String("path", "", "path to the usb device")
	render := flag.Bool("render", false, "render the frames to screen (higher performance but requires a display)")

	flag.Parse()

	fd, err := os.OpenFile(*path, os.O_RDWR, 0)
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

	controlRequests := tview.NewList().ShowSecondaryText(false)
	controlRequests.SetBorder(true).SetTitle("Control Requests")

	ifaces := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(streamingIfaces, 0, 1, true).AddItem(controlIfaces, 0, 1, false)

	secondColumn := tview.NewFlex()

	formats := tview.NewList()
	formats.SetBorder(true).SetTitle("Formats")

	secondColumn.SetDirection(tview.FlexRow).AddItem(formats, 0, 1, false).AddItem(controlRequests, 0, 1, false)

	frames := tview.NewList()
	frames.SetBorder(true).SetTitle("Frames")

	preview := tview.NewImage()
	preview.SetColors(256).SetDithering(tview.DitheringNone).SetBorder(true).SetTitle("Preview")

	logText := tview.NewTextView()
	logText.SetMaxLines(10).SetBorder(true).SetTitle("Log")

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
									reader, err := si.ClaimFrameReader(fd.Index(), fr.Index())
									if err != nil {
										log.Printf("error claiming frame reader: %s", err)
										return
									}
									decoder, err := decode.NewFrameReaderDecoder(reader, fd, fr)
									if err != nil {
										log.Printf("error creating decoder: %s", err)
										return
									}
									if *render {
										g := &Display{}
										go func() {
											defer reader.Close()
											for active.Load() == track {
												img, err := decoder.ReadFrame()
												if err != nil {
													log.Printf("error reading frame: %s", err)
													continue
												}
												if g.frame.Swap(ebiten.NewImageFromImage(img)) == nil {
													go func() {
														if err := ebiten.RunGame(g); err != nil {
															log.Printf("ebiten error: %s", err)
														}
													}()
												}
											}
										}()
									} else {
										go func() {
											defer reader.Close()
											t0 := time.Now().Add(-1 * time.Second)
											for active.Load() == track {
												img, err := decoder.ReadFrame()
												if err != nil {
													log.Printf("error reading frame: %s", err)
													return
												}
												t1 := time.Now()
												if t1.Sub(t0) < 50*time.Millisecond {
													continue
												}
												t0 = t1
												w := 64
												h := img.Bounds().Dy() * w / img.Bounds().Dx()
												preview.SetImage(resize(img, w, h))
												app.ForceDraw()
											}
										}()
									}
									app.SetFocus(controlIfaces)
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
			switch ci.Descriptor.(type) {
			case *descriptors.CameraTerminalDescriptor:
				app.SetFocus(controlRequests)
				controlRequests.AddItem("Zoom Absolute", "", 0, func() {
					controlRequestInput := tview.NewInputField()

					controlRequestInput.SetLabel("Enter zoom value (>= 100): ").
						SetFieldWidth(10).
						SetAcceptanceFunc(tview.InputFieldInteger).
						SetDoneFunc(func(key tcell.Key) {
							capture, err := strconv.ParseUint(controlRequestInput.GetText(), 10, 16)
							if err != nil {
								log.Printf("failed parsing value %s", err)
								return
							}
							setControl := &descriptors.ZoomAbsoluteControl{ObjectiveFocalLength: uint16(capture)}
							err = ci.CameraTerminal.Set(setControl)
							if err != nil {
								log.Printf("control request failed %s", err)
							}
							secondColumn.RemoveItem(controlRequestInput)
							app.SetFocus(controlRequests)
						})
					secondColumn.AddItem(controlRequestInput, 0, 1, false)
					app.SetFocus(controlRequestInput)
				})
			}
		})
	}

	// Create the layout.

	flex := tview.NewFlex().
		AddItem(ifaces, 0, 1, true).
		AddItem(secondColumn, 0, 1, false).
		AddItem(frames, 0, 1, false)

	if !*render {
		flex.AddItem(preview, 0, 3, false)
	}

	if err := app.SetRoot(tview.NewFlex().SetDirection(tview.FlexRow).AddItem(flex, 0, 1, true).AddItem(logText, 10, 0, false), true).Run(); err != nil {
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

func controlInterfaceTitle(ci *uvc.ControlInterface) string {
	switch ci.Descriptor.(type) {
	case *descriptors.HeaderDescriptor:
		return "Header"
	case *descriptors.InputTerminalDescriptor:
		return "Input Terminal"
	case *descriptors.CameraTerminalDescriptor:
		return "Camera Terminal"
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
