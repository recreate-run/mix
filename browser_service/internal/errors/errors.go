package errors

import "fmt"

// Error types for structured error handling

// BrowserError represents a browser-related error
type BrowserError struct {
	Op  string // Operation that failed
	Err error  // Underlying error
}

func (e *BrowserError) Error() string {
	return fmt.Sprintf("browser error in %s: %v", e.Op, e.Err)
}

func (e *BrowserError) Unwrap() error {
	return e.Err
}

// NavigationError represents a navigation failure
type NavigationError struct {
	URL string
	Err error
}

func (e *NavigationError) Error() string {
	return fmt.Sprintf("navigation to %s failed: %v", e.URL, e.Err)
}

func (e *NavigationError) Unwrap() error {
	return e.Err
}

// ElementError represents an element interaction error
type ElementError struct {
	Index int
	Op    string // Operation: click, type, etc.
	Err   error
}

func (e *ElementError) Error() string {
	return fmt.Sprintf("element %d %s failed: %v", e.Index, e.Op, e.Err)
}

func (e *ElementError) Unwrap() error {
	return e.Err
}

// ContextError represents a context lifecycle error
type ContextError struct {
	Op  string
	Err error
}

func (e *ContextError) Error() string {
	return fmt.Sprintf("context %s failed: %v", e.Op, e.Err)
}

func (e *ContextError) Unwrap() error {
	return e.Err
}

// ValidationError represents invalid input
type ValidationError struct {
	Field string
	Value interface{}
	Err   error
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for %s=%v: %v", e.Field, e.Value, e.Err)
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

// Helper constructors

func NewBrowserError(op string, err error) error {
	return &BrowserError{Op: op, Err: err}
}

func NewNavigationError(url string, err error) error {
	return &NavigationError{URL: url, Err: err}
}

func NewElementError(index int, op string, err error) error {
	return &ElementError{Index: index, Op: op, Err: err}
}

func NewContextError(op string, err error) error {
	return &ContextError{Op: op, Err: err}
}

func NewValidationError(field string, value interface{}, err error) error {
	return &ValidationError{Field: field, Value: value, Err: err}
}

// NotImplementedError represents a feature that is not yet implemented
type NotImplementedError struct {
	Feature string
}

func (e *NotImplementedError) Error() string {
	return fmt.Sprintf("feature not implemented: %s", e.Feature)
}

func NewNotImplementedError(feature string) error {
	return &NotImplementedError{Feature: feature}
}

// FileError represents a file-related error
type FileError struct {
	Path string
	Op   string
	Err  error
}

func (e *FileError) Error() string {
	return fmt.Sprintf("file %s in %s failed: %v", e.Op, e.Path, e.Err)
}

func (e *FileError) Unwrap() error {
	return e.Err
}

func NewFileError(path string, op string, err error) error {
	return &FileError{Path: path, Op: op, Err: err}
}
