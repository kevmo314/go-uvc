package decode

import (
	"image"
	"image/color"
)

// RGB is an in-memory image whose At method returns [color.RGB] values.
type RGB struct {
	// Pix holds the image's pixels, in R, G, B, A order. The pixel at
	// (x, y) starts at Pix[(y-Rect.Min.Y)*Stride + (x-Rect.Min.X)*3].
	Pix []uint8
	// Stride is the Pix stride (in bytes) between vertically adjacent pixels.
	Stride int
	// Rect is the image's bounds.
	Rect image.Rectangle
}

var _ image.Image = &RGB{}

func (p *RGB) ColorModel() color.Model { return color.RGBAModel }

func (p *RGB) Bounds() image.Rectangle { return p.Rect }

func (p *RGB) At(x, y int) color.Color {
	return p.RGBAAt(x, y)
}

func (p *RGB) RGBAAt(x, y int) color.RGBA {
	if !(image.Point{x, y}.In(p.Rect)) {
		return color.RGBA{}
	}
	i := p.PixOffset(x, y)
	s := p.Pix[i : i+3 : i+3] // Small cap improves performance, see https://golang.org/issue/27857
	return color.RGBA{s[0], s[1], s[2], 0}
}

// PixOffset returns the index of the first element of Pix that corresponds to
// the pixel at (x, y).
func (p *RGB) PixOffset(x, y int) int {
	return (y-p.Rect.Min.Y)*p.Stride + (x-p.Rect.Min.X)*3
}

// SubImage returns an image representing the portion of the image p visible
// through r. The returned value shares pixels with the original image.
func (p *RGB) SubImage(r image.Rectangle) image.Image {
	r = r.Intersect(p.Rect)
	// If r1 and r2 are Rectangles, r1.Intersect(r2) is not guaranteed to be inside
	// either r1 or r2 if the intersection is empty. Without explicitly checking for
	// this, the Pix[i:] expression below can panic.
	if r.Empty() {
		return &RGB{}
	}
	i := p.PixOffset(r.Min.X, r.Min.Y)
	return &RGB{
		Pix:    p.Pix[i:],
		Stride: p.Stride,
		Rect:   r,
	}
}

// BGR is an in-memory image whose At method returns [color.BGR] values.
type BGR struct {
	// Pix holds the image's pixels, in R, G, B, A order. The pixel at
	// (x, y) starts at Pix[(y-Rect.Min.Y)*Stride + (x-Rect.Min.X)*3].
	Pix []uint8
	// Stride is the Pix stride (in bytes) between vertically adjacent pixels.
	Stride int
	// Rect is the image's bounds.
	Rect image.Rectangle
}

var _ image.Image = &BGR{}

func (p *BGR) ColorModel() color.Model { return color.RGBAModel }

func (p *BGR) Bounds() image.Rectangle { return p.Rect }

func (p *BGR) At(x, y int) color.Color {
	return p.BGRAAt(x, y)
}

func (p *BGR) BGRAAt(x, y int) color.RGBA {
	if !(image.Point{x, y}.In(p.Rect)) {
		return color.RGBA{}
	}
	i := p.PixOffset(x, y)
	s := p.Pix[i : i+3 : i+3] // Small cap improves performance, see https://golang.org/issue/27857
	return color.RGBA{s[2], s[1], s[0], 0}
}

// PixOffset returns the index of the first element of Pix that corresponds to
// the pixel at (x, y).
func (p *BGR) PixOffset(x, y int) int {
	return (y-p.Rect.Min.Y)*p.Stride + (x-p.Rect.Min.X)*3
}

// SubImage returns an image representing the portion of the image p visible
// through r. The returned value shares pixels with the original image.
func (p *BGR) SubImage(r image.Rectangle) image.Image {
	r = r.Intersect(p.Rect)
	// If r1 and r2 are Rectangles, r1.Intersect(r2) is not guaranteed to be inside
	// either r1 or r2 if the intersection is empty. Without explicitly checking for
	// this, the Pix[i:] expression below can panic.
	if r.Empty() {
		return &BGR{}
	}
	i := p.PixOffset(r.Min.X, r.Min.Y)
	return &BGR{
		Pix:    p.Pix[i:],
		Stride: p.Stride,
		Rect:   r,
	}
}
