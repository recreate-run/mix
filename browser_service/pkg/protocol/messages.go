package protocol

import "encoding/json"

// Request represents an incoming CDP-like command
type Request struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Response represents a command result
type Response struct {
	ID     string      `json:"id"`
	Result interface{} `json:"result,omitempty"`
	Error  *Error      `json:"error,omitempty"`
}

// Error represents an error response
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Event represents a server-initiated event
type Event struct {
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
}

// --- Parameter Types ---

// --- Tab Management Types ---

// TabInfo represents information about a browser tab
type TabInfo struct {
	ID       string `json:"id"`       // Tab ID (e.g., "tab-1")
	URL      string `json:"url"`      // Current URL
	Title    string `json:"title"`    // Page title
	IsActive bool   `json:"isActive"` // Whether this is the active tab
}

// TabCreateParams for Tab.create (optional URL parameter)
type TabCreateParams struct {
	URL *string `json:"url,omitempty"` // Optional URL to navigate to after creating tab
}

// TabCreateResult for Tab.create response
type TabCreateResult struct {
	Tab TabInfo `json:"tab"`
}

// TabListResult for Tab.list response
type TabListResult struct {
	Tabs        []TabInfo `json:"tabs"`
	ActiveTabID string    `json:"activeTabId"`
}

// TabSwitchParams for Tab.switch
type TabSwitchParams struct {
	TabID string `json:"tabId"`
}

// TabCloseParams for Tab.close
type TabCloseParams struct {
	TabID string `json:"tabId"`
}

// NavigateParams for Page.navigate
type NavigateParams struct {
	URL     string  `json:"url"`
	Timeout int     `json:"timeout,omitempty"` // milliseconds
	TabID   *string `json:"tabId,omitempty"`   // Optional tab ID (defaults to active tab)
}

// NavigateResult for Page.navigate response
type NavigateResult struct {
	FrameID  string `json:"frameId"`
	LoaderID string `json:"loaderId,omitempty"`
}

// BoundingBox represents element coordinates
type BoundingBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// RawAccessibilityNode represents an unprocessed CDP accessibility tree node
type RawAccessibilityNode struct {
	Role       string            `json:"role"`                 // Accessibility role (button, link, textbox, etc.)
	Name       string            `json:"name,omitempty"`       // Accessible name
	Bounds     BoundingBox       `json:"bounds"`               // Element bounding box
	BackendID  int64             `json:"backendId,omitempty"`  // CDP backend node ID
	FrameID    string            `json:"frameId,omitempty"`    // CDP frame identifier
	Attributes map[string]string `json:"attributes,omitempty"` // HTML attributes (href, id, type, etc.)
}

// ViewportBounds represents browser viewport dimensions
type ViewportBounds BoundingBox

// ScreenshotParams for Page.screenshot
type ScreenshotParams struct {
	Format   string  `json:"format,omitempty"`   // png, jpeg
	Quality  int     `json:"quality,omitempty"`  // jpeg quality
	FullPage bool    `json:"fullPage,omitempty"` // capture full page
	Raw      bool    `json:"raw,omitempty"`      // Return raw accessibility tree nodes and viewport
	TabID    *string `json:"tabId,omitempty"`    // Optional tab ID (defaults to active tab)
}

// ScreenshotResult for Page.screenshot response
type ScreenshotResult struct {
	Data        string                 `json:"data"`                  // base64 encoded image
	Format      string                 `json:"format"`                // png, jpeg
	RawNodes    []RawAccessibilityNode `json:"rawNodes,omitempty"`    // Raw accessibility tree nodes (if Raw=true)
	RawViewport *ViewportBounds        `json:"rawViewport,omitempty"` // Viewport bounds (if Raw=true)
}

// Element represents an interactive element on the page
type Element struct {
	Index  int         `json:"index"`
	Role   string      `json:"role"` // button, link, textbox, etc.
	Name   string      `json:"name"` // accessible name
	Bounds BoundingBox `json:"bounds"`
}

// ClickParams for Page.click
type ClickParams struct {
	Index int     `json:"index"`             // element index from getElements
	TabID *string `json:"tabId,omitempty"`   // Optional tab ID (defaults to active tab)
}

// RightClickParams for Page.rightClick
type RightClickParams struct {
	Index int     `json:"index"`             // element index from getElements
	TabID *string `json:"tabId,omitempty"`   // Optional tab ID (defaults to active tab)
}

