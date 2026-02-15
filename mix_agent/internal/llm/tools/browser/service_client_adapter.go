package browser

import (
	"context"
	"fmt"

	browserclient "github.com/sarathmenon/browser-service/pkg/client"
	browserprotocol "github.com/sarathmenon/browser-service/pkg/protocol"
)

// ServiceClientAdapter wraps the external browserclient.Client to implement BrowserClient interface
// This adapter bridges the external browser-service client with our internal interface requirements
type ServiceClientAdapter struct {
	client *browserclient.Client
}

// NewServiceClientAdapter creates a new service client adapter
func NewServiceClientAdapter(client *browserclient.Client) *ServiceClientAdapter {
	return &ServiceClientAdapter{
		client: client,
	}
}

// Navigate navigates to a URL
func (a *ServiceClientAdapter) Navigate(ctx context.Context, url string, tabID ...string) (*browserprotocol.NavigateResult, error) {
	return a.client.Navigate(ctx, url, tabID...)
}

// GoBack navigates back in history
func (a *ServiceClientAdapter) GoBack(ctx context.Context, tabID ...string) (string, error) {
	return a.client.GoBack(ctx, tabID...)
}

// GoForward navigates forward in history
func (a *ServiceClientAdapter) GoForward(ctx context.Context, tabID ...string) (string, error) {
	return a.client.GoForward(ctx, tabID...)
}

// Screenshot captures a screenshot
func (a *ServiceClientAdapter) Screenshot(ctx context.Context, params browserprotocol.ScreenshotParams) (*browserprotocol.ScreenshotResult, error) {
	return a.client.Screenshot(ctx, params)
}

// ReadPage reads the accessibility tree
func (a *ServiceClientAdapter) ReadPage(ctx context.Context, interactiveOnly bool, tabID ...string) (*browserprotocol.ReadPageResult, error) {
	return a.client.ReadPage(ctx, interactiveOnly, tabID...)
}

// GetText extracts text from the page
func (a *ServiceClientAdapter) GetText(ctx context.Context, strategy string, tabID ...string) (*browserprotocol.GetTextResult, error) {
	return a.client.GetText(ctx, strategy, tabID...)
}

// Find searches for elements
func (a *ServiceClientAdapter) Find(ctx context.Context, query string, limit int, tabID ...string) (*browserprotocol.FindResult, error) {
	return a.client.Find(ctx, query, limit, tabID...)
}

// GetElements returns all interactive elements (populates click cache in browser-service)
func (a *ServiceClientAdapter) GetElements(ctx context.Context, tabID ...string) ([]browserprotocol.RawAccessibilityNode, error) {
	return a.client.GetElements(ctx, tabID...)
}

// Click clicks an element by index
func (a *ServiceClientAdapter) Click(ctx context.Context, index int, tabID ...string) error {
	return a.client.Click(ctx, index, tabID...)
}

// ClickByBackendID clicks an element by backend ID
func (a *ServiceClientAdapter) ClickByBackendID(ctx context.Context, backendID int64, tabID ...string) error {
	return a.client.ClickByBackendID(ctx, backendID, tabID...)
}

// RightClick right-clicks an element by index
func (a *ServiceClientAdapter) RightClick(ctx context.Context, index int, tabID ...string) error {
	return a.client.RightClick(ctx, index, tabID...)
}

// RightClickByBackendID right-clicks an element by backend ID
func (a *ServiceClientAdapter) RightClickByBackendID(ctx context.Context, backendID int64, tabID ...string) error {
	return a.client.RightClickByBackendID(ctx, backendID, tabID...)
}

// DoubleClick double-clicks an element by index
func (a *ServiceClientAdapter) DoubleClick(ctx context.Context, index int, tabID ...string) error {
	return a.client.DoubleClick(ctx, index, tabID...)
}

// DoubleClickByBackendID double-clicks an element by backend ID
func (a *ServiceClientAdapter) DoubleClickByBackendID(ctx context.Context, backendID int64, tabID ...string) error {
	return a.client.DoubleClickByBackendID(ctx, backendID, tabID...)
}

// TripleClick triple-clicks an element by index
func (a *ServiceClientAdapter) TripleClick(ctx context.Context, index int, tabID ...string) error {
	return a.client.TripleClick(ctx, index, tabID...)
}

// TripleClickByBackendID triple-clicks an element by backend ID
func (a *ServiceClientAdapter) TripleClickByBackendID(ctx context.Context, backendID int64, tabID ...string) error {
	return a.client.TripleClickByBackendID(ctx, backendID, tabID...)
}

