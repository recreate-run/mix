package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestBrowserError(t *testing.T) {
	underlyingErr := fmt.Errorf("connection failed")
	err := &BrowserError{
		Op:  "launch",
		Err: underlyingErr,
	}

	// Test Error() string
	expected := "browser error in launch: connection failed"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}

	// Test Unwrap()
	if unwrapped := err.Unwrap(); !errors.Is(unwrapped, underlyingErr) {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, underlyingErr)
	}

	// Test errors.Is()
	if !errors.Is(err, underlyingErr) {
		t.Error("errors.Is() should return true for underlying error")
	}

	// Test errors.As()
	var browserErr *BrowserError
	if !errors.As(err, &browserErr) {
		t.Error("errors.As() should return true for BrowserError")
	}
	if browserErr.Op != "launch" {
		t.Errorf("As() Op = %q, want %q", browserErr.Op, "launch")
	}
}

func TestNavigationError(t *testing.T) {
	underlyingErr := fmt.Errorf("timeout")
	err := &NavigationError{
		URL: "https://example.com",
		Err: underlyingErr,
	}

	// Test Error() string includes URL
	expected := "navigation to https://example.com failed: timeout"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}

	// Test Unwrap()
	if unwrapped := err.Unwrap(); !errors.Is(unwrapped, underlyingErr) {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, underlyingErr)
	}

	// Test error wrapping
	if !errors.Is(err, underlyingErr) {
		t.Error("errors.Is() should return true for underlying error")
	}

	// Test errors.As()
	var navErr *NavigationError
	if !errors.As(err, &navErr) {
		t.Error("errors.As() should return true for NavigationError")
	}
	if navErr.URL != "https://example.com" {
		t.Errorf("As() URL = %q, want %q", navErr.URL, "https://example.com")
	}
}

func TestElementError(t *testing.T) {
	underlyingErr := fmt.Errorf("element not found")
	err := &ElementError{
		Index: 5,
		Op:    "click",
		Err:   underlyingErr,
	}

	// Test Error() string includes index and operation
	expected := "element 5 click failed: element not found"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}

	// Test Unwrap()
	if unwrapped := err.Unwrap(); !errors.Is(unwrapped, underlyingErr) {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, underlyingErr)
	}

	// Test errors.As()
	var elemErr *ElementError
	if !errors.As(err, &elemErr) {
		t.Error("errors.As() should return true for ElementError")
	}
	if elemErr.Index != 5 {
		t.Errorf("As() Index = %d, want 5", elemErr.Index)
	}
	if elemErr.Op != "click" {
		t.Errorf("As() Op = %q, want %q", elemErr.Op, "click")
	}
}

func TestContextError(t *testing.T) {
	underlyingErr := fmt.Errorf("context already closed")
	err := &ContextError{
		Op:  "navigate",
		Err: underlyingErr,
	}

	// Test Error() string
	expected := "context navigate failed: context already closed"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}

	// Test Unwrap()
	if unwrapped := err.Unwrap(); !errors.Is(unwrapped, underlyingErr) {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, underlyingErr)
	}

	// Test errors.As()
	var ctxErr *ContextError
	if !errors.As(err, &ctxErr) {
		t.Error("errors.As() should return true for ContextError")
	}
	if ctxErr.Op != "navigate" {
		t.Errorf("As() Op = %q, want %q", ctxErr.Op, "navigate")
	}
}

func TestValidationError(t *testing.T) {
	underlyingErr := fmt.Errorf("must not be empty")
	err := &ValidationError{
		Field: "url",
		Value: "",
		Err:   underlyingErr,
	}

	// Test Error() string includes field and value
	expected := "validation failed for url=: must not be empty"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}

	// Test Unwrap()
	if unwrapped := err.Unwrap(); !errors.Is(unwrapped, underlyingErr) {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, underlyingErr)
	}

	// Test with non-empty value
	err2 := &ValidationError{
		Field: "timeout",
		Value: -1,
		Err:   fmt.Errorf("must be positive"),
	}
	expected2 := "validation failed for timeout=-1: must be positive"
	if err2.Error() != expected2 {
		t.Errorf("Error() = %q, want %q", err2.Error(), expected2)
	}

	// Test errors.As()
	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Error("errors.As() should return true for ValidationError")
	}
	if valErr.Field != "url" {
		t.Errorf("As() Field = %q, want %q", valErr.Field, "url")
	}
}

