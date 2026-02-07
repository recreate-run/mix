package vision

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOverlayBoundingBoxes(t *testing.T) {
	t.Helper()

	// Create minimal test PNG (500x500 white image)
	img := image.NewRGBA(image.Rect(0, 0, 500, 500))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	require.NoError(t, err)

	pngBytes := buf.Bytes()

	elements := []Element{
		{Index: 0, Role: "button", Name: "Submit", Bounds: BoundingBox{X: 100, Y: 200, Width: 80, Height: 40}},
		{Index: 1, Role: "link", Name: "Home", Bounds: BoundingBox{X: 200, Y: 300, Width: 60, Height: 30}},
	}

	// Overlay bounding boxes
	result, err := OverlayBoundingBoxes(pngBytes, elements)
	require.NoError(t, err)

	// Verify it's valid PNG
	resultImg, err := png.Decode(bytes.NewReader(result))
	require.NoError(t, err)

	// Verify dimensions unchanged
	assert.Equal(t, img.Bounds(), resultImg.Bounds())

	// Verify size increased (overlay adds data)
	assert.Greater(t, len(result), len(pngBytes))
}

func TestOverlayBoundingBoxes_EmptyElements(t *testing.T) {
	t.Helper()

	// Create test PNG
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	require.NoError(t, err)

	// Overlay with no elements should work
	result, err := OverlayBoundingBoxes(buf.Bytes(), []Element{})
	require.NoError(t, err)

	// Should still be valid PNG
	_, err = png.Decode(bytes.NewReader(result))
	require.NoError(t, err)
}

func TestOverlayBoundingBoxes_InvalidPNG(t *testing.T) {
	t.Helper()

	invalidPNG := []byte("not a valid PNG")

	_, err := OverlayBoundingBoxes(invalidPNG, []Element{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode PNG")
}

func TestOverlayBoundingBoxes_MultipleElements(t *testing.T) {
	t.Helper()

	// Create larger test image
	img := image.NewRGBA(image.Rect(0, 0, 1920, 1080))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	require.NoError(t, err)

	// Create multiple elements
	elements := []Element{
		{Index: 0, Role: "button", Bounds: BoundingBox{X: 10, Y: 10, Width: 100, Height: 50}},
		{Index: 1, Role: "link", Bounds: BoundingBox{X: 200, Y: 200, Width: 150, Height: 30}},
		{Index: 2, Role: "checkbox", Bounds: BoundingBox{X: 500, Y: 500, Width: 20, Height: 20}},
		{Index: 3, Role: "textbox", Bounds: BoundingBox{X: 1000, Y: 800, Width: 300, Height: 40}},
	}

	result, err := OverlayBoundingBoxes(buf.Bytes(), elements)
	require.NoError(t, err)

	// Verify it's valid PNG
	resultImg, err := png.Decode(bytes.NewReader(result))
	require.NoError(t, err)
	assert.Equal(t, img.Bounds(), resultImg.Bounds())
}

func TestOverlayBoundingBoxes_EdgeCases(t *testing.T) {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 500, 500))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	require.NoError(t, err)

	tests := []struct {
		name     string
		elements []Element
	}{
		{
			name: "element at origin",
			elements: []Element{
				{Index: 0, Role: "button", Bounds: BoundingBox{X: 0, Y: 0, Width: 50, Height: 50}},
			},
		},
		{
			name: "element at bottom-right corner",
			elements: []Element{
				{Index: 0, Role: "button", Bounds: BoundingBox{X: 450, Y: 450, Width: 50, Height: 50}},
			},
		},
		{
			name: "very small element",
			elements: []Element{
				{Index: 0, Role: "checkbox", Bounds: BoundingBox{X: 100, Y: 100, Width: 5, Height: 5}},
			},
		},
		{
			name: "large element spanning most of image",
			elements: []Element{
				{Index: 0, Role: "button", Bounds: BoundingBox{X: 10, Y: 10, Width: 480, Height: 480}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := OverlayBoundingBoxes(buf.Bytes(), tt.elements)
			require.NoError(t, err)

			// Verify it's valid PNG
			_, err = png.Decode(bytes.NewReader(result))
			require.NoError(t, err)
		})
	}
}

func TestDrawRect(t *testing.T) {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	bounds := BoundingBox{X: 50, Y: 50, Width: 100, Height: 80}
	testColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}

	drawRect(img, bounds, testColor, 3)

	// Verify corners have the color (part of the rectangle)
	assert.Equal(t, testColor, img.At(50, 50))
	assert.Equal(t, testColor, img.At(149, 50))
	assert.Equal(t, testColor, img.At(50, 129))
	assert.Equal(t, testColor, img.At(149, 129))

	// Verify center is still white (not filled)
	assert.Equal(t, color.RGBA{R: 255, G: 255, B: 255, A: 255}, img.At(100, 90))
}

func TestDrawText(t *testing.T) {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	testColor := color.RGBA{R: 0, G: 0, B: 0, A: 255}

	// Draw text should not panic
	drawText(img, "[0]", 10, 20, testColor)

	// Verify some pixels changed (text was drawn)
	changed := false
	for x := 10; x < 30; x++ {
		for y := 10; y < 30; y++ {
			if img.At(x, y) != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
				changed = true
				break
			}
		}
	}
	assert.True(t, changed, "Text should have changed some pixels")
}
