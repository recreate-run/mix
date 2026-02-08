package browser

import (
	"context"

	browserprotocol "github.com/sarathmenon/browser-service/pkg/protocol"
)

// BrowserClient is the common interface for both service and tunnel browser clients
// All methods used by browser tool handlers must be defined here
type BrowserClient interface {
	// Navigation
	Navigate(ctx context.Context, url string, tabID ...string) (*browserprotocol.NavigateResult, error)
	GoBack(ctx context.Context, tabID ...string) (string, error)
	GoForward(ctx context.Context, tabID ...string) (string, error)

	// Screenshots and Page Reading
	Screenshot(ctx context.Context, params browserprotocol.ScreenshotParams) (*browserprotocol.ScreenshotResult, error)
	ReadPage(ctx context.Context, interactiveOnly bool, tabID ...string) (*browserprotocol.ReadPageResult, error)
	GetText(ctx context.Context, strategy string, tabID ...string) (*browserprotocol.GetTextResult, error)
	Find(ctx context.Context, query string, limit int, tabID ...string) (*browserprotocol.FindResult, error)

	// Mouse Actions
	Click(ctx context.Context, index int, tabID ...string) error
	ClickByBackendID(ctx context.Context, backendID int64, tabID ...string) error
	RightClick(ctx context.Context, index int, tabID ...string) error
	RightClickByBackendID(ctx context.Context, backendID int64, tabID ...string) error
	DoubleClick(ctx context.Context, index int, tabID ...string) error
	DoubleClickByBackendID(ctx context.Context, backendID int64, tabID ...string) error
	TripleClick(ctx context.Context, index int, tabID ...string) error
	TripleClickByBackendID(ctx context.Context, backendID int64, tabID ...string) error
	Drag(ctx context.Context, fromIndex, toIndex *int, fromX, fromY, toX, toY *float64, duration *int, tabID ...string) error

	// Keyboard Actions
	Type(ctx context.Context, index int, text string, tabID ...string) error
	PressKey(ctx context.Context, keys string, tabID ...string) error
	FormInput(ctx context.Context, index int, value string, tabID ...string) error

	// Scrolling
	Scroll(ctx context.Context, direction string, amount int, tabID ...string) error
	ScrollIntoView(ctx context.Context, index *int, backendID *int64, tabID ...string) error
	ScrollIntoViewByIndex(ctx context.Context, index int, tabID ...string) error
	ScrollIntoViewByBackendID(ctx context.Context, backendID int64, tabID ...string) error

	// File Upload
	UploadFile(ctx context.Context, index int, filePaths []string, tabID ...string) (*browserprotocol.UploadFileResult, error)

	// Tab Management
	CreateTab(ctx context.Context, url ...string) (*browserprotocol.TabInfo, error)
	ListTabs(ctx context.Context) (*browserprotocol.TabListResult, error)
	SwitchTab(ctx context.Context, tabID string) error
	CloseTab(ctx context.Context, tabID string) error

	// Waiting
	Wait(ctx context.Context, duration int, tabID ...string) error

	// Connection Management
	Close() error
}
