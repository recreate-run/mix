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

// ClickAtParams for Page.clickAt
type ClickAtParams struct {
	X          float64 `json:"x"`                   // X coordinate
	Y          float64 `json:"y"`                   // Y coordinate
	Button     *string `json:"button,omitempty"`    // left, right, middle (default: left)
	ClickCount *int    `json:"clickCount,omitempty"` // 1, 2, 3 (default: 1)
	Duration   *int    `json:"duration,omitempty"`  // Optional duration in ms (default: 0)
	TabID      *string `json:"tabId,omitempty"`     // Optional tab ID (defaults to active tab)
}

// RightClickAtParams for Page.rightClickAt
type RightClickAtParams struct {
	X        float64 `json:"x"`                  // X coordinate
	Y        float64 `json:"y"`                  // Y coordinate
	Duration *int    `json:"duration,omitempty"` // Optional duration in ms (default: 0)
	TabID    *string `json:"tabId,omitempty"`    // Optional tab ID (defaults to active tab)
}

// DoubleClickAtParams for Page.doubleClickAt
type DoubleClickAtParams struct {
	X        float64 `json:"x"`                  // X coordinate
	Y        float64 `json:"y"`                  // Y coordinate
	Button   *string `json:"button,omitempty"`   // left, right, middle (default: left)
	Duration *int    `json:"duration,omitempty"` // Optional duration in ms (default: 0)
	TabID    *string `json:"tabId,omitempty"`    // Optional tab ID (defaults to active tab)
}

// TripleClickAtParams for Page.tripleClickAt
type TripleClickAtParams struct {
	X        float64 `json:"x"`                  // X coordinate
	Y        float64 `json:"y"`                  // Y coordinate
	Button   *string `json:"button,omitempty"`   // left, right, middle (default: left)
	Duration *int    `json:"duration,omitempty"` // Optional duration in ms (default: 0)
	TabID    *string `json:"tabId,omitempty"`    // Optional tab ID (defaults to active tab)
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
	Index *int    `json:"index,omitempty"`   // Optional element index (if nil, types into currently focused element)
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

// --- Storage State Types ---

// Cookie represents a browser cookie
type Cookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires"`           // Unix timestamp
	HTTPOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
	SameSite string  `json:"sameSite"`          // "Strict", "Lax", "None"
}

// OriginState represents localStorage for a specific origin
type OriginState struct {
	Origin       string            `json:"origin"`
	LocalStorage map[string]string `json:"localStorage"`
}

// StorageState represents combined cookies and localStorage state
type StorageState struct {
	Cookies []Cookie      `json:"cookies"`
	Origins []OriginState `json:"origins"`
}

// GetCookiesParams for Browser.getCookies
type GetCookiesParams struct {
	TabID *string `json:"tabId,omitempty"` // Optional tab ID (defaults to active tab)
}

// GetCookiesResult for Browser.getCookies response
type GetCookiesResult struct {
	Cookies []Cookie `json:"cookies"`
}

// SetCookiesParams for Browser.setCookies
type SetCookiesParams struct {
	Cookies []Cookie `json:"cookies"`
	TabID   *string  `json:"tabId,omitempty"` // Optional tab ID (defaults to active tab)
}

// SetCookiesResult for Browser.setCookies response
type SetCookiesResult struct {
	Set int `json:"set"` // number of cookies set
}

// ClearCookiesParams for Browser.clearCookies
type ClearCookiesParams struct {
	TabID *string `json:"tabId,omitempty"` // Optional tab ID (defaults to active tab)
}

// ClearCookiesResult for Browser.clearCookies response
type ClearCookiesResult struct {
	Cleared int `json:"cleared"` // number of cookies cleared
}

// SaveStorageStateParams for Browser.saveStorageState
type SaveStorageStateParams struct {
	TabID *string `json:"tabId,omitempty"` // Optional tab ID (defaults to active tab)
}

// SaveStorageStateResult for Browser.saveStorageState response
type SaveStorageStateResult struct {
	State StorageState `json:"state"`
}

