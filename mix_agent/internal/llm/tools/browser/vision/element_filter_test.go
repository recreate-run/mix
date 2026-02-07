package vision

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterInteractiveElements(t *testing.T) {
	t.Helper()

	viewport := ViewportBounds{X: 0, Y: 0, Width: 1920, Height: 1080}

	tests := []struct {
		name          string
		nodes         []RawAccessibilityNode
		expectedCount int
		expectedRoles []string
	}{
		{
			name: "filters interactive elements only",
			nodes: []RawAccessibilityNode{
				{Role: "button", Name: "Submit", Bounds: BoundingBox{X: 100, Y: 200, Width: 80, Height: 40}},
				{Role: "StaticText", Name: "Label", Bounds: BoundingBox{X: 50, Y: 50, Width: 100, Height: 20}},
				{Role: "link", Name: "Home", Bounds: BoundingBox{X: 200, Y: 300, Width: 60, Height: 30}},
			},
			expectedCount: 2,
			expectedRoles: []string{"button", "link"},
		},
		{
			name: "filters zero-size elements",
			nodes: []RawAccessibilityNode{
				{Role: "button", Name: "Valid", Bounds: BoundingBox{X: 100, Y: 200, Width: 80, Height: 40}},
				{Role: "button", Name: "ZeroWidth", Bounds: BoundingBox{X: 100, Y: 200, Width: 0, Height: 40}},
				{Role: "button", Name: "ZeroHeight", Bounds: BoundingBox{X: 100, Y: 200, Width: 80, Height: 0}},
			},
			expectedCount: 1,
			expectedRoles: []string{"button"},
		},
		{
			name: "filters out-of-viewport elements",
			nodes: []RawAccessibilityNode{
				{Role: "button", Name: "Visible", Bounds: BoundingBox{X: 100, Y: 200, Width: 80, Height: 40}},
				{Role: "button", Name: "Below", Bounds: BoundingBox{X: 100, Y: 2000, Width: 80, Height: 40}},
				{Role: "button", Name: "Right", Bounds: BoundingBox{X: 3000, Y: 200, Width: 80, Height: 40}},
			},
			expectedCount: 1,
			expectedRoles: []string{"button"},
		},
		{
			name: "sequential indexing",
			nodes: []RawAccessibilityNode{
				{Role: "button", Name: "First", Bounds: BoundingBox{X: 100, Y: 100, Width: 80, Height: 40}},
				{Role: "StaticText", Name: "Skip", Bounds: BoundingBox{X: 200, Y: 100, Width: 80, Height: 40}},
				{Role: "link", Name: "Second", Bounds: BoundingBox{X: 300, Y: 100, Width: 80, Height: 40}},
				{Role: "checkbox", Name: "Third", Bounds: BoundingBox{X: 400, Y: 100, Width: 20, Height: 20}},
			},
			expectedCount: 3,
			expectedRoles: []string{"button", "link", "checkbox"},
		},
		{
			name:          "empty nodes",
			nodes:         []RawAccessibilityNode{},
			expectedCount: 0,
			expectedRoles: []string{},
		},
		{
			name: "all interactive roles",
			nodes: []RawAccessibilityNode{
				{Role: "button", Bounds: BoundingBox{X: 10, Y: 10, Width: 80, Height: 40}},
				{Role: "link", Bounds: BoundingBox{X: 10, Y: 60, Width: 80, Height: 40}},
				{Role: "textbox", Bounds: BoundingBox{X: 10, Y: 110, Width: 80, Height: 40}},
				{Role: "searchbox", Bounds: BoundingBox{X: 10, Y: 160, Width: 80, Height: 40}},
				{Role: "combobox", Bounds: BoundingBox{X: 10, Y: 210, Width: 80, Height: 40}},
				{Role: "listbox", Bounds: BoundingBox{X: 10, Y: 260, Width: 80, Height: 40}},
				{Role: "menu", Bounds: BoundingBox{X: 10, Y: 310, Width: 80, Height: 40}},
				{Role: "menuitem", Bounds: BoundingBox{X: 10, Y: 360, Width: 80, Height: 40}},
				{Role: "menuitemcheckbox", Bounds: BoundingBox{X: 10, Y: 410, Width: 80, Height: 40}},
				{Role: "menuitemradio", Bounds: BoundingBox{X: 10, Y: 460, Width: 80, Height: 40}},
				{Role: "tab", Bounds: BoundingBox{X: 10, Y: 510, Width: 80, Height: 40}},
				{Role: "checkbox", Bounds: BoundingBox{X: 10, Y: 560, Width: 20, Height: 20}},
				{Role: "radio", Bounds: BoundingBox{X: 10, Y: 590, Width: 20, Height: 20}},
				{Role: "slider", Bounds: BoundingBox{X: 10, Y: 620, Width: 80, Height: 40}},
				{Role: "spinbutton", Bounds: BoundingBox{X: 10, Y: 670, Width: 80, Height: 40}},
				{Role: "switch", Bounds: BoundingBox{X: 10, Y: 720, Width: 50, Height: 30}},
			},
			expectedCount: 16,
			expectedRoles: []string{"button", "link", "textbox", "searchbox", "combobox", "listbox", "menu", "menuitem", "menuitemcheckbox", "menuitemradio", "tab", "checkbox", "radio", "slider", "spinbutton", "switch"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			elements := FilterInteractiveElements(tt.nodes, viewport)

			assert.Len(t, elements, tt.expectedCount, "Element count mismatch")

			for i, elem := range elements {
				assert.Equal(t, i, elem.Index, "Index should be sequential starting from 0")
				if i < len(tt.expectedRoles) {
					assert.Equal(t, tt.expectedRoles[i], elem.Role, "Role mismatch at index %d", i)
				}
			}
		})
	}
}

