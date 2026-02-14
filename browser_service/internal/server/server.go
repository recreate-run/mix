package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sarathmenon/browser-service/internal/browser"
	"github.com/sarathmenon/browser-service/internal/browser/events"
	"github.com/sarathmenon/browser-service/internal/constants"
	"github.com/sarathmenon/browser-service/pkg/protocol"
)

// Config holds server configuration
type Config struct {
	Port             string
	Headless         bool
	Stealth          bool   // Enable stealth mode (disable automation detection)
	WindowWidth      int    // Browser window width (default: 1280)
	WindowHeight     int    // Browser window height (default: 720)
	StorageStatePath string // Path to save/load storage state (empty to disable)
}

// Server represents the WebSocket server
type Server struct {
	config        Config
	upgrader      websocket.Upgrader
	manager       *browser.Manager
	clients       map[string]*Client
	mu            sync.RWMutex
	srv           *http.Server
	ctx           context.Context
	cancel        context.CancelFunc
	clientCounter uint64
}

// Client represents a connected WebSocket client
type Client struct {
	ID      string
	Conn    *websocket.Conn
	Context *browser.Context
	done    chan struct{}
}

// New creates a new server instance
func New(ctx context.Context, cfg Config) (*Server, error) {
	// Create browser config from server config
	browserCfg := browser.Config{
		Headless:         cfg.Headless,
		Stealth:          cfg.Stealth,
		WindowWidth:      cfg.WindowWidth,
		WindowHeight:     cfg.WindowHeight,
		StorageStatePath: cfg.StorageStatePath,
	}

	mgr, err := browser.NewManager(ctx, browserCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser manager: %w", err)
	}

	srvCtx, cancel := context.WithCancel(ctx)

	return &Server{
		config: cfg,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
		manager: mgr,
		clients: make(map[string]*Client),
		ctx:     srvCtx,
		cancel:  cancel,
	}, nil
}

// Start starts the WebSocket server
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc(constants.WebSocketEndpoint, s.handleWebSocket)
	mux.HandleFunc(constants.HealthEndpoint, s.handleHealth)

	addr := ":" + s.config.Port
	if s.config.Port == "" {
		addr = ":" + constants.DefaultPort
	}

	s.srv = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	return s.srv.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	// Cancel server context
	s.cancel()

	// Close all client connections
	s.mu.Lock()
	for id, client := range s.clients {
		close(client.done)
		if err := client.Conn.Close(); err != nil {
			log.Printf("Error closing client connection %s: %v", id, err)
		}
		if client.Context != nil {
			if err := client.Context.Close(ctx); err != nil {
				log.Printf("Error closing client context %s: %v", id, err)
			}
		}
		delete(s.clients, id)
	}
	s.mu.Unlock()

	// Close browser manager
	if err := s.manager.Close(ctx); err != nil {
		log.Printf("Error closing browser manager: %v", err)
	}

	// Shutdown HTTP server
	if s.srv != nil {
		return s.srv.Shutdown(ctx)
	}

	return nil
}

// handleHealth returns server health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
		log.Printf("Failed to write health response: %v", err)
	}
}

// handleWebSocket handles new WebSocket connections
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Check concurrent connection limit
	s.mu.RLock()
	if len(s.clients) >= constants.MaxConcurrentConns {
		s.mu.RUnlock()
		log.Printf("Max concurrent connections reached")
		if err := conn.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
		return
	}
	s.mu.RUnlock()

	// Generate client ID using atomic counter for thread safety
	clientID := fmt.Sprintf("client-%d", atomic.AddUint64(&s.clientCounter, 1))

	// Create browser context for this client
	//nolint:contextcheck // s.ctx is the server context, not request context
	ctx, err := s.manager.NewContext(s.ctx)
	if err != nil {
		log.Printf("Failed to create browser context: %v", err)
		if err := conn.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
		return
	}

	client := &Client{
		ID:      clientID,
		Conn:    conn,
		Context: ctx,
		done:    make(chan struct{}),
	}

	s.mu.Lock()
	s.clients[clientID] = client
	s.mu.Unlock()

	log.Printf("Client connected: %s", clientID)

	// Start event forwarding goroutine
	go s.forwardBrowserEvents(client)

	// Start handling messages
	go s.handleClient(client)
}

// forwardBrowserEvents forwards browser events to the WebSocket client
func (s *Server) forwardBrowserEvents(client *Client) {
	eventChan := client.Context.SubscribeToBrowserEvents(s.ctx)

	for {
		select {
		case <-client.done:
			return
		case <-s.ctx.Done():
			return
		case event, ok := <-eventChan:
			if !ok {
				// Event channel closed
				return
			}

			// Convert event to protocol event
			var protocolEvent protocol.Event
			if errorEvent, ok := event.(events.BrowserErrorEvent); ok {
				protocolEvent = protocol.Event{
					Method: "Browser.errorOccurred",
					Params: protocol.BrowserErrorEventParams{
						ErrorType: errorEvent.ErrorType,
						Details:   errorEvent.Details,
					},
				}
			} else {
				// Unknown event type, skip
				continue
			}

			// Send event to client
			if err := client.Conn.WriteJSON(protocolEvent); err != nil {
				log.Printf("Error forwarding event to client %s: %v", client.ID, err)
				return
			}
		}
	}
}

// handleClient processes messages from a client
func (s *Server) handleClient(client *Client) {
	defer func() {
		s.mu.Lock()
		delete(s.clients, client.ID)
		s.mu.Unlock()

		if client.Context != nil {
			if err := client.Context.Close(s.ctx); err != nil {
				log.Printf("Error closing client context: %v", err)
			}
		}
		if err := client.Conn.Close(); err != nil {
			log.Printf("Error closing client connection: %v", err)
		}
		log.Printf("Client disconnected: %s", client.ID)
	}()

	handler := NewMessageHandler(client)

	for {
		select {
		case <-client.done:
			return
		case <-s.ctx.Done():
			return
		default:
			_, message, err := client.Conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				return
			}

			response := handler.Handle(s.ctx, message)
			if err := client.Conn.WriteJSON(response); err != nil {
				log.Printf("Write error: %v", err)
				return
			}
		}
	}
}
