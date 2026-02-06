package browser

// Action constants for browser operations
const (
	ActionOpen       = "open"
	ActionScreenshot = "screenshot"
	ActionClick      = "click"
	ActionType       = "type"
	ActionScroll     = "scroll"
	ActionClose      = "close"
)

// Scroll direction constants
const (
	DirectionUp    = "up"
	DirectionDown  = "down"
	DirectionLeft  = "left"
	DirectionRight = "right"
)

// BrowserParams represents the parameters for browser tool operations
type BrowserParams struct {
	Action      string `json:"action"`                // Required: open|screenshot|click|type|scroll|close
	URL         string `json:"url,omitempty"`         // For open action
	WithOverlay *bool  `json:"withOverlay,omitempty"` // For screenshot (default: true)
	Index       int    `json:"index,omitempty"`       // For click/type actions
	Text        string `json:"text,omitempty"`        // For type action
	Direction   string `json:"direction,omitempty"`   // For scroll action (up/down/left/right)
	Amount      int    `json:"amount,omitempty"`      // For scroll action (pixels)
}

// ScreenshotMetadata contains metadata about a captured screenshot
type ScreenshotMetadata struct {
	URL          string        `json:"url"`
	Timestamp    int64         `json:"timestamp"`
	WithOverlay  bool          `json:"with_overlay"`
	ElementCount int           `json:"element_count"`
	Elements     []ElementInfo `json:"elements,omitempty"`
}

// ElementInfo contains information about an interactive element (for metadata only)
type ElementInfo struct {
	Index  int     `json:"index"`
	Role   string  `json:"role"`
	Name   string  `json:"name"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}
