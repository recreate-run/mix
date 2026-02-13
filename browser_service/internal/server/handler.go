package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/sarathmenon/browser-service/internal/constants"
	"github.com/sarathmenon/browser-service/pkg/protocol"
)

// MessageHandler handles incoming WebSocket messages
type MessageHandler struct {
	client *Client
}

// NewMessageHandler creates a new message handler
func NewMessageHandler(client *Client) *MessageHandler {
	return &MessageHandler{client: client}
}

// Handle processes an incoming message and returns a response
func (h *MessageHandler) Handle(ctx context.Context, data []byte) protocol.Response {
	var req protocol.Request
	if err := json.Unmarshal(data, &req); err != nil {
		return protocol.NewErrorResponse("",
			protocol.NewError(protocol.ErrCodeInvalidRequest, "Invalid JSON"))
	}

	log.Printf("[%s] %s", h.client.ID, req.Method)

	switch req.Method {
	case constants.MethodPageNavigate:
		return h.handleNavigate(ctx, req)
	case constants.MethodPageScreenshot:
		return h.handleScreenshot(ctx, req)
	case constants.MethodPageGetElements:
		return h.handleGetElements(ctx, req)
	case constants.MethodPageReadPage:
		return h.handleReadPage(ctx, req)
	case constants.MethodPageClick:
		return h.handleClick(ctx, req)
	case constants.MethodPageClickByBackendID:
		return h.handleClickByBackendID(ctx, req)
	case constants.MethodPageType:
		return h.handleType(ctx, req)
	case constants.MethodPageScroll:
		return h.handleScroll(ctx, req)
	case constants.MethodPageUploadFile:
		return h.handleUploadFile(ctx, req)
	case constants.MethodPageGetText:
		return h.handleGetText(ctx, req)
	case constants.MethodPageFind:
		return h.handleFind(ctx, req)
	case constants.MethodBrowserClose:
		return h.handleClose(ctx, req)
	case constants.MethodBrowserImportCookies:
		return h.handleImportCookies(ctx, req)
	case constants.MethodBrowserSetUserAgent:
		return h.handleSetUserAgent(ctx, req)
	case constants.MethodPageRightClick:
		return h.handleRightClick(ctx, req)
	case constants.MethodPageRightClickByBackendID:
		return h.handleRightClickByBackendID(ctx, req)
	case constants.MethodPageDoubleClick:
		return h.handleDoubleClick(ctx, req)
	case constants.MethodPageDoubleClickByBackendID:
		return h.handleDoubleClickByBackendID(ctx, req)
	case constants.MethodPageTripleClick:
		return h.handleTripleClick(ctx, req)
	case constants.MethodPageTripleClickByBackendID:
		return h.handleTripleClickByBackendID(ctx, req)
	case constants.MethodPageDrag:
		return h.handleDrag(ctx, req)
	case constants.MethodPageFormInput:
		return h.handleFormInput(ctx, req)
	case constants.MethodPageGoBack:
		return h.handleGoBack(ctx, req)
	case constants.MethodPageGoForward:
		return h.handleGoForward(ctx, req)
	case constants.MethodPageWait:
		return h.handleWait(ctx, req)
	case constants.MethodTabCreate:
		return h.handleTabCreate(ctx, req)
	case constants.MethodTabList:
		return h.handleTabList(ctx, req)
	case constants.MethodTabSwitch:
		return h.handleTabSwitch(ctx, req)
	case constants.MethodTabClose:
		return h.handleTabClose(ctx, req)
	case constants.MethodPagePressKey:
		return h.handlePressKey(ctx, req)
	case constants.MethodPageScrollIntoView:
		return h.handleScrollIntoView(ctx, req)
	default:
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeMethodNotFound, "Method not found: "+req.Method))
	}
}