func TestIsInteractiveRole(t *testing.T) {
	t.Helper()

	tests := []struct {
		role     string
		expected bool
	}{
		{"button", true},
		{"link", true},
		{"textbox", true},
		{"checkbox", true},
		{"StaticText", false},
		{"heading", false},
		{"image", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			result := isInteractiveRole(tt.role)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsInViewport(t *testing.T) {
	t.Helper()

	viewport := BoundingBox{X: 0, Y: 0, Width: 1920, Height: 1080}

	tests := []struct {
		name     string
		elem     BoundingBox
		expected bool
	}{
		{
			name:     "fully inside viewport",
			elem:     BoundingBox{X: 100, Y: 200, Width: 80, Height: 40},
			expected: true,
		},
		{
			name:     "partially visible (right edge cut off)",
			elem:     BoundingBox{X: 1900, Y: 200, Width: 80, Height: 40},
			expected: true,
		},
		{
			name:     "partially visible (bottom edge cut off)",
			elem:     BoundingBox{X: 100, Y: 1060, Width: 80, Height: 40},
			expected: true,
		},
		{
			name:     "completely below viewport",
			elem:     BoundingBox{X: 100, Y: 2000, Width: 80, Height: 40},
			expected: false,
		},
		{
			name:     "completely to the right of viewport",
			elem:     BoundingBox{X: 3000, Y: 200, Width: 80, Height: 40},
			expected: false,
		},
		{
			name:     "completely above viewport",
			elem:     BoundingBox{X: 100, Y: -200, Width: 80, Height: 40},
			expected: false,
		},
		{
			name:     "completely to the left of viewport",
			elem:     BoundingBox{X: -200, Y: 200, Width: 80, Height: 40},
			expected: false,
		},
		{
			name:     "at viewport origin",
			elem:     BoundingBox{X: 0, Y: 0, Width: 80, Height: 40},
			expected: true,
		},
		{
			name:     "at viewport bottom-right corner",
			elem:     BoundingBox{X: 1840, Y: 1040, Width: 80, Height: 40},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isInViewport(tt.elem, viewport)
			assert.Equal(t, tt.expected, result)
		})
	}
}
