package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sarathmenon/browser-service/pkg/protocol"
)

func skipIfIntegrationTestsDisabled(t *testing.T) {
	t.Helper()
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration test")
	}
}

func TestWebSocketConnection(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx := context.Background()

	// Create server
	srv, err := New(ctx, Config{Port: "0", Headless: true})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer func() {
		if err := srv.Shutdown(ctx); err != nil {
			t.Errorf("Failed to shutdown server: %v", err)
		}
	}()

	// Create test HTTP server
	httpServer := httptest.NewServer(http.HandlerFunc(srv.handleWebSocket))
	defer httpServer.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http")

	// Connect WebSocket client
	//nolint:bodyclose // websocket connection doesn't have a response body to close
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect WebSocket: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Logf("Failed to close connection: %v", err)
		}
	}()

	// Send navigate command
	req := protocol.Request{
		ID:     "1",
		Method: "Page.navigate",
		Params: json.RawMessage(`{"url":"https://example.com"}`),
	}

	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read response with timeout
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatalf("Failed to set read deadline: %v", err)
	}

	var resp protocol.Response
	if err := conn.ReadJSON(&resp); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// Verify response
	if resp.ID != "1" {
		t.Errorf("Expected response ID '1', got '%s'", resp.ID)
	}

	if resp.Error != nil {
		t.Errorf("Expected no error, got: %v", resp.Error)
	}

	if resp.Result == nil {
		t.Error("Expected non-nil result")
	}
}

func TestContextIsolation(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx := context.Background()

	// Create server
	srv, err := New(ctx, Config{Port: "0", Headless: true})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer func() {
		if err := srv.Shutdown(ctx); err != nil {
			t.Errorf("Failed to shutdown server: %v", err)
		}
	}()

	// Create test HTTP server
	httpServer := httptest.NewServer(http.HandlerFunc(srv.handleWebSocket))
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http")

	// Connect first client
	//nolint:bodyclose // websocket connection doesn't have a response body to close
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect first client: %v", err)
	}
	defer func() {
		if err := conn1.Close(); err != nil {
			t.Logf("Failed to close connection 1: %v", err)
		}
	}()

	// Connect second client
	//nolint:bodyclose // websocket connection doesn't have a response body to close
	conn2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect second client: %v", err)
	}
	defer func() {
		if err := conn2.Close(); err != nil {
			t.Logf("Failed to close connection 2: %v", err)
		}
	}()

	// Navigate first client to example.com
	req1 := protocol.Request{
		ID:     "1",
		Method: "Page.navigate",
		Params: json.RawMessage(`{"url":"https://example.com"}`),
	}

	if err := conn1.WriteJSON(req1); err != nil {
		t.Fatalf("Failed to send first request: %v", err)
	}

	// Navigate second client to different page
	req2 := protocol.Request{
		ID:     "2",
		Method: "Page.navigate",
		Params: json.RawMessage(`{"url":"https://www.iana.org"}`),
	}

	if err := conn2.WriteJSON(req2); err != nil {
		t.Fatalf("Failed to send second request: %v", err)
	}

	// Read responses with timeout
	if err := conn1.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatalf("Failed to set read deadline for conn1: %v", err)
	}
	if err := conn2.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatalf("Failed to set read deadline for conn2: %v", err)
	}

	var resp1, resp2 protocol.Response

	if err := conn1.ReadJSON(&resp1); err != nil {
		t.Fatalf("Failed to read first response: %v", err)
	}

	if err := conn2.ReadJSON(&resp2); err != nil {
		t.Fatalf("Failed to read second response: %v", err)
	}

	// Verify both responses succeeded
	if resp1.Error != nil {
		t.Errorf("First client got error: %v", resp1.Error)
	}

	if resp2.Error != nil {
		t.Errorf("Second client got error: %v", resp2.Error)
	}

	// Wait for pages to load
	time.Sleep(1 * time.Second)

	// Take screenshots from both
	screenshotReq1 := protocol.Request{
		ID:     "3",
		Method: "Page.screenshot",
		Params: json.RawMessage(`{}`),
	}

	screenshotReq2 := protocol.Request{
		ID:     "4",
		Method: "Page.screenshot",
		Params: json.RawMessage(`{}`),
	}

	if err := conn1.WriteJSON(screenshotReq1); err != nil {
		t.Fatalf("Failed to send first screenshot request: %v", err)
	}

	if err := conn2.WriteJSON(screenshotReq2); err != nil {
		t.Fatalf("Failed to send second screenshot request: %v", err)
	}

	if err := conn1.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatalf("Failed to set read deadline for conn1: %v", err)
	}
	if err := conn2.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatalf("Failed to set read deadline for conn2: %v", err)
	}

	var screenshotResp1, screenshotResp2 protocol.Response

	if err := conn1.ReadJSON(&screenshotResp1); err != nil {
		t.Fatalf("Failed to read first screenshot response: %v", err)
	}

	if err := conn2.ReadJSON(&screenshotResp2); err != nil {
		t.Fatalf("Failed to read second screenshot response: %v", err)
	}

	// Verify screenshots are different
	if screenshotResp1.Result == nil || screenshotResp2.Result == nil {
		t.Fatal("Expected screenshot results")
	}

	// Marshal results to compare
	data1, err := json.Marshal(screenshotResp1.Result)
	if err != nil {
		t.Fatalf("Failed to marshal screenshot1: %v", err)
	}
	data2, err := json.Marshal(screenshotResp2.Result)
	if err != nil {
		t.Fatalf("Failed to marshal screenshot2: %v", err)
	}

	var screenshot1, screenshot2 protocol.ScreenshotResult
	if err := json.Unmarshal(data1, &screenshot1); err != nil {
		t.Fatalf("Failed to unmarshal screenshot1: %v", err)
	}
	if err := json.Unmarshal(data2, &screenshot2); err != nil {
		t.Fatalf("Failed to unmarshal screenshot2: %v", err)
	}

	if screenshot1.Data == screenshot2.Data {
		t.Error("Expected different screenshots from isolated contexts")
	}
}