// ClickAt clicks at specific coordinates
func (a *ServiceClientAdapter) ClickAt(ctx context.Context, x, y float64, button *string, clickCount, duration *int, tabID ...string) error {
	return a.client.ClickAt(ctx, x, y, button, clickCount, duration, tabID...)
}

// RightClickAt right-clicks at specific coordinates
func (a *ServiceClientAdapter) RightClickAt(ctx context.Context, x, y float64, duration *int, tabID ...string) error {
	return a.client.RightClickAt(ctx, x, y, duration, tabID...)
}

// DoubleClickAt double-clicks at specific coordinates
func (a *ServiceClientAdapter) DoubleClickAt(ctx context.Context, x, y float64, button *string, duration *int, tabID ...string) error {
	return a.client.DoubleClickAt(ctx, x, y, button, duration, tabID...)
}

// TripleClickAt triple-clicks at specific coordinates
func (a *ServiceClientAdapter) TripleClickAt(ctx context.Context, x, y float64, button *string, duration *int, tabID ...string) error {
	return a.client.TripleClickAt(ctx, x, y, button, duration, tabID...)
}

// Drag performs a drag operation
func (a *ServiceClientAdapter) Drag(ctx context.Context, fromIndex, toIndex *int, fromX, fromY, toX, toY *float64, duration *int, tabID ...string) error {
	return a.client.Drag(ctx, fromIndex, toIndex, fromX, fromY, toX, toY, duration, tabID...)
}

// Type types text into an element
func (a *ServiceClientAdapter) Type(ctx context.Context, index *int, text string, tabID ...string) error {
	return a.client.Type(ctx, index, text, tabID...)
}

// PressKey presses keyboard keys
func (a *ServiceClientAdapter) PressKey(ctx context.Context, keys string, tabID ...string) error {
	return a.client.PressKey(ctx, keys, tabID...)
}

// FormInput sets form input value
func (a *ServiceClientAdapter) FormInput(ctx context.Context, index int, value string, tabID ...string) error {
	return a.client.FormInput(ctx, index, value, tabID...)
}

// Scroll scrolls the page
func (a *ServiceClientAdapter) Scroll(ctx context.Context, direction string, amount int, tabID ...string) error {
	return a.client.Scroll(ctx, direction, amount, tabID...)
}

// ScrollIntoView scrolls an element into view
func (a *ServiceClientAdapter) ScrollIntoView(ctx context.Context, index *int, backendID *int64, tabID ...string) error {
	if backendID != nil {
		return a.ScrollIntoViewByBackendID(ctx, *backendID, tabID...)
	}
	if index != nil {
		return a.ScrollIntoViewByIndex(ctx, *index, tabID...)
	}
	return fmt.Errorf("either index or backendID must be provided")
}

// ScrollIntoViewByIndex scrolls an element into view by index
func (a *ServiceClientAdapter) ScrollIntoViewByIndex(ctx context.Context, index int, tabID ...string) error {
	return a.client.ScrollIntoViewByIndex(ctx, index, tabID...)
}

// ScrollIntoViewByBackendID scrolls an element into view by backend ID
func (a *ServiceClientAdapter) ScrollIntoViewByBackendID(ctx context.Context, backendID int64, tabID ...string) error {
	return a.client.ScrollIntoViewByBackendID(ctx, backendID, tabID...)
}

// UploadFile uploads files to a file input element
func (a *ServiceClientAdapter) UploadFile(ctx context.Context, index int, filePaths []string, tabID ...string) (*browserprotocol.UploadFileResult, error) {
	return a.client.UploadFile(ctx, index, filePaths, tabID...)
}

// CreateTab creates a new tab
func (a *ServiceClientAdapter) CreateTab(ctx context.Context, url ...string) (*browserprotocol.TabInfo, error) {
	return a.client.CreateTab(ctx, url...)
}

// ListTabs lists all tabs
func (a *ServiceClientAdapter) ListTabs(ctx context.Context) (*browserprotocol.TabListResult, error) {
	return a.client.ListTabs(ctx)
}

// SwitchTab switches to a different tab
func (a *ServiceClientAdapter) SwitchTab(ctx context.Context, tabID string) error {
	return a.client.SwitchTab(ctx, tabID)
}

// CloseTab closes a tab
func (a *ServiceClientAdapter) CloseTab(ctx context.Context, tabID string) error {
	return a.client.CloseTab(ctx, tabID)
}

// Wait pauses execution
func (a *ServiceClientAdapter) Wait(ctx context.Context, duration int, tabID ...string) error {
	return a.client.Wait(ctx, duration, tabID...)
}

// Close closes the client connection
func (a *ServiceClientAdapter) Close() error {
	return a.client.Close()
}