func TestNewBrowserError(t *testing.T) {
	underlyingErr := fmt.Errorf("test error")
	err := NewBrowserError("connect", underlyingErr)

	var browserErr *BrowserError
	if !errors.As(err, &browserErr) {
		t.Fatal("NewBrowserError should return *BrowserError")
	}

	if browserErr.Op != "connect" {
		t.Errorf("Op = %q, want %q", browserErr.Op, "connect")
	}
	if !errors.Is(browserErr.Err, underlyingErr) {
		t.Errorf("Err = %v, want %v", browserErr.Err, underlyingErr)
	}
}

func TestNewNavigationError(t *testing.T) {
	underlyingErr := fmt.Errorf("test error")
	err := NewNavigationError("https://test.com", underlyingErr)

	var navErr *NavigationError
	if !errors.As(err, &navErr) {
		t.Fatal("NewNavigationError should return *NavigationError")
	}

	if navErr.URL != "https://test.com" {
		t.Errorf("URL = %q, want %q", navErr.URL, "https://test.com")
	}
	if !errors.Is(navErr.Err, underlyingErr) {
		t.Errorf("Err = %v, want %v", navErr.Err, underlyingErr)
	}
}

func TestNewElementError(t *testing.T) {
	underlyingErr := fmt.Errorf("test error")
	err := NewElementError(3, "type", underlyingErr)

	var elemErr *ElementError
	if !errors.As(err, &elemErr) {
		t.Fatal("NewElementError should return *ElementError")
	}

	if elemErr.Index != 3 {
		t.Errorf("Index = %d, want 3", elemErr.Index)
	}
	if elemErr.Op != "type" {
		t.Errorf("Op = %q, want %q", elemErr.Op, "type")
	}
	if !errors.Is(elemErr.Err, underlyingErr) {
		t.Errorf("Err = %v, want %v", elemErr.Err, underlyingErr)
	}
}

func TestNewContextError(t *testing.T) {
	underlyingErr := fmt.Errorf("test error")
	err := NewContextError("close", underlyingErr)

	var ctxErr *ContextError
	if !errors.As(err, &ctxErr) {
		t.Fatal("NewContextError should return *ContextError")
	}

	if ctxErr.Op != "close" {
		t.Errorf("Op = %q, want %q", ctxErr.Op, "close")
	}
	if !errors.Is(ctxErr.Err, underlyingErr) {
		t.Errorf("Err = %v, want %v", ctxErr.Err, underlyingErr)
	}
}

func TestNewValidationError(t *testing.T) {
	underlyingErr := fmt.Errorf("test error")
	err := NewValidationError("field", "value", underlyingErr)

	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Fatal("NewValidationError should return *ValidationError")
	}

	if valErr.Field != "field" {
		t.Errorf("Field = %q, want %q", valErr.Field, "field")
	}
	if valErr.Value != "value" {
		t.Errorf("Value = %v, want %v", valErr.Value, "value")
	}
	if !errors.Is(valErr.Err, underlyingErr) {
		t.Errorf("Err = %v, want %v", valErr.Err, underlyingErr)
	}
}

func TestErrorWrapping(t *testing.T) {
	// Test multiple levels of wrapping
	baseErr := fmt.Errorf("base error")
	navErr := NewNavigationError("https://example.com", baseErr)
	browserErr := NewBrowserError("navigate", navErr)

	// Should be able to unwrap through multiple levels
	if !errors.Is(browserErr, baseErr) {
		t.Error("errors.Is() should find base error through multiple levels")
	}

	// Should be able to extract intermediate errors
	var nav *NavigationError
	if !errors.As(browserErr, &nav) {
		t.Error("errors.As() should find NavigationError in chain")
	}
}
