package events

// BrowserEvent is a generic interface for all browser events
type BrowserEvent interface {
	EventType() string
}

// BrowserErrorEvent represents an error event from the browser
type BrowserErrorEvent struct {
	ErrorType string
	Details   map[string]any
}

// EventType returns the event type
func (e BrowserErrorEvent) EventType() string {
	return "error"
}

// TargetCrashedEvent represents a target crash event
type TargetCrashedEvent struct {
	TargetID string
}

// EventType returns the event type
func (e TargetCrashedEvent) EventType() string {
	return "target_crashed"
}

// StorageStateSavedEvent represents a storage state save event
type StorageStateSavedEvent struct {
	Path         string
	CookiesCount int
	OriginsCount int
}

// EventType returns the event type
func (e StorageStateSavedEvent) EventType() string {
	return "storage_state_saved"
}

// StorageStateLoadedEvent represents a storage state load event
type StorageStateLoadedEvent struct {
	Path         string
	CookiesCount int
	OriginsCount int
}

// EventType returns the event type
func (e StorageStateLoadedEvent) EventType() string {
	return "storage_state_loaded"
}

// FileDownloadedEvent represents a file download event
type FileDownloadedEvent struct {
	Download     Download
	AutoDownload bool
	FileType     string
}

// EventType returns the event type
func (e FileDownloadedEvent) EventType() string {
	return "file_downloaded"
}

// Download represents a downloaded file
type Download struct {
	GUID              string
	URL               string
	SuggestedFilename string
	TotalBytes        int64
	State             string
	Path              string
}