func TestWebSocketInvalidJSON(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx := context.Background()

	srv, err := New(ctx, Config{Port: "0", Headless: true})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer func() {
		if err := srv.Shutdown(ctx); err != nil {
			t.Errorf("Failed to shutdown server: %v", err)
		}
	}()

	httpServer := httptest.NewServer(http.HandlerFunc(srv.handleWebSocket))
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http")

	//nolint:bodyclose // websocket connection doesn't have a response body to close
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Logf("Failed to close connection: %v", err)
		}
	}()

	// Send invalid JSON
	if err := conn.WriteMessage(websocket.TextMessage, []byte("invalid json")); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("Failed to set read deadline: %v", err)
	}

	var resp protocol.Response
	if err := conn.ReadJSON(&resp); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if resp.Error == nil {
		t.Error("Expected error for invalid JSON")
	}

	if resp.Error.Code != protocol.ErrCodeInvalidRequest {
		t.Errorf("Expected error code %d, got %d", protocol.ErrCodeInvalidRequest, resp.Error.Code)
	}
}

func TestWebSocketUnknownMethod(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx := context.Background()

	srv, err := New(ctx, Config{Port: "0", Headless: true})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer func() {
		if err := srv.Shutdown(ctx); err != nil {
			t.Errorf("Failed to shutdown server: %v", err)
		}
	}()

	httpServer := httptest.NewServer(http.HandlerFunc(srv.handleWebSocket))
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http")

	//nolint:bodyclose // websocket connection doesn't have a response body to close
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Logf("Failed to close connection: %v", err)
		}
	}()

	req := protocol.Request{
		ID:     "1",
		Method: "Unknown.method",
	}

	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("Failed to set read deadline: %v", err)
	}

	var resp protocol.Response
	if err := conn.ReadJSON(&resp); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if resp.Error == nil {
		t.Error("Expected error for unknown method")
	}

	if resp.Error.Code != protocol.ErrCodeMethodNotFound {
		t.Errorf("Expected error code %d, got %d", protocol.ErrCodeMethodNotFound, resp.Error.Code)
	}
}

func TestWebSocketNavigateInvalidParams(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx := context.Background()

	srv, err := New(ctx, Config{Port: "0", Headless: true})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer func() {
		if err := srv.Shutdown(ctx); err != nil {
			t.Errorf("Failed to shutdown server: %v", err)
		}
	}()

	httpServer := httptest.NewServer(http.HandlerFunc(srv.handleWebSocket))
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http")

	//nolint:bodyclose // websocket connection doesn't have a response body to close
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Logf("Failed to close connection: %v", err)
		}
	}()

	// Send navigate without URL
	req := protocol.Request{
		ID:     "1",
		Method: "Page.navigate",
		Params: json.RawMessage(`{}`),
	}

	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("Failed to set read deadline: %v", err)
	}

	var resp protocol.Response
	if err := conn.ReadJSON(&resp); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if resp.Error == nil {
		t.Error("Expected error for missing URL")
	}

	if resp.Error.Code != protocol.ErrCodeInvalidParams {
		t.Errorf("Expected error code %d, got %d", protocol.ErrCodeInvalidParams, resp.Error.Code)
	}
}

