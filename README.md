# github.com/kevmo314/go-uvc

An almost-pure-Go library for accessing UVC devices. The library currently depends on libusb via cgo
but _not_ libuvc. One day this may change but libusb is much more complex.

The non-Go equivalent of this library is [libuvc](https://github.com/libuvc/libuvc).

![image](https://github.com/kevmo314/go-uvc/assets/511342/1e4d4a0b-37ad-44c0-b97d-9e2a3d4551d2)

## Features

- [x] UVC 1.5 support
  - [x] Input terminals (recording from cameras)
  - [ ] Output terminals (displaying images on a device)
- [x] Isochronous and bulk transfer support
- [x] Android support
- [x] Video decoding API
- [ ] Camera controls
- [ ] UAC Audio support

## Demo

`go-uvc` includes a debugging tool (the screenshot above) to connect to and debug cameras. To use it,

```sh
go run github.com/kevmo314/go-uvc/cmd/inspect -path /dev/bus/usb/001/002
```

## Usage

A minimal example of how you might use `go-uvc`.

```go
package main

import (
	"image/jpeg"
	"fmt"
	"log"
	"syscall"

	"github.com/kevmo314/go-uvc"
	"github.com/kevmo314/go-uvc/pkg/descriptors"
)

func main() {
	fd, err := syscall.Open("/dev/bus/usb/001/002", syscall.O_RDWR, 0)
	if err != nil {
		panic(err)
	}

	ctx, err := uvc.NewUVCDevice(uintptr(fd))
	if err != nil {
		panic(err)
	}

	go ctx.EventLoop()

	info, err := ctx.DeviceInfo()
	if err != nil {
		panic(err)
	}

	for _, iface := range info.StreamingInterfaces {
		for i, desc := range iface.Descriptors {
			fd, ok := desc.(*descriptors.MJPEGFormatDescriptor)
			if !ok {
				continue
			}
			frd := iface.Descriptors[i+1].(*descriptors.MJPEGFrameDescriptor)

			resp, err := iface.ClaimFrameReader(fd.Index(), frd.Index())
			if err != nil {
   				panic(err)
			}

			for i := 0; ; i++ {
				fr, err := resp.ReadFrame()
				if err != nil {
   					panic(err)
				}
				img, err := jpeg.Decode(fr)
				if err != nil {
					continue
				}
    				// do something with img
			}
		}
	}
}
```
