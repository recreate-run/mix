package browser

import (
	"encoding/json"
	"fmt"
)

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
	ActionKey         = "key"
	ActionScrollTo    = "scroll_to"
	ActionSequence    = "action"
)

// Sub-action constants for action sequences
const (
	SubActionLeftClick     = "left_click"
	SubActionRightClick    = "right_click"
	SubActionDoubleClick   = "double_click"
	SubActionTripleClick   = "triple_click"
	SubActionLeftClickDrag = "left_click_drag"
	SubActionType          = "type"
	SubActionKey           = "key"
	SubActionScroll        = "scroll"
	SubActionScrollTo      = "scroll_to"
	SubActionFormInput     = "form_input"
	SubActionWait          = "wait"
	SubActionScreenshot    = "screenshot"
)

// Scroll direction constants
const (
	DirectionUp    = "up"
	DirectionDown  = "down"
	DirectionLeft  = "left"
	DirectionRight = "right"
)

// Wait duration constraints
const (
	MaxWaitDuration = 150000 // Maximum wait duration in milliseconds
)

// Coordinate represents an x,y coordinate encoded as [x,y]
type Coordinate struct {
	X float64
	Y float64
}

// UnmarshalJSON decodes [x,y] into Coordinate
func (c *Coordinate) UnmarshalJSON(data []byte) error {
	var coords []float64
	if err := json.Unmarshal(data, &coords); err != nil {
		return fmt.Errorf("invalid coordinate: %w", err)
	}
	if len(coords) != 2 {
		return fmt.Errorf("invalid coordinate length: %d", len(coords))
	}
	c.X = coords[0]
	c.Y = coords[1]
	return nil
}

// SubAction represents a single action in an action sequence
type SubAction struct {
	Type         string      `json:"type"`                    // Action type: left_click, type, key, scroll_to, etc.
	Index        *int        `json:"index,omitempty"`         // Element index
	Ref          string      `json:"ref,omitempty"`           // Element reference (e.g. f0_ref_1)
	Coordinate   *Coordinate `json:"coordinate,omitempty"`    // Click coordinate
	FromIndex    *int        `json:"fromIndex,omitempty"`     // Drag start element index (index mode)
	ToIndex      *int        `json:"toIndex,omitempty"`       // Drag end element index (index mode)
	FromX        *float64    `json:"fromX,omitempty"`         // Drag start X coordinate (coordinate mode)
	FromY        *float64    `json:"fromY,omitempty"`         // Drag start Y coordinate (coordinate mode)
	ToX          *float64    `json:"toX,omitempty"`           // Drag end X coordinate (coordinate mode)
	ToY          *float64    `json:"toY,omitempty"`           // Drag end Y coordinate (coordinate mode)
	Text         string      `json:"text,omitempty"`          // For type action
	Value        interface{} `json:"value,omitempty"`         // For form_input (string, number, or boolean)
	Key          string      `json:"key,omitempty"`           // For key action
	Direction    string      `json:"direction,omitempty"`     // For scroll
	ScrollAmount int         `json:"scroll_amount,omitempty"` // For scroll
	Duration     int         `json:"duration,omitempty"`      // For wait or click-and-hold
	Repeat       int         `json:"repeat,omitempty"`        // For multiple clicks
	FilePath     string      `json:"file_path,omitempty"`     // For screenshot sub-action
}

// SubActionResult represents the result of a single sub-action
type SubActionResult struct {
	Index          int    `json:"index"`
	Type           string `json:"type"`
	Success        bool   `json:"success"`
	Error          string `json:"error,omitempty"`
	ScreenshotFile string `json:"screenshot_file,omitempty"` // For screenshot sub-actions
}

// BrowserParams represents the parameters for browser tool operations
type BrowserParams struct {
	Action          string      `json:"action"`                    // Required: open|screenshot|read_page|click|type|scroll|upload|get_text|find|close|right_click|double_click|triple_click|drag|form_input|go_back|go_forward|tab_create|tab_list|tab_switch|tab_close
	Description     string      `json:"description,omitempty"`     // For action batching
	URL             string      `json:"url,omitempty"`             // For open action
	InteractiveOnly *bool       `json:"interactiveOnly,omitempty"` // For read_page action (default: false)
	Index           int         `json:"index,omitempty"`           // For click/type/upload/right_click/double_click/triple_click/form_input actions
	Ref             string      `json:"ref,omitempty"`             // Element reference (e.g. f0_ref_1)
	Coordinate      *Coordinate `json:"coordinate,omitempty"`      // Click coordinate
	Text            string      `json:"text,omitempty"`            // For type action
	Direction       string      `json:"direction,omitempty"`       // For scroll action (up/down/left/right)
	ScrollAmount    int         `json:"scroll_amount,omitempty"`   // For scroll action (pixels)
	FilePath        string      `json:"filePath,omitempty"`        // For upload action (absolute or session-relative)
	Strategy        string      `json:"strategy,omitempty"`        // For get_text action (auto/article/main/body)
	Query           string      `json:"query,omitempty"`           // For find action (keyword query)
	Value           interface{} `json:"value,omitempty"`           // For form_input action (string, number, or boolean)
	TabID           string      `json:"tabId,omitempty"`           // Optional: specify tab for operations (defaults to active tab)
	TabIDAlias      string      `json:"tab_id,omitempty"`          // Reference-style tab field
	Duration        int         `json:"duration,omitempty"`        // For wait action (milliseconds) or drag duration
	FromIndex       *int        `json:"fromIndex,omitempty"`       // For drag action (index-based mode)
	ToIndex         *int        `json:"toIndex,omitempty"`         // For drag action (index-based mode)
	FromX           *float64    `json:"fromX,omitempty"`           // For drag action (coordinate-based mode)
	FromY           *float64    `json:"fromY,omitempty"`           // For drag action (coordinate-based mode)
	ToX             *float64    `json:"toX,omitempty"`             // For drag action (coordinate-based mode)
	ToY             *float64    `json:"toY,omitempty"`             // For drag action (coordinate-based mode)

	// NEW: For key action
	Key string `json:"key,omitempty"` // Keyboard key(s) to press

	// NEW: For action batching
	Actions []SubAction `json:"actions,omitempty"` // Array of actions to execute
}

// EffectiveTabID returns the resolved tab ID value
func (p BrowserParams) EffectiveTabID() string {
	if p.TabID != "" {
		return p.TabID
	}
	return p.TabIDAlias
}