// handleNavigate navigates to a URL
func (h *MessageHandler) handleNavigate(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.NavigateParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
	}

	if params.URL == "" {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "URL is required"))
	}

	result, err := h.client.Context.Navigate(ctx, params.URL, params.Timeout, params.TabID)
	if err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeNavigationError, err.Error()))
	}

	return protocol.NewResponse(req.ID, result)
}

// handleScreenshot captures a screenshot
func (h *MessageHandler) handleScreenshot(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.ScreenshotParams
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return protocol.NewErrorResponse(req.ID,
				protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
		}
	}

	// Default to PNG
	if params.Format == "" {
		params.Format = constants.DefaultImageFormat
	}

	result, err := h.client.Context.Screenshot(ctx, params)
	if err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeBrowserError, err.Error()))
	}

	return protocol.NewResponse(req.ID, result)
}

// handleGetElements returns interactive elements on the page
func (h *MessageHandler) handleGetElements(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.GetElementsParams
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return protocol.NewErrorResponse(req.ID,
				protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
		}
	}

	elements, err := h.client.Context.GetElements(ctx, params.TabID)
	if err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeBrowserError, err.Error()))
	}

	return protocol.NewResponse(req.ID, protocol.GetElementsResult{Elements: elements})
}

// handleReadPage returns accessibility tree for visible viewport elements
func (h *MessageHandler) handleReadPage(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.ReadPageParams
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return protocol.NewErrorResponse(req.ID,
				protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
		}
	}

	elements, viewport, err := h.client.Context.ReadPage(ctx, params.InteractiveOnly, params.TabID)
	if err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeBrowserError, err.Error()))
	}

	return protocol.NewResponse(req.ID, protocol.ReadPageResult{
		Elements: elements,
		Viewport: *viewport,
	})
}

// handleClick clicks an element by index
func (h *MessageHandler) handleClick(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.ClickParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
	}

	if err := h.client.Context.Click(ctx, params.Index, params.TabID); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeElementNotFound, err.Error()))
	}

	return protocol.NewResponse(req.ID, protocol.SuccessResult{Success: true})
}

// handleRightClick right-clicks an element
func (h *MessageHandler) handleRightClick(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.RightClickParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
	}

	if err := h.client.Context.RightClick(ctx, params.Index, params.TabID); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeElementNotFound, err.Error()))
	}

	return protocol.NewResponse(req.ID, protocol.SuccessResult{Success: true})
}

// handleDoubleClick double-clicks an element
func (h *MessageHandler) handleDoubleClick(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.DoubleClickParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
	}

	if err := h.client.Context.DoubleClick(ctx, params.Index, params.TabID); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeElementNotFound, err.Error()))
	}

	return protocol.NewResponse(req.ID, protocol.SuccessResult{Success: true})
}

// handleTripleClick triple-clicks an element
func (h *MessageHandler) handleTripleClick(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.TripleClickParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
	}

	if err := h.client.Context.TripleClick(ctx, params.Index, params.TabID); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeElementNotFound, err.Error()))
	}

	return protocol.NewResponse(req.ID, protocol.SuccessResult{Success: true})
}

// handleClickByBackendID clicks an element by its CDP backend node ID
func (h *MessageHandler) handleClickByBackendID(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.ClickByBackendIDParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
	}

	if err := h.client.Context.ClickByBackendID(ctx, params.BackendID, params.TabID); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeElementNotFound, err.Error()))
	}

	return protocol.NewResponse(req.ID, protocol.SuccessResult{Success: true})
}

// handleRightClickByBackendID right-clicks an element by its CDP backend node ID
func (h *MessageHandler) handleRightClickByBackendID(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.RightClickByBackendIDParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
	}

	if err := h.client.Context.RightClickByBackendID(ctx, params.BackendID, params.TabID); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeElementNotFound, err.Error()))
	}

	return protocol.NewResponse(req.ID, protocol.SuccessResult{Success: true})
}

