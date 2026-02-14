package constants

import "time"

// Protocol methods
const (
	MethodPageNavigate      = "Page.navigate"
	MethodPageScreenshot    = "Page.screenshot"
	MethodPageGetElements   = "Page.getElements"
	MethodPageReadPage      = "Page.readPage"
	MethodPageClick         = "Page.click"
	MethodPageClickByBackendID = "Page.clickByBackendID"
	MethodPageType          = "Page.type"
	MethodPageScroll        = "Page.scroll"
	MethodPageUploadFile    = "Page.uploadFile"
	MethodPageGetText       = "Page.getText"
	MethodPageFind          = "Page.find"
	MethodPageEvalJS        = "Page.evalJS"
	MethodBrowserClose      = "Browser.close"
	MethodBrowserImportCookies = "Browser.importCookies"
	MethodBrowserSetUserAgent  = "Browser.setUserAgent"
	MethodPageRightClick    = "Page.rightClick"
	MethodPageRightClickByBackendID = "Page.rightClickByBackendID"
	MethodPageDoubleClick   = "Page.doubleClick"
	MethodPageDoubleClickByBackendID = "Page.doubleClickByBackendID"
	MethodPageTripleClick   = "Page.tripleClick"
	MethodPageTripleClickByBackendID = "Page.tripleClickByBackendID"
	MethodPageClickAt       = "Page.clickAt"
	MethodPageRightClickAt  = "Page.rightClickAt"
	MethodPageDoubleClickAt = "Page.doubleClickAt"
	MethodPageTripleClickAt = "Page.tripleClickAt"
	MethodPageDrag          = "Page.drag"
	MethodPageFormInput     = "Page.formInput"
	MethodPageGoBack        = "Page.goBack"
	MethodPageGoForward     = "Page.goForward"
	MethodTabCreate         = "Tab.create"
	MethodTabList           = "Tab.list"
	MethodTabSwitch         = "Tab.switch"
	MethodTabClose          = "Tab.close"
	MethodPageWait          = "Page.wait"
	MethodPagePressKey      = "Page.pressKey"
	MethodPageScrollIntoView = "Page.scrollIntoView"
	MethodBrowserGetCookies     = "Browser.getCookies"
	MethodBrowserSetCookies     = "Browser.setCookies"
	MethodBrowserClearCookies   = "Browser.clearCookies"
	MethodBrowserSaveStorageState = "Browser.saveStorageState"
	MethodBrowserLoadStorageState = "Browser.loadStorageState"
	MethodPageSetLocalStorage   = "Page.setLocalStorage"
	MethodPageGetLocalStorage   = "Page.getLocalStorage"
	MethodBrowserSetDownloadBehavior = "Browser.setDownloadBehavior"
	MethodPageGetDownloads        = "Page.getDownloads"
	MethodPageWaitForDownload     = "Page.waitForDownload"
	MethodBrowserGetClosedPopupMessages = "Browser.getClosedPopupMessages"
	MethodBrowserLoadTaskCredentials    = "Browser.loadTaskCredentials"
)

// Timeouts
const (
	DefaultNavigationTimeout = 30 * time.Second
	DefaultScreenshotTimeout = 5 * time.Second
	DefaultClickTimeout      = 2 * time.Second
	DefaultShutdownTimeout   = 10 * time.Second
)

// Browser
const (
	DefaultBrowser        = "chrome"
	DefaultBrowserProfile = "Default"
)

// Screenshot
const (
	DefaultImageFormat = "png"
	ImageFormatPNG     = "png"
	ImageFormatJPEG    = "jpeg"
)

// Scroll directions
const (
	ScrollUp    = "up"
	ScrollDown  = "down"
	ScrollLeft  = "left"
	ScrollRight = "right"
)

// Interactive element roles
const (
	RoleButton           = "button"
	RoleLink             = "link"
	RoleTextbox          = "textbox"
	RoleSearchbox        = "searchbox"
	RoleCombobox         = "combobox"
	RoleListbox          = "listbox"
	RoleMenu             = "menu"
	RoleMenuItem         = "menuitem"
	RoleMenuItemCheckbox = "menuitemcheckbox"
	RoleMenuItemRadio    = "menuitemradio"
	RoleTab              = "tab"
	RoleCheckbox         = "checkbox"
	RoleRadio            = "radio"
	RoleSlider           = "slider"
	RoleSpinbutton       = "spinbutton"
	RoleSwitch           = "switch"
)

// Text extraction
const (
	MaxTextSize         = 1048576 // 1MB in bytes
	TextStrategyAuto    = "auto"
	TextStrategyArticle = "article"
	TextStrategyMain    = "main"
	TextStrategyBody    = "body"
)

// DOM search
const (
	MaxFindResults  = 100
	DefaultFindLimit = 100
)

// Server
const (
	DefaultPort        = "8081"
	HealthEndpoint     = "/health"
	WebSocketEndpoint  = "/ws"
	MaxConcurrentConns = 100
)
