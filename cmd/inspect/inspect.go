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
	controlRequests.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			controlRequests.Clear()
			app.SetFocus(controlIfaces)
			return nil
		}
		return event
	})

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

				controls := ci.CameraTerminal.GetSupportedControls()
				uiControls := formatCameraControls(ci, app, secondColumn, controls)
				for _, option := range uiControls {
					controlRequests.AddItem(option.title, "", 0, option.handler)
				}
			case *descriptors.ProcessingUnitDescriptor:
				app.SetFocus(controlRequests)

				controls := ci.ProcessingUnit.GetSupportedControls()
				uiControls := formatProcessingControls(ci, app, secondColumn, controls)
				for _, option := range uiControls {
					controlRequests.AddItem(option.title, "", 0, option.handler)
				}
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

type ControlRequestListItem struct {
	title   string
	handler func()
}

func formatProcessingControls(ci *uvc.ControlInterface, app *tview.Application, secondColumn *tview.Flex,
	controls []descriptors.ProcessingUnitControlDescriptor) []*ControlRequestListItem {
	var uiControls []*ControlRequestListItem
	for _, control := range controls {
		controlUI := generatePUDUI(control, app, ci, secondColumn)

		if controlUI != nil {
			uiControls = append(uiControls, controlUI)
		}

	}
	return uiControls
}

