# github.com/kevmo314/go-uvc

An almost-pure-Go library for accessing UVC devices. The library currently depends on libusb via cgo
but _not_ libuvc. One day this may change but libusb is much more complex.

The non-Go equivalent of this library is [libuvc](https://github.com/libuvc/libuvc).

## Usage

```go
package main

import (
	"fmt"
	"log"

	"github.com/kevmo314/go-uvc"
)

func main() {
	ctx := uvc.NewContext()
	defer ctx.Close()

	devs, err := ctx.Devices()
	if err != nil {
		log.Fatal(err)
	}

	for _, dev := range devs {
		fmt.Printf("%s\n", dev.Description())
	}
}
```
