package browser

import (
	"testing"

	browserprotocol "github.com/sarathmenon/browser-service/pkg/protocol"
	"github.com/stretchr/testify/assert"
)

func TestFormatAttributes(t *testing.T) {
	t.Helper()

	tests := []struct {
		name     string
		attrs    map[string]string
		expected string
	}{
		{
			name:     "empty attributes",
			attrs:    map[string]string{},
			expected: "",
		},
		{
			name: "priority ordering",
			attrs: map[string]string{
				"type":        "text",
				"href":        "/link",
				"id":          "test",
				"placeholder": "Enter text",
			},
			expected: ` href="/link" id="test" type="text" placeholder="Enter text"`,
		},
		{
			name: "single attribute",
			attrs: map[string]string{
				"id": "test-element",
			},
			expected: ` id="test-element"`,
		},
		{
			name: "non-priority attributes sorted alphabetically",
			attrs: map[string]string{
				"data-test": "value",
				"class":     "btn",
				"id":        "submit",
			},
			expected: ` id="submit" class="btn" data-test="value"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAttributes(tt.attrs)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildFrameNumberMap(t *testing.T) {
	t.Helper()

	elements := []browserprotocol.RawAccessibilityNode{
		{FrameID: "frame-123", BackendID: 1},
		{FrameID: "frame-123", BackendID: 2},
		{FrameID: "frame-456", BackendID: 3},
		{FrameID: "frame-123", BackendID: 4},
	}

	frameMap := buildFrameNumberMap(elements)

	// First unique frame gets 0, second gets 1
	assert.Equal(t, 0, frameMap["frame-123"])
	assert.Equal(t, 1, frameMap["frame-456"])
	assert.Len(t, frameMap, 2)
}

func TestBuildFrameNumberMapEmptyFrameIDs(t *testing.T) {
	t.Helper()

	elements := []browserprotocol.RawAccessibilityNode{
		{FrameID: "", BackendID: 1},
		{FrameID: "", BackendID: 2},
	}

	frameMap := buildFrameNumberMap(elements)
	assert.Empty(t, frameMap)
}

func TestFormatReadPageResponse(t *testing.T) {
	t.Helper()

	elements := []browserprotocol.RawAccessibilityNode{
		{
			Role:      "link",
			Name:      "Test Link",
			BackendID: 1005,
			FrameID:   "frame-123",
			Bounds:    browserprotocol.BoundingBox{X: 50, Y: 100, Width: 120, Height: 30},
			Attributes: map[string]string{
				"href": "https://example.com",
				"id":   "test-link",
			},
		},
		{
			Role:      "button",
			Name:      "",
			BackendID: 1010,
			FrameID:   "frame-123",
			Bounds:    browserprotocol.BoundingBox{X: 200, Y: 150, Width: 80, Height: 30},
		},
	}

	viewport := browserprotocol.BoundingBox{
		X: 0, Y: 0, Width: 1280, Height: 720,
	}

	result := formatReadPageResponse(elements, viewport, true)

	// Verify format
	assert.Contains(t, result, `- link "Test Link" [ref=f0_ref_1005] (x=50,y=100)`)
	assert.Contains(t, result, `href="https://example.com"`)
	assert.Contains(t, result, `id="test-link"`)
	assert.Contains(t, result, `- button [ref=f0_ref_1010] (x=200,y=150)`)
	assert.NotContains(t, result, `""`) // No empty name quotes

	// Verify old format is gone
	assert.NotContains(t, result, "[0]")
	assert.NotContains(t, result, "Position:")
	assert.NotContains(t, result, "Size:")
}

func TestFormatReadPageResponseMultipleFrames(t *testing.T) {
	t.Helper()

	elements := []browserprotocol.RawAccessibilityNode{
		{
			Role:      "link",
			Name:      "Main Frame Link",
			BackendID: 1005,
			FrameID:   "main-frame",
			Bounds:    browserprotocol.BoundingBox{X: 50, Y: 100, Width: 120, Height: 30},
		},
		{
			Role:      "button",
			Name:      "Iframe Button",
			BackendID: 2010,
			FrameID:   "iframe-1",
			Bounds:    browserprotocol.BoundingBox{X: 200, Y: 150, Width: 80, Height: 30},
		},
		{
			Role:      "textbox",
			Name:      "Main Frame Input",
			BackendID: 1020,
			FrameID:   "main-frame",
			Bounds:    browserprotocol.BoundingBox{X: 300, Y: 200, Width: 150, Height: 30},
		},
	}

	viewport := browserprotocol.BoundingBox{
		X: 0, Y: 0, Width: 1280, Height: 720,
	}

	result := formatReadPageResponse(elements, viewport, false)

	// Verify frame numbering is consistent
	assert.Contains(t, result, "[ref=f0_ref_1005]") // main-frame gets f0
	assert.Contains(t, result, "[ref=f1_ref_2010]") // iframe-1 gets f1
	assert.Contains(t, result, "[ref=f0_ref_1020]") // main-frame still f0
}