// LoadStorageStateParams for Browser.loadStorageState
type LoadStorageStateParams struct {
	State StorageState `json:"state"`
	TabID *string      `json:"tabId,omitempty"` // Optional tab ID (defaults to active tab)
}

// LoadStorageStateResult for Browser.loadStorageState response
type LoadStorageStateResult struct {
	Loaded bool `json:"loaded"`
}

// SetLocalStorageParams for Page.setLocalStorage
type SetLocalStorageParams struct {
	Items map[string]string `json:"items"`
	TabID *string           `json:"tabId,omitempty"` // Optional tab ID (defaults to active tab)
}

// SetLocalStorageResult for Page.setLocalStorage response
type SetLocalStorageResult struct {
	Set int `json:"set"` // number of items set
}

// GetLocalStorageParams for Page.getLocalStorage
type GetLocalStorageParams struct {
	TabID *string `json:"tabId,omitempty"` // Optional tab ID (defaults to active tab)
}

// GetLocalStorageResult for Page.getLocalStorage response
type GetLocalStorageResult struct {
	Items map[string]string `json:"items"`
}

// --- Downloads Types ---

// Download represents a browser download
type Download struct {
	GUID              string `json:"guid"`
	URL               string `json:"url"`
	SuggestedFilename string `json:"suggestedFilename"`
	TotalBytes        int64  `json:"totalBytes"`
	State             string `json:"state"` // "inProgress", "completed"
	Path              string `json:"path"`  // Final file path
}

// SetDownloadBehaviorParams for Browser.setDownloadBehavior
type SetDownloadBehaviorParams struct {
	Path   string `json:"path"`   // Download directory path
	Accept bool   `json:"accept"` // Whether to accept downloads
	TabID  *string `json:"tabId,omitempty"` // Optional tab ID (defaults to active tab)
}

// SetDownloadBehaviorResult for Browser.setDownloadBehavior response
type SetDownloadBehaviorResult struct {
	Configured bool `json:"configured"`
}

// GetDownloadsParams for Page.getDownloads
type GetDownloadsParams struct {
	TabID *string `json:"tabId,omitempty"` // Optional tab ID (defaults to active tab)
}

// GetDownloadsResult for Page.getDownloads response
type GetDownloadsResult struct {
	Downloads []Download `json:"downloads"`
}

// WaitForDownloadParams for Page.waitForDownload
type WaitForDownloadParams struct {
	Timeout int     `json:"timeout"`         // Timeout in milliseconds
	TabID   *string `json:"tabId,omitempty"` // Optional tab ID (defaults to active tab)
}

// WaitForDownloadResult for Page.waitForDownload response
type WaitForDownloadResult struct {
	Download Download `json:"download"`
}

// GetClosedPopupMessagesResult for Browser.getClosedPopupMessages response
type GetClosedPopupMessagesResult struct {
	Messages []string `json:"messages"`
}

// EvalJSParams for Page.evalJS
type EvalJSParams struct {
	Expression string  `json:"expression"`
	TabID      *string `json:"tabId,omitempty"` // Optional tab ID (defaults to active tab)
}

// EvalJSResult for Page.evalJS response
type EvalJSResult struct {
	Result interface{} `json:"result"`
}

// --- Error Codes ---
// --- Credential Management Types ---

// LoadTaskCredentialsParams for Browser.loadTaskCredentials
type LoadTaskCredentialsParams struct {
	TestCaseName string `json:"testCaseName"` // Convex test case name
	TaskID       string `json:"taskId"`       // Task ID to fetch credentials for
}

// LoadTaskCredentialsResult for Browser.loadTaskCredentials response
type LoadTaskCredentialsResult struct {
	Loaded       bool   `json:"loaded"`       // Whether credentials were successfully loaded
	CookiesCount int    `json:"cookiesCount"` // Number of cookies loaded
	TaskID       string `json:"taskId"`       // Task ID
}

// --- Browser Event Types ---

// BrowserErrorEventParams represents parameters for Browser.errorOccurred event
type BrowserErrorEventParams struct {
	ErrorType string         `json:"errorType"` // "TargetCrash", "NetworkTimeout", "BrowserUnresponsive"
	Details   map[string]any `json:"details"`   // Event-specific details
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