func generatePUDUI(control descriptors.ProcessingUnitControlDescriptor, app *tview.Application, ci *uvc.ControlInterface, secondColumn *tview.Flex) *ControlRequestListItem {
	switch control.(type) {
	case *descriptors.BacklightCompensationControl:
	case *descriptors.BrightnessControl:
		return &ControlRequestListItem{
			title: "Brightness",
			handler: func() {
				initFocus := app.GetFocus()
				controlRequestInput := tview.NewInputField()
				controlRequestInput.SetLabel("Enter brightness value: ").
					SetFieldWidth(10).
					SetAcceptanceFunc(tview.InputFieldInteger).
					SetDoneFunc(func(key tcell.Key) {
						capture, err := strconv.ParseUint(controlRequestInput.GetText(), 10, 16)
						if err != nil {
							log.Printf("failed parsing value %s", err)
						}
						setBrightness := &descriptors.BrightnessControl{Brightness: uint16(capture)}
						err = ci.ProcessingUnit.Set(setBrightness)
						if err != nil {
							log.Printf("brightness request failed %s", err)
						}
						secondColumn.RemoveItem(controlRequestInput)
						app.SetFocus(initFocus)
					})
				secondColumn.AddItem(controlRequestInput, 1, 1, false)
				app.SetFocus(controlRequestInput)
			},
		}
	case *descriptors.ContrastControl:
		return &ControlRequestListItem{
			title: "Contrast",
			handler: func() {
				initFocus := app.GetFocus()
				controlRequestInput := tview.NewInputField()
				controlRequestInput.SetLabel("Enter contrast value: ").
					SetFieldWidth(10).
					SetAcceptanceFunc(tview.InputFieldInteger).
					SetDoneFunc(func(key tcell.Key) {
						capture, err := strconv.ParseUint(controlRequestInput.GetText(), 10, 16)
						if err != nil {
							log.Printf("failed parsing value %s", err)
						}
						setContrast := &descriptors.ContrastControl{Contrast: uint16(capture)}
						err = ci.ProcessingUnit.Set(setContrast)
						if err != nil {
							log.Printf("contrast request failed %s", err)
						}
						secondColumn.RemoveItem(controlRequestInput)
						app.SetFocus(initFocus)
					})
				secondColumn.AddItem(controlRequestInput, 1, 1, false)
				app.SetFocus(controlRequestInput)
			},
		}
	case *descriptors.ContrastAutoControl:
	case *descriptors.GainControl:
		return &ControlRequestListItem{
			title: "Gain",
			handler: func() {
				initFocus := app.GetFocus()
				controlRequestInput := tview.NewInputField()
				controlRequestInput.SetLabel("Enter value: ").
					SetFieldWidth(10).
					SetAcceptanceFunc(tview.InputFieldInteger).
					SetDoneFunc(func(key tcell.Key) {
						capture, err := strconv.ParseUint(controlRequestInput.GetText(), 10, 16)
						if err != nil {
							log.Printf("failed parsing value %s", err)
						}
						setGain := &descriptors.GainControl{Gain: uint16(capture)}
						err = ci.ProcessingUnit.Set(setGain)
						if err != nil {
							log.Printf("gain request failed %s", err)
						}
						secondColumn.RemoveItem(controlRequestInput)
						app.SetFocus(initFocus)
					})
				secondColumn.AddItem(controlRequestInput, 1, 1, false)
				app.SetFocus(controlRequestInput)
			}}
	case *descriptors.PowerLineFrequencyControl:
	case *descriptors.HueControl:
	case *descriptors.HueAutoControl:
	case *descriptors.SaturationControl:
		return &ControlRequestListItem{
			title: "Saturation",
			handler: func() {
				initFocus := app.GetFocus()
				controlRequestInput := tview.NewInputField()
				controlRequestInput.SetLabel("Enter saturation value: ").
					SetFieldWidth(10).
					SetAcceptanceFunc(tview.InputFieldInteger).
					SetDoneFunc(func(key tcell.Key) {
						capture, err := strconv.ParseUint(controlRequestInput.GetText(), 10, 16)
						if err != nil {
							log.Printf("failed parsing value %s", err)
						}
						setSaturation := &descriptors.SaturationControl{Saturation: uint16(capture)}
						err = ci.ProcessingUnit.Set(setSaturation)
						if err != nil {
							log.Printf("saturation request failed %s", err)
						}
						secondColumn.RemoveItem(controlRequestInput)
						app.SetFocus(initFocus)
					})
				secondColumn.AddItem(controlRequestInput, 1, 1, false)
				app.SetFocus(controlRequestInput)
			}}
	case *descriptors.SharpnessControl:
		return &ControlRequestListItem{
			title: "Sharpness",
			handler: func() {
				initFocus := app.GetFocus()
				controlRequestInput := tview.NewInputField()
				controlRequestInput.SetLabel("Enter sharpness value: ").
					SetFieldWidth(10).
					SetAcceptanceFunc(tview.InputFieldInteger).
					SetDoneFunc(func(key tcell.Key) {
						capture, err := strconv.ParseUint(controlRequestInput.GetText(), 10, 16)
						if err != nil {
							log.Printf("failed parsing value %s", err)
						}
						setSaturation := &descriptors.SharpnessControl{Sharpness: uint16(capture)}
						err = ci.ProcessingUnit.Set(setSaturation)
						if err != nil {
							log.Printf("sharpness request failed %s", err)
						}
						secondColumn.RemoveItem(controlRequestInput)
						app.SetFocus(initFocus)
					})
				secondColumn.AddItem(controlRequestInput, 1, 1, false)
				app.SetFocus(controlRequestInput)
			}}
	case *descriptors.GammaControl:
	case *descriptors.WhiteBalanceTemperatureControl:
	case *descriptors.WhiteBalanceTemperatureAutoControl:
	case *descriptors.WhiteBalanceComponentControl:
	case *descriptors.WhiteBalanceComponentAutoControl:
	case *descriptors.DigitalMultiplerControl:
	case *descriptors.DigitalMultiplerLimitControl:
	case *descriptors.AnalogVideoStandardControl:
	case *descriptors.AnalogVideoLockStatusControl:
	}
	return nil
}

func formatCameraControls(ci *uvc.ControlInterface, app *tview.Application, secondColumn *tview.Flex,
	controls []descriptors.CameraTerminalControlDescriptor) []*ControlRequestListItem {
	var uiControls []*ControlRequestListItem
	for _, control := range controls {
		controlUI := generateCTUI(control, app, ci, secondColumn)

		if controlUI != nil {
			uiControls = append(uiControls, controlUI)
		}
	}
	return uiControls
}