// handleDoubleClickByBackendID double-clicks an element by its CDP backend node ID
func (h *MessageHandler) handleDoubleClickByBackendID(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.DoubleClickByBackendIDParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
	}

	if err := h.client.Context.DoubleClickByBackendID(ctx, params.BackendID, params.TabID); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeElementNotFound, err.Error()))
	}

	return protocol.NewResponse(req.ID, protocol.SuccessResult{Success: true})
}

// handleTripleClickByBackendID triple-clicks an element by its CDP backend node ID
func (h *MessageHandler) handleTripleClickByBackendID(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.TripleClickByBackendIDParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
	}

	if err := h.client.Context.TripleClickByBackendID(ctx, params.BackendID, params.TabID); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeElementNotFound, err.Error()))
	}

	return protocol.NewResponse(req.ID, protocol.SuccessResult{Success: true})
}

// handleDrag performs a drag operation
func (h *MessageHandler) handleDrag(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.DragParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
	}

	if err := h.client.Context.Drag(ctx, params.FromIndex, params.ToIndex, params.FromX, params.FromY, params.ToX, params.ToY, params.Duration, params.TabID); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeElementNotFound, err.Error()))
	}

	return protocol.NewResponse(req.ID, protocol.SuccessResult{Success: true})
}

// handleFormInput sets form input value directly
func (h *MessageHandler) handleFormInput(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.FormInputParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
	}

	if err := h.client.Context.FormInput(ctx, params.Index, params.Value, params.TabID); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeElementNotFound, err.Error()))
	}

	return protocol.NewResponse(req.ID, protocol.SuccessResult{Success: true})
}

// handleGoBack navigates backward in browser history
func (h *MessageHandler) handleGoBack(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.GoBackParams
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return protocol.NewErrorResponse(req.ID,
				protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
		}
	}

	url, err := h.client.Context.GoBack(ctx, params.TabID)
	if err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInternalError, err.Error()))
	}

	return protocol.NewResponse(req.ID, protocol.GoBackResult{URL: url})
}

// handleGoForward navigates forward in browser history
func (h *MessageHandler) handleGoForward(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.GoForwardParams
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return protocol.NewErrorResponse(req.ID,
				protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
		}
	}

	url, err := h.client.Context.GoForward(ctx, params.TabID)
	if err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInternalError, err.Error()))
	}

	return protocol.NewResponse(req.ID, protocol.GoForwardResult{URL: url})
}

// handleWait pauses execution for specified milliseconds
func (h *MessageHandler) handleWait(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.WaitParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
	}

	if params.Duration <= 0 {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "Duration must be positive"))
	}

	if err := h.client.Context.Wait(ctx, params.Duration, params.TabID); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeBrowserError, err.Error()))
	}

	return protocol.NewResponse(req.ID, protocol.WaitResult{Waited: params.Duration})
}

// handleType types text into an element
func (h *MessageHandler) handleType(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.TypeParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
	}

	if err := h.client.Context.Type(ctx, params.Index, params.Text, params.TabID); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeElementNotFound, err.Error()))
	}

	return protocol.NewResponse(req.ID, protocol.SuccessResult{Success: true})
}

// handleScroll scrolls the page
func (h *MessageHandler) handleScroll(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.ScrollParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
	}

	if err := h.client.Context.Scroll(ctx, params.Direction, params.Amount, params.TabID); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeBrowserError, err.Error()))
	}

	return protocol.NewResponse(req.ID, protocol.SuccessResult{Success: true})
}

// handleClose closes the browser context
func (h *MessageHandler) handleClose(ctx context.Context, req protocol.Request) protocol.Response {
	if err := h.client.Context.Close(ctx); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeBrowserError, err.Error()))
	}

	return protocol.NewResponse(req.ID, protocol.SuccessResult{Success: true})
}

