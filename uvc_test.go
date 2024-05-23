package uvc

import (
	"image/jpeg"
	"log"
	"os"
	"sync/atomic"
	"syscall"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kevmo314/go-uvc/pkg/decode"
	"github.com/kevmo314/go-uvc/pkg/descriptors"
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

func TestDeviceInfo(t *testing.T) {
	fd, err := syscall.Open("/dev/bus/usb/001/007", syscall.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}

	ctx, err := NewUVCDevice(uintptr(fd))
	if err != nil {
		t.Fatal(err)
	}

	go ctx.EventLoop()

	info, err := ctx.DeviceInfo()
	if err != nil {
		t.Fatal(err)
	}

	// get format descriptors
	for _, iface := range info.ControlInterfaces {
		log.Printf("got control interface: %#v", iface)
	}

	for _, iface := range info.StreamingInterfaces {
		for i, desc := range iface.Descriptors {
			fd, ok := desc.(*descriptors.MJPEGFormatDescriptor)
			if !ok {
				continue
			}
			fr := iface.Descriptors[i+1].(*descriptors.MJPEGFrameDescriptor)

			resp, err := iface.ClaimFrameReader(fd.Index(), fr.Index())
			if err != nil {
				t.Fatal(err)
			}
			decoder, err := decode.NewFrameReaderDecoder(resp, fd, fr)
			if err != nil {
				t.Fatal(err)
			}

			g := &Display{}
			for i := 0; ; i++ {
				img, err := decoder.ReadFrame()
				if err != nil {
					log.Printf("got error: %s", err)
					continue
				}
				// write fr to a file
				log.Printf("got frame: %#v", img.Bounds())
				if g.frame.Swap(ebiten.NewImageFromImage(img)) == nil {
					go func() {
						if err := ebiten.RunGame(g); err != nil {
							panic(err)
						}
					}()
				}
			}
		}
	}
	log.Printf("done")
	t.Fail()
}

func TestJPEGDecode(t *testing.T) {
	// open frame-1.jpg and decode it
	f, err := os.Open("frame-0.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	img, err := jpeg.Decode(f)
	if err != nil {
		t.Fatal(err)
	}

	log.Printf("got frame: %#v", img.Bounds())
}
