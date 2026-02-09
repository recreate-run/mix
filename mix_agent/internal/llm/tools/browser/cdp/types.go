package cdp

// Chrome DevTools Protocol typed message structs
// These replace map[string]interface{} usage for type safety

// Command represents a CDP command request
type Command struct {
	ID        int         `json:"id"`
	Method    string      `json:"method"`
	Params    interface{} `json:"params,omitempty"`
	SessionID string      `json:"sessionId,omitempty"`
}

// Response represents a CDP command response
type Response struct {
	ID     int         `json:"id"`
	Result interface{} `json:"result,omitempty"`
	Error  *Error      `json:"error,omitempty"`
}

// Error represents a CDP error
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// Target domain commands

type TargetCreateParams struct {
	URL string `json:"url"`
}

type TargetCreateResult struct {
	TargetID string `json:"targetId"`
}

type TargetAttachParams struct {
	TargetID string `json:"targetId"`
	Flatten  bool   `json:"flatten"`
}

type TargetAttachResult struct {
	SessionID string `json:"sessionId"`
}

type TargetCloseParams struct {
	TargetID string `json:"targetId"`
}

// Page domain commands

type PageNavigateParams struct {
	URL string `json:"url"`
}

type PageNavigateResult struct {
	FrameID  string `json:"frameId"`
	LoaderID string `json:"loaderId"`
}

type PageNavigateToHistoryParams struct {
	EntryID int `json:"entryId"`
}

type PageCaptureScreenshotParams struct {
	Format      string `json:"format"`
	FromSurface bool   `json:"fromSurface"`
	Quality     int    `json:"quality,omitempty"`
}

type PageCaptureScreenshotResult struct {
	Data string `json:"data"`
}

type PageGetLayoutMetricsResult struct {
	LayoutViewport    LayoutViewport    `json:"layoutViewport"`
	VisualViewport    VisualViewport    `json:"visualViewport"`
	CSSLayoutViewport CSSLayoutViewport `json:"cssLayoutViewport"`
}

type LayoutViewport struct {
	PageX        float64 `json:"pageX"`
	PageY        float64 `json:"pageY"`
	ClientWidth  float64 `json:"clientWidth"`
	ClientHeight float64 `json:"clientHeight"`
}

type VisualViewport struct {
	PageX        float64 `json:"pageX"`
	PageY        float64 `json:"pageY"`
	ClientWidth  float64 `json:"clientWidth"`
	ClientHeight float64 `json:"clientHeight"`
}

type CSSLayoutViewport struct {
	PageX        float64 `json:"pageX"`
	PageY        float64 `json:"pageY"`
	ClientWidth  float64 `json:"clientWidth"`
	ClientHeight float64 `json:"clientHeight"`
}

// Accessibility domain commands

type AccessibilityGetFullAXTreeResult struct {
	Nodes []AccessibilityNode `json:"nodes"`
}

type AccessibilityNode struct {
	Role            RoleValue               `json:"role"`
	Name            StringValue             `json:"name"`
	BackendDOMNodeID int64                  `json:"backendDOMNodeId"`
	FrameID         string                  `json:"frameId"`
	BoundingBox     *BoundingBox            `json:"boundingBox,omitempty"`
	Properties      []AccessibilityProperty `json:"properties,omitempty"`
}

type RoleValue struct {
	Value interface{} `json:"value"`
}

type StringValue struct {
	Value interface{} `json:"value"`
}

type BoundingBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type AccessibilityProperty struct {
	Name  string      `json:"name"`
	Value StringValue `json:"value"`
}

// DOMSnapshot domain commands

type DOMSnapshotCaptureSnapshotParams struct {
	ComputedStyles  []string `json:"computedStyles"`
	IncludeDOMRects bool     `json:"includeDOMRects,omitempty"`
	IncludePaintOrder bool   `json:"includePaintOrder,omitempty"`
}

type DOMSnapshotCaptureSnapshotResult struct {
	Documents []DOMSnapshotDocument `json:"documents"`
}

type DOMSnapshotDocument struct {
	Nodes  DOMSnapshotNodeTree  `json:"nodes"`
	Layout DOMSnapshotLayoutTree `json:"layout"`
}

type DOMSnapshotNodeTree struct {
	BackendNodeID []int64 `json:"backendNodeId"`
}

type DOMSnapshotLayoutTree struct {
	NodeIndex []int     `json:"nodeIndex"`
	Bounds    [][]float64 `json:"bounds"`
}

// Runtime domain commands

type RuntimeEvaluateParams struct {
	Expression    string `json:"expression"`
	ReturnByValue bool   `json:"returnByValue"`
}

type RuntimeEvaluateResult struct {
	Result           RuntimeRemoteObject `json:"result"`
	ExceptionDetails interface{}         `json:"exceptionDetails,omitempty"`
}

type RuntimeRemoteObject struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

// Input domain commands

type InputDispatchMouseEventParams struct {
	Type       string  `json:"type"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Button     string  `json:"button,omitempty"`
	ClickCount int     `json:"clickCount,omitempty"`
	DeltaX     float64 `json:"deltaX,omitempty"`
	DeltaY     float64 `json:"deltaY,omitempty"`
}

type InputInsertTextParams struct {
	Text string `json:"text"`
}

// DOM domain commands

type DOMClickParams struct {
	BackendNodeID int64 `json:"backendNodeId"`
}