// handleImportCookies imports cookies from a browser
func (h *MessageHandler) handleImportCookies(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.ImportCookiesParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
	}

	if params.Browser == "" {
		params.Browser = constants.DefaultBrowser
	}
	if params.Profile == "" {
		params.Profile = constants.DefaultBrowserProfile
	}

	result, err := h.client.Context.ImportCookies(ctx, params.Browser, params.Profile, nil)
	if err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeBrowserError, err.Error()))
	}

	return protocol.NewResponse(req.ID, result)
}

// handleSetUserAgent sets the user agent
func (h *MessageHandler) handleSetUserAgent(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.SetUserAgentParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
	}

	if err := h.client.Context.SetUserAgent(ctx, params.UserAgent, nil); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeBrowserError, err.Error()))
	}

	return protocol.NewResponse(req.ID, protocol.SuccessResult{Success: true})
}

// handleTabCreate creates a new tab
func (h *MessageHandler) handleTabCreate(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.TabCreateParams
	if req.Params != nil && len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return protocol.NewErrorResponse(req.ID,
				protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
		}
	}

	tab, err := h.client.Context.CreateTab(ctx)
	if err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeBrowserError, err.Error()))
	}

	// If URL provided, navigate immediately
	if params.URL != nil && *params.URL != "" {
		_, err := h.client.Context.Navigate(ctx, *params.URL, 0, &tab.ID)
		if err != nil {
			return protocol.NewErrorResponse(req.ID,
				protocol.NewError(protocol.ErrCodeNavigationError,
					fmt.Sprintf("Tab created but navigation failed: %v", err)))
		}

		// Refresh tab info after navigation
		tabs, _, err := h.client.Context.ListTabs(ctx)
		if err == nil {
			for _, t := range tabs {
				if t.ID == tab.ID {
					tab = &t
					break
				}
			}
		}
	}

	return protocol.NewResponse(req.ID, protocol.TabCreateResult{Tab: *tab})
}

// handleTabList lists all tabs
func (h *MessageHandler) handleTabList(ctx context.Context, req protocol.Request) protocol.Response {
	tabs, activeTabID, err := h.client.Context.ListTabs(ctx)
	if err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeBrowserError, err.Error()))
	}

	return protocol.NewResponse(req.ID, protocol.TabListResult{
		Tabs:        tabs,
		ActiveTabID: activeTabID,
	})
}

// handleTabSwitch switches the active tab
func (h *MessageHandler) handleTabSwitch(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.TabSwitchParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
	}

	if params.TabID == "" {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "TabID is required"))
	}

	err := h.client.Context.SwitchTab(ctx, params.TabID)
	if err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeBrowserError, err.Error()))
	}

	return protocol.NewResponse(req.ID, protocol.SuccessResult{Success: true})
}

// handleTabClose closes a tab
func (h *MessageHandler) handleTabClose(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.TabCloseParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
	}

	if params.TabID == "" {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "TabID is required"))
	}

	err := h.client.Context.CloseTab(ctx, params.TabID)
	if err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeBrowserError, err.Error()))
	}

	return protocol.NewResponse(req.ID, protocol.SuccessResult{Success: true})
}

// handlePressKey handles key press requests
func (h *MessageHandler) handlePressKey(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.PressKeyParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
	}

	if params.Keys == "" {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "Keys parameter is required"))
	}

	if err := h.client.Context.PressKey(ctx, params.Keys, params.TabID); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeBrowserError, err.Error()))
	}

	return protocol.NewResponse(req.ID, protocol.SuccessResult{Success: true})
}

// handleScrollIntoView handles scroll into view requests
func (h *MessageHandler) handleScrollIntoView(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.ScrollIntoViewParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
	}

	// Validate exactly one of index or backendId
	if (params.Index == nil && params.BackendID == nil) ||
		(params.Index != nil && params.BackendID != nil) {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams,
				"Exactly one of index or backendId must be provided"))
	}

	if err := h.client.Context.ScrollIntoView(ctx, params.Index, params.BackendID, params.TabID); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeBrowserError, err.Error()))
	}

	return protocol.NewResponse(req.ID, protocol.SuccessResult{Success: true})
}
