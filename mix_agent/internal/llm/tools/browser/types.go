package browser

// Action constants for browser operations
const (
	ActionOpen        = "open"
	ActionScreenshot  = "screenshot"
	ActionReadPage    = "read_page"
	ActionClick       = "click"
	ActionType        = "type"
	ActionScroll      = "scroll"
	ActionUpload      = "upload"
	ActionGetText     = "get_text"
	ActionFind        = "find"
	ActionClose       = "close"
	ActionRightClick  = "right_click"
	ActionDoubleClick = "double_click"
	ActionTripleClick = "triple_click"
	ActionDrag        = "drag"
	ActionFormInput   = "form_input"
	ActionGoBack      = "go_back"
	ActionGoForward   = "go_forward"
	ActionTabCreate   = "tab_create"
	ActionTabList     = "tab_list"
	ActionTabSwitch   = "tab_switch"
	ActionTabClose    = "tab_close"
	ActionWait        = "wait"
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
	Action          string `json:"action"`                    // Required: open|screenshot|read_page|click|type|scroll|upload|get_text|find|close|right_click|double_click|triple_click|drag|form_input|go_back|go_forward|tab_create|tab_list|tab_switch|tab_close
	URL             string `json:"url,omitempty"`             // For open action
	WithOverlay     *bool  `json:"withOverlay,omitempty"`     // For screenshot (default: true)
	InteractiveOnly *bool  `json:"interactiveOnly,omitempty"` // For read_page action (default: false)
	Index       int      `json:"index,omitempty"`       // For click/type/upload/right_click/double_click/triple_click/form_input actions
	Text        string   `json:"text,omitempty"`        // For type action
	Direction   string   `json:"direction,omitempty"`   // For scroll action (up/down/left/right)
	Amount      int      `json:"amount,omitempty"`      // For scroll action (pixels)
	FilePath    string   `json:"filePath,omitempty"`    // For upload action (absolute or session-relative)
	Strategy    string   `json:"strategy,omitempty"`    // For get_text action (auto/article/main/body)
	Query       string   `json:"query,omitempty"`       // For find action (keyword query)
	Value       string   `json:"value,omitempty"`       // For form_input action
	TabID       string   `json:"tabId,omitempty"`       // Optional: specify tab for operations (defaults to active tab)
	Duration    int      `json:"duration,omitempty"`    // For wait action (milliseconds) or drag duration
	FromIndex   *int     `json:"fromIndex,omitempty"`   // For drag action (index-based mode)
	ToIndex     *int     `json:"toIndex,omitempty"`     // For drag action (index-based mode)
	FromX       *float64 `json:"fromX,omitempty"`       // For drag action (coordinate-based mode)
	FromY       *float64 `json:"fromY,omitempty"`       // For drag action (coordinate-based mode)
	ToX         *float64 `json:"toX,omitempty"`         // For drag action (coordinate-based mode)
	ToY         *float64 `json:"toY,omitempty"`         // For drag action (coordinate-based mode)
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