// DoubleClickParams for Page.doubleClick
type DoubleClickParams struct {
	Index int     `json:"index"`             // element index from getElements
	TabID *string `json:"tabId,omitempty"`   // Optional tab ID (defaults to active tab)
}

// TripleClickParams for Page.tripleClick
type TripleClickParams struct {
	Index int     `json:"index"`             // element index from getElements
	TabID *string `json:"tabId,omitempty"`   // Optional tab ID (defaults to active tab)
}

// ClickByBackendIDParams for Page.clickByBackendID
type ClickByBackendIDParams struct {
	BackendID int64   `json:"backendId"`       // CDP backend node ID
	TabID     *string `json:"tabId,omitempty"` // Optional tab ID (defaults to active tab)
}

// RightClickByBackendIDParams for Page.rightClickByBackendID
type RightClickByBackendIDParams struct {
	BackendID int64   `json:"backendId"`       // CDP backend node ID
	TabID     *string `json:"tabId,omitempty"` // Optional tab ID (defaults to active tab)
}

// DoubleClickByBackendIDParams for Page.doubleClickByBackendID
type DoubleClickByBackendIDParams struct {
	BackendID int64   `json:"backendId"`       // CDP backend node ID
	TabID     *string `json:"tabId,omitempty"` // Optional tab ID (defaults to active tab)
}

// TripleClickByBackendIDParams for Page.tripleClickByBackendID
type TripleClickByBackendIDParams struct {
	BackendID int64   `json:"backendId"`       // CDP backend node ID
	TabID     *string `json:"tabId,omitempty"` // Optional tab ID (defaults to active tab)
}

// DragParams for Page.drag
type DragParams struct {
	FromIndex *int     `json:"fromIndex,omitempty"` // element index to drag from
	ToIndex   *int     `json:"toIndex,omitempty"`   // element index to drag to
	FromX     *float64 `json:"fromX,omitempty"`     // X coordinate to drag from
	FromY     *float64 `json:"fromY,omitempty"`     // Y coordinate to drag from
	ToX       *float64 `json:"toX,omitempty"`       // X coordinate to drag to
	ToY       *float64 `json:"toY,omitempty"`       // Y coordinate to drag to
	Duration  *int     `json:"duration,omitempty"`  // Optional duration in ms (default: 500)
	TabID     *string  `json:"tabId,omitempty"`     // Optional tab ID (defaults to active tab)
}

// FormInputParams for Page.formInput
type FormInputParams struct {
	Index int     `json:"index"`             // element index from getElements
	Value string  `json:"value"`             // value to set
	TabID *string `json:"tabId,omitempty"`   // Optional tab ID (defaults to active tab)
}

// GoBackParams for Page.goBack
type GoBackParams struct {
	TabID *string `json:"tabId,omitempty"` // Optional tab ID (defaults to active tab)
}

// GoBackResult for Page.goBack response
type GoBackResult struct {
	URL string `json:"url"` // new URL after navigation
}

// GoForwardParams for Page.goForward
type GoForwardParams struct {
	TabID *string `json:"tabId,omitempty"` // Optional tab ID (defaults to active tab)
}

// GoForwardResult for Page.goForward response
type GoForwardResult struct {
	URL string `json:"url"` // new URL after navigation
}

// TypeParams for Page.type
type TypeParams struct {
	Index int     `json:"index"`             // element index
	Text  string  `json:"text"`              // text to type
	TabID *string `json:"tabId,omitempty"`   // Optional tab ID (defaults to active tab)
}

// ScrollParams for Page.scroll
type ScrollParams struct {
	Direction string  `json:"direction"`         // up, down, left, right
	Amount    int     `json:"amount"`            // pixels
	TabID     *string `json:"tabId,omitempty"`   // Optional tab ID (defaults to active tab)
}

// GetElementsResult for Page.getElements response
type GetElementsResult struct {
	Elements []RawAccessibilityNode `json:"elements"`
}

// ReadPageParams for Page.readPage request
type ReadPageParams struct {
	InteractiveOnly bool    `json:"interactiveOnly,omitempty"` // Filter to interactive elements only
	TabID           *string `json:"tabId,omitempty"`           // Optional tab ID
}

// ReadPageResult for Page.readPage response
type ReadPageResult struct {
	Elements []RawAccessibilityNode `json:"elements"` // Visible elements in viewport
	Viewport BoundingBox            `json:"viewport"` // Current viewport bounds
}

