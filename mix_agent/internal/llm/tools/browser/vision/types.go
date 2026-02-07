package vision

// RawAccessibilityNode represents a CDP accessibility tree node
type RawAccessibilityNode struct {
	Role      string       `json:"role"`
	Name      string       `json:"name,omitempty"`
	Bounds    BoundingBox  `json:"bounds"`
	BackendID int64        `json:"backendId,omitempty"`
}

// BoundingBox represents element position and size
type BoundingBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// ViewportBounds represents browser viewport dimensions
type ViewportBounds struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// Element represents a processed interactive element with index
type Element struct {
	Index     int         `json:"index"`
	Role      string      `json:"role"`
	Name      string      `json:"name"`
	Bounds    BoundingBox `json:"bounds"`
	BackendID int64       `json:"backendId,omitempty"`
}