func TestMultipleCommands(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx := context.Background()

	srv, err := New(ctx, Config{Port: "0", Headless: true})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer func() {
		if err := srv.Shutdown(ctx); err != nil {
			t.Errorf("Failed to shutdown server: %v", err)
		}
	}()

	httpServer := httptest.NewServer(http.HandlerFunc(srv.handleWebSocket))
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http")

	//nolint:bodyclose // websocket connection doesn't have a response body to close
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Logf("Failed to close connection: %v", err)
		}
	}()

	commands := []struct {
		id     string
		method string
		params string
	}{
		{"1", "Page.navigate", `{"url":"https://example.com"}`},
		{"2", "Page.screenshot", `{}`},
		{"3", "Page.screenshot", `{"fullPage":true}`},
	}

	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		t.Fatalf("Failed to set read deadline: %v", err)
	}

	for _, cmd := range commands {
		req := protocol.Request{
			ID:     cmd.id,
			Method: cmd.method,
			Params: json.RawMessage(cmd.params),
		}

		if err := conn.WriteJSON(req); err != nil {
			t.Fatalf("Failed to send command %s: %v", cmd.id, err)
		}

		var resp protocol.Response
		if err := conn.ReadJSON(&resp); err != nil {
			t.Fatalf("Failed to read response for %s: %v", cmd.id, err)
		}

		if resp.ID != cmd.id {
			t.Errorf("Expected response ID %s, got %s", cmd.id, resp.ID)
		}

		if resp.Error != nil {
			t.Errorf("Command %s returned error: %v", cmd.id, resp.Error)
		}

		if resp.Result == nil {
			t.Errorf("Command %s returned nil result", cmd.id)
		}
	}
}

func TestCleanupOnDisconnect(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx := context.Background()

	srv, err := New(ctx, Config{Port: "0", Headless: true})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer func() {
		if err := srv.Shutdown(ctx); err != nil {
			t.Errorf("Failed to shutdown server: %v", err)
		}
	}()

	httpServer := httptest.NewServer(http.HandlerFunc(srv.handleWebSocket))
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http")

	//nolint:bodyclose // websocket connection doesn't have a response body to close
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Navigate to a page
	req := protocol.Request{
		ID:     "1",
		Method: "Page.navigate",
		Params: json.RawMessage(`{"url":"https://example.com"}`),
	}

	if err := conn.WriteJSON(req); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatalf("Failed to set read deadline: %v", err)
	}

	var resp protocol.Response
	if err := conn.ReadJSON(&resp); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	srv.mu.RLock()
	initialClientCount := len(srv.clients)
	srv.mu.RUnlock()

	if initialClientCount == 0 {
		t.Error("Expected at least one client")
	}

	// Close connection
	if err := conn.Close(); err != nil {
		t.Logf("Failed to close connection: %v", err)
	}

	// Wait for cleanup
	time.Sleep(500 * time.Millisecond)

	// Verify client was removed
	srv.mu.RLock()
	finalClientCount := len(srv.clients)
	srv.mu.RUnlock()

	if finalClientCount >= initialClientCount {
		t.Errorf("Expected client count to decrease, got initial=%d final=%d", initialClientCount, finalClientCount)
	}
}

func TestConcurrentConnections(t *testing.T) {
	skipIfIntegrationTestsDisabled(t)

	ctx := context.Background()

	srv, err := New(ctx, Config{Port: "0", Headless: true})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer func() {
		if err := srv.Shutdown(ctx); err != nil {
			t.Errorf("Failed to shutdown server: %v", err)
		}
	}()

	httpServer := httptest.NewServer(http.HandlerFunc(srv.handleWebSocket))
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http")

	// Connect multiple clients concurrently
	numClients := 5
	done := make(chan error, numClients)

	for i := 0; i < numClients; i++ {
		go func(clientNum int) {
			//nolint:bodyclose // websocket connection doesn't have a response body to close
			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				done <- fmt.Errorf("client %d failed to connect: %w", clientNum, err)
				return
			}
			defer func() {
				if err := conn.Close(); err != nil {
					t.Logf("Client %d failed to close connection: %v", clientNum, err)
				}
			}()

			// Send a navigate command
			req := protocol.Request{
				ID:     fmt.Sprintf("%d", clientNum),
				Method: "Page.navigate",
				Params: json.RawMessage(`{"url":"https://example.com"}`),
			}

			if err := conn.WriteJSON(req); err != nil {
				done <- fmt.Errorf("client %d failed to send: %w", clientNum, err)
				return
			}

			if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
				done <- fmt.Errorf("client %d failed to set deadline: %w", clientNum, err)
				return
			}

			var resp protocol.Response
			if err := conn.ReadJSON(&resp); err != nil {
				done <- fmt.Errorf("client %d failed to read: %w", clientNum, err)
				return
			}

			if resp.Error != nil {
				done <- fmt.Errorf("client %d got error: %v", clientNum, resp.Error)
				return
			}

			done <- nil
		}(i)
	}

	// Wait for all clients to complete
	for i := 0; i < numClients; i++ {
		if err := <-done; err != nil {
			t.Error(err)
		}
	}
}