// ImportCookiesParams for Browser.importCookies
type ImportCookiesParams struct {
	Browser string `json:"browser"`           // chrome, arc, brave
	Profile string `json:"profile,omitempty"` // profile name, default "Default"
}

// ImportCookiesResult for Browser.importCookies response
type ImportCookiesResult struct {
	Imported int      `json:"imported"` // number of cookies imported
	Domains  []string `json:"domains"`  // domains with cookies
}

// SetUserAgentParams for Browser.setUserAgent
type SetUserAgentParams struct {
	UserAgent string `json:"userAgent"`
}

// GetElementsParams for Page.getElements
type GetElementsParams struct {
	TabID *string `json:"tabId,omitempty"` // Optional tab ID (defaults to active tab)
}

// UploadFileParams for Page.uploadFile
type UploadFileParams struct {
	Index     int      `json:"index"`             // element index from getElements
	FilePaths []string `json:"filePaths"`         // absolute file paths to upload
	TabID     *string  `json:"tabId,omitempty"`   // Optional tab ID (defaults to active tab)
}

// UploadFileResult for Page.uploadFile response
type UploadFileResult struct {
	FilesUploaded int      `json:"filesUploaded"` // number of files uploaded
	FileNames     []string `json:"fileNames"`     // names of uploaded files
}

// GetTextParams for Page.getText
type GetTextParams struct {
	Strategy string  `json:"strategy,omitempty"`  // auto, article, main, body (default: auto)
	TabID    *string `json:"tabId,omitempty"`     // Optional tab ID (defaults to active tab)
}

// GetTextResult for Page.getText response
type GetTextResult struct {
	Text      string `json:"text"`      // extracted text content
	Length    int    `json:"length"`    // character count
	Source    string `json:"source"`    // element source: article, main, or body
	Truncated bool   `json:"truncated"` // true if exceeded size limit
}

// FindParams for Page.find
type FindParams struct {
	Query string  `json:"query"`              // keyword query
	Limit int     `json:"limit,omitempty"`    // max results (default: 100)
	TabID *string `json:"tabId,omitempty"`    // Optional tab ID (defaults to active tab)
}

// FindResult for Page.find response
type FindResult struct {
	Elements   []RawAccessibilityNode `json:"elements"`             // found elements
	Total      int                    `json:"total"`                // total matches
	Truncated  bool                   `json:"truncated"`            // true if limited
	ResultFile string                 `json:"resultFile,omitempty"` // file path if >100 results
}

// SuccessResult for operations that only return success/failure
type SuccessResult struct {
	Success bool `json:"success"`
}

// WaitParams for Page.wait
type WaitParams struct {
	Duration int     `json:"duration"`        // milliseconds
	TabID    *string `json:"tabId,omitempty"` // Optional tab ID (defaults to active tab)
}

// WaitResult for Page.wait response
type WaitResult struct {
	Waited int `json:"waited"` // actual milliseconds waited
}

// PressKeyParams for Page.pressKey method
type PressKeyParams struct {
	Keys  string  `json:"keys"`              // Space-separated key sequence (e.g., "Enter", "cmd+a", "Backspace Backspace")
	TabID *string `json:"tabId,omitempty"`   // Optional tab ID
}

// ScrollIntoViewParams for Page.scrollIntoView method
type ScrollIntoViewParams struct {
	Index     *int    `json:"index,omitempty"`     // Element index (mutually exclusive with BackendID)
	BackendID *int64  `json:"backendId,omitempty"` // CDP backend node ID (mutually exclusive with Index)
	TabID     *string `json:"tabId,omitempty"`     // Optional tab ID
}

// --- Error Codes ---
const (
	ErrCodeInvalidRequest  = -32600
	ErrCodeMethodNotFound  = -32601
	ErrCodeInvalidParams   = -32602
	ErrCodeInternalError   = -32603
	ErrCodeBrowserError    = -32000
	ErrCodeNavigationError = -32001
	ErrCodeElementNotFound = -32002
	ErrCodeTimeout         = -32003
	ErrCodeFileNotFound    = -32004
	ErrCodeInvalidElement  = -32005
	ErrCodePathTraversal   = -32006
)

// NewError creates a new error response
func NewError(code int, message string) *Error {
	return &Error{Code: code, Message: message}
}

// NewResponse creates a success response
func NewResponse(id string, result interface{}) Response {
	return Response{ID: id, Result: result}
}

// NewErrorResponse creates an error response
func NewErrorResponse(id string, err *Error) Response {
	return Response{ID: id, Error: err}
}