func generateCTUI(control descriptors.CameraTerminalControlDescriptor, app *tview.Application,
	ci *uvc.ControlInterface, secondColumn *tview.Flex) *ControlRequestListItem {
	switch control.(type) {
	case *descriptors.ScanningModeControl:
	case *descriptors.AutoExposurePriorityControl:
	case *descriptors.DigitalWindowControl:
	case *descriptors.PrivacyControl:
	case *descriptors.FocusAbsoluteControl:
		return &ControlRequestListItem{
			title: "Focus (Absolute)",
			handler: func() {
				initFocus := app.GetFocus()
				controlRequestInput := tview.NewInputField()
				controlRequestInput.SetLabel("Enter focus value: ").
					SetFieldWidth(10).
					SetAcceptanceFunc(tview.InputFieldInteger).
					SetDoneFunc(func(key tcell.Key) {
						manualFocus := &descriptors.FocusAutoControl{FocusAuto: false}
						err := ci.CameraTerminal.Set(manualFocus)
						if err != nil {
							log.Printf("manual focus request failed %s", err)
						}

						capture, err := strconv.ParseUint(controlRequestInput.GetText(), 10, 16)
						if err != nil {
							log.Printf("failed parsing value %s", err)
						}
						setExposure := &descriptors.FocusAbsoluteControl{Focus: uint16(capture)}
						err = ci.CameraTerminal.Set(setExposure)
						if err != nil {
							log.Printf("absolute focus request failed %s", err)
						}
						secondColumn.RemoveItem(controlRequestInput)
						app.SetFocus(initFocus)
					})
				secondColumn.AddItem(controlRequestInput, 1, 1, false)
				app.SetFocus(controlRequestInput)
			}}
	case *descriptors.FocusAutoControl:
		return &ControlRequestListItem{
			title: "Enable Automatic Focus",
			handler: func() {
				manualFocus := &descriptors.FocusAutoControl{FocusAuto: true}
				err := ci.CameraTerminal.Set(manualFocus)
				if err != nil {
					log.Printf("auto focus request failed %s", err)
				}
			}}
	case *descriptors.ExposureTimeAbsoluteControl:
		return &ControlRequestListItem{
			title: "Exposure Time (Absolute)",
			handler: func() {
				initFocus := app.GetFocus()
				controlRequestInput := tview.NewInputField()
				controlRequestInput.SetLabel("Enter exposure value: ").
					SetFieldWidth(10).
					SetAcceptanceFunc(tview.InputFieldInteger).
					SetDoneFunc(func(key tcell.Key) {
						capture, err := strconv.ParseUint(controlRequestInput.GetText(), 10, 16)
						if err != nil {
							log.Printf("failed parsing value %s", err)
						}

						manualExposure := &descriptors.AutoExposureModeControl{Mode: descriptors.AutoExposureModeManual}
						err = ci.CameraTerminal.Set(manualExposure)
						if err != nil {
							log.Printf("manual focus request failed %s", err)
						}

						setExposure := &descriptors.ExposureTimeAbsoluteControl{Time: uint32(capture)}
						err = ci.CameraTerminal.Set(setExposure)
						if err != nil {
							log.Printf("control request failed %s", err)
						}
						secondColumn.RemoveItem(controlRequestInput)
						app.SetFocus(initFocus)
					})
				secondColumn.AddItem(controlRequestInput, 1, 1, false)
				app.SetFocus(controlRequestInput)
			}}
	case *descriptors.ExposureTimeRelativeControl:
	case *descriptors.FocusRelativeControl:
	case *descriptors.FocusSimpleRangeControl:
	case *descriptors.RollAbsoluteControl:
	case *descriptors.IrisAbsoluteControl:
	case *descriptors.IrisRelativeControl:
	case *descriptors.PanTiltAbsoluteControl:
	case *descriptors.PanTiltRelativeControl:
	case *descriptors.RegionOfInterestControl:
	case *descriptors.RollRelativeControl:
	case *descriptors.ZoomAbsoluteControl:
		return &ControlRequestListItem{
			title: "Zoom (Absolute)",
			handler: func() {
				initFocus := app.GetFocus()
				controlRequestInput := tview.NewInputField()
				controlRequestInput.SetLabel("Enter zoom value (>= 100): ").
					SetFieldWidth(10).
					SetAcceptanceFunc(tview.InputFieldInteger).
					SetDoneFunc(func(key tcell.Key) {
						capture, err := strconv.ParseUint(controlRequestInput.GetText(), 10, 16)
						if err != nil {
							log.Printf("failed parsing value %s", err)
						}
						setControl := &descriptors.ZoomAbsoluteControl{ObjectiveFocalLength: uint16(capture)}
						err = ci.CameraTerminal.Set(setControl)
						if err != nil {
							log.Printf("control request failed %s", err)
						}
						secondColumn.RemoveItem(controlRequestInput)
						app.SetFocus(initFocus)
					})
				secondColumn.AddItem(controlRequestInput, 1, 1, false)
				app.SetFocus(controlRequestInput)
			}}
	case *descriptors.ZoomRelativeControl:
	}
	return nil
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
