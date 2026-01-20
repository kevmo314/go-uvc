# github.com/kevmo314/go-uvc

A pure-Go library for accessing UVC devices.

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
	- [x] UAC1 support
 	- [x] Basic audio controls
 	- [ ] UAC2/3 support

## Demo

`go-uvc` includes a debugging tool (the screenshot above) to connect to and debug cameras. To use it,

```sh
go run github.com/kevmo314/go-uvc/cmd/inspect -path /dev/bus/usb/001/002
```

## Docker Support

This project includes Docker support with USB device passthrough capabilities for accessing UVC devices from within containers.

### Building the Docker Image

```bash
docker build -t go-uvc .
```

### Running with USB Passthrough

To run the container with USB device access, you need to use privileged mode and mount the USB bus:

```bash
# Run with full USB bus access (recommended for development)
docker run -it --rm \
  --privileged \
  --device=/dev/bus/usb:/dev/bus/usb \
  -v /dev/bus/usb:/dev/bus/usb \
  go-uvc \
  /app/inspect -path /dev/bus/usb/001/002

# Run with specific USB device access
docker run -it --rm \
  --device=/dev/bus/usb/001/002 \
  go-uvc \
  /app/inspect -path /dev/bus/usb/001/002
```

### Finding Your USB Device

To find your USB camera device path:

```bash
# List all USB devices
lsusb

# Find the bus and device number (e.g., Bus 001 Device 002)
# The path will be /dev/bus/usb/{bus}/{device}
```

### Development with DevContainers

This project includes a DevContainer configuration for VS Code with USB passthrough support.

#### Prerequisites
- Docker Desktop or Docker Engine
- VS Code with the Remote-Containers extension

#### Using the DevContainer

1. Open the project in VS Code
2. When prompted, click "Reopen in Container" or use the command palette (F1) and select "Remote-Containers: Reopen in Container"
3. The container will build with all necessary dependencies and USB access configured
4. USB devices will be automatically available at `/dev/bus/usb/`

#### Manual DevContainer Build

```bash
# Build the development container
docker build -f .devcontainer/Dockerfile -t go-uvc-dev .

# Run the development container with USB access
docker run -it --rm \
  --privileged \
  --device=/dev/bus/usb:/dev/bus/usb \
  -v /dev/bus/usb:/dev/bus/usb \
  -v $(pwd):/workspace \
  -w /workspace \
  go-uvc-dev
```

### Troubleshooting USB Access

1. **Permission Denied**: Make sure to run the container with `--privileged` flag
2. **Device Not Found**: Verify the device path with `lsusb` on the host
3. **Device Busy**: Ensure no other application is using the USB device
4. **SELinux/AppArmor**: On systems with SELinux or AppArmor, you may need additional configuration

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
