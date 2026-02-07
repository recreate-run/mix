package vision

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// OverlayBoundingBoxes draws numbered bounding boxes on a PNG screenshot
func OverlayBoundingBoxes(pngBytes []byte, elements []Element) ([]byte, error) {
	// 1. Decode PNG
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decode PNG: %w", err)
	}

	// 2. Convert to RGBA for drawing
	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{}, draw.Src)

	// 3. Draw each element
	for _, elem := range elements {
		// Draw rectangle
		drawRect(rgba, elem.Bounds, BoxColor, BoxThickness)

		// Draw label
		label := fmt.Sprintf("[%d]", elem.Index)
		drawText(rgba, label, int(elem.Bounds.X)+LabelOffsetX, int(elem.Bounds.Y)+LabelOffsetY, TextColor)
	}

	// 4. Re-encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, rgba); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	return buf.Bytes(), nil
}

// drawRect draws a rectangle with specified thickness
func drawRect(img *image.RGBA, bounds BoundingBox, c color.RGBA, thickness int) {
	x1, y1 := int(bounds.X), int(bounds.Y)
	x2, y2 := x1+int(bounds.Width), y1+int(bounds.Height)

	// Clamp to image bounds
	imgBounds := img.Bounds()
	x1 = max(x1, imgBounds.Min.X)
	y1 = max(y1, imgBounds.Min.Y)
	x2 = min(x2, imgBounds.Max.X)
	y2 = min(y2, imgBounds.Max.Y)

	// Draw horizontal lines (top and bottom)
	for t := 0; t < thickness; t++ {
		for x := x1; x < x2; x++ {
			if y1+t < imgBounds.Max.Y {
				img.Set(x, y1+t, c) // Top
			}
			if y2-t-1 >= imgBounds.Min.Y {
				img.Set(x, y2-t-1, c) // Bottom
			}
		}
	}

	// Draw vertical lines (left and right)
	for t := 0; t < thickness; t++ {
		for y := y1; y < y2; y++ {
			if x1+t < imgBounds.Max.X {
				img.Set(x1+t, y, c) // Left
			}
			if x2-t-1 >= imgBounds.Min.X {
				img.Set(x2-t-1, y, c) // Right
			}
		}
	}
}

// drawText draws text at the specified position
func drawText(img *image.RGBA, text string, x, y int, c color.RGBA) {
	point := fixed.Point26_6{
		X: fixed.Int26_6(x * 64),
		Y: fixed.Int26_6(y * 64),
	}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(text)
}
