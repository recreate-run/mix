package test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/sarathmenon/browser-service/internal/server"
	"github.com/sarathmenon/browser-service/pkg/client"
	"github.com/sarathmenon/browser-service/pkg/protocol"
)

const (
	wikipediaCatsURL = "https://en.wikipedia.org/wiki/Cat"
	aboutBlankURL    = "about:blank"

	text10Chars  = "HelloWorld"
	text50Chars  = "The quick brown fox jumps over the lazy dog. Hello!"
	text100Chars = "The quick brown fox jumps over the lazy dog. The quick brown fox jumps over the lazy dog. Hello World!"
)

// setupBenchmark creates a test browser service and client for benchmarking
func setupBenchmark(b *testing.B) (*client.Client, context.Context, func()) {
	ctx := context.Background()

	// Get a free port
	lc := net.ListenConfig{}
	listener, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("Failed to get free port: %v", err)
	}
	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		b.Fatalf("Failed to get TCP address")
	}
	port := tcpAddr.Port
	if err := listener.Close(); err != nil {
		b.Fatalf("Failed to close listener: %v", err)
	}

	// Create server with default settings
	srv, err := server.New(ctx, server.Config{
		Port:        fmt.Sprintf("%d", port),
		Headless:    true,
		BlockModals: true,
	})
	if err != nil {
		b.Fatalf("Failed to create server: %v", err)
	}

	// Start server in background
	go func() {
		_ = srv.Start()
	}()

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)

	wsURL := fmt.Sprintf("ws://127.0.0.1:%d/ws", port)

	// Create client
	c, err := client.New(wsURL)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	cleanup := func() {
		_ = c.Close()
		_ = srv.Shutdown(ctx)
	}

	return c, ctx, cleanup
}

// ==============================================================================
// Tab Management Benchmarks
// ==============================================================================

func BenchmarkTabCreate(b *testing.B) {
	c, ctx, cleanup := setupBenchmark(b)
	defer cleanup()

	b.Run("BlankTab", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			tab, err := c.CreateTab(ctx)
			if err != nil {
				b.Fatalf("Failed to create blank tab: %v", err)
			}
			// Cleanup after each iteration
			_ = c.CloseTab(ctx, tab.ID)
		}
	})

	b.Run("WithURL", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			tab, err := c.CreateTab(ctx, aboutBlankURL)
			if err != nil {
				b.Fatalf("Failed to create tab with URL: %v", err)
			}
			_ = c.CloseTab(ctx, tab.ID)
		}
	})
}

func BenchmarkTabList(b *testing.B) {
	c, ctx, cleanup := setupBenchmark(b)
	defer cleanup()

	// Create some tabs for listing
	tab1, _ := c.CreateTab(ctx)
	tab2, _ := c.CreateTab(ctx, aboutBlankURL)
	defer c.CloseTab(ctx, tab1.ID)
	defer c.CloseTab(ctx, tab2.ID)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.ListTabs(ctx)
		if err != nil {
			b.Fatalf("Failed to list tabs: %v", err)
		}
	}
}

func BenchmarkTabSwitch(b *testing.B) {
	c, ctx, cleanup := setupBenchmark(b)
	defer cleanup()

	// Create two tabs to switch between
	tab1, _ := c.CreateTab(ctx)
	tab2, _ := c.CreateTab(ctx, aboutBlankURL)
	defer c.CloseTab(ctx, tab1.ID)
	defer c.CloseTab(ctx, tab2.ID)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			_ = c.SwitchTab(ctx, tab1.ID)
		} else {
			_ = c.SwitchTab(ctx, tab2.ID)
		}
	}
}

func BenchmarkTabClose(b *testing.B) {
	c, ctx, cleanup := setupBenchmark(b)
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		tab, err := c.CreateTab(ctx)
		if err != nil {
			b.Fatalf("Failed to create tab: %v", err)
		}
		b.StartTimer()

		err = c.CloseTab(ctx, tab.ID)
		if err != nil {
			b.Fatalf("Failed to close tab: %v", err)
		}
	}
}

// ==============================================================================
// Navigation Benchmarks
// ==============================================================================

func BenchmarkNavigate(b *testing.B) {
	c, ctx, cleanup := setupBenchmark(b)
	defer cleanup()

	tab, _ := c.CreateTab(ctx)
	defer c.CloseTab(ctx, tab.ID)

	b.Run("Wikipedia", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := c.Navigate(ctx, wikipediaCatsURL, tab.ID)
			if err != nil {
				b.Fatalf("Failed to navigate: %v", err)
			}
		}
	})

	b.Run("AboutBlank", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := c.Navigate(ctx, aboutBlankURL, tab.ID)
			if err != nil {
				b.Fatalf("Failed to navigate: %v", err)
			}
		}
	})
}

func BenchmarkGoBack(b *testing.B) {
	c, ctx, cleanup := setupBenchmark(b)
	defer cleanup()

	tab, _ := c.CreateTab(ctx)
	defer c.CloseTab(ctx, tab.ID)

	// Navigate to create history
	_, _ = c.Navigate(ctx, aboutBlankURL, tab.ID)
	_, _ = c.Navigate(ctx, wikipediaCatsURL, tab.ID)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		if i > 0 {
			// Go forward to setup for next back
			_, _ = c.GoForward(ctx, tab.ID)
		}
		b.StartTimer()

		_, err := c.GoBack(ctx, tab.ID)
		if err != nil {
			b.Fatalf("Failed to go back: %v", err)
		}
	}
}

func BenchmarkGoForward(b *testing.B) {
	c, ctx, cleanup := setupBenchmark(b)
	defer cleanup()

	tab, _ := c.CreateTab(ctx)
	defer c.CloseTab(ctx, tab.ID)

	// Navigate to create history
	_, _ = c.Navigate(ctx, aboutBlankURL, tab.ID)
	_, _ = c.Navigate(ctx, wikipediaCatsURL, tab.ID)
	_, _ = c.GoBack(ctx, tab.ID)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		if i > 0 {
			// Go back to setup for next forward
			_, _ = c.GoBack(ctx, tab.ID)
		}
		b.StartTimer()

		_, err := c.GoForward(ctx, tab.ID)
		if err != nil {
			b.Fatalf("Failed to go forward: %v", err)
		}
	}
}

// ==============================================================================
// Page Interaction Benchmarks
// ==============================================================================

func BenchmarkClick(b *testing.B) {
	c, ctx, cleanup := setupBenchmark(b)
	defer cleanup()

	tab, _ := c.CreateTab(ctx, wikipediaCatsURL)
	defer c.CloseTab(ctx, tab.ID)

	// Wait for page to load
	time.Sleep(2 * time.Second)

	// Get elements to click on
	elements, err := c.GetElements(ctx, tab.ID)
	if err != nil || len(elements) == 0 {
		b.Skip("No elements found to click")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Click first interactive element
		err := c.Click(ctx, 0, tab.ID)
		if err != nil {
			b.Fatalf("Failed to click: %v", err)
		}
	}
}

func BenchmarkType(b *testing.B) {
	c, ctx, cleanup := setupBenchmark(b)
	defer cleanup()

	tab, _ := c.CreateTab(ctx, wikipediaCatsURL)
	defer c.CloseTab(ctx, tab.ID)

	// Wait for page to load
	time.Sleep(2 * time.Second)

	// Get elements - find a search box or input field
	elements, err := c.GetElements(ctx, tab.ID)
	if err != nil || len(elements) == 0 {
		b.Skip("No elements found to type into")
	}

	// Find first input element
	inputIndex := -1
	for i, elem := range elements {
		if elem.Role == "textbox" || elem.Role == "searchbox" {
			inputIndex = i
			break
		}
	}
	if inputIndex == -1 {
		b.Skip("No input element found")
	}

	b.Run("10Chars", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := c.Type(ctx, inputIndex, text10Chars, tab.ID)
			if err != nil {
				b.Fatalf("Failed to type: %v", err)
			}
		}
	})

	b.Run("50Chars", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := c.Type(ctx, inputIndex, text50Chars, tab.ID)
			if err != nil {
				b.Fatalf("Failed to type: %v", err)
			}
		}
	})

	b.Run("100Chars", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := c.Type(ctx, inputIndex, text100Chars, tab.ID)
			if err != nil {
				b.Fatalf("Failed to type: %v", err)
			}
		}
	})
}

func BenchmarkScroll(b *testing.B) {
	c, ctx, cleanup := setupBenchmark(b)
	defer cleanup()

	tab, _ := c.CreateTab(ctx, wikipediaCatsURL)
	defer c.CloseTab(ctx, tab.ID)

	// Wait for page to load
	time.Sleep(2 * time.Second)

	b.Run("GestureBased", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := c.Scroll(ctx, "down", 500, tab.ID)
			if err != nil {
				b.Fatalf("Failed to scroll: %v", err)
			}
			// Scroll back up
			_ = c.Scroll(ctx, "up", 500, tab.ID)
		}
	})

	b.Run("InstantViaEvalJS", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := c.EvalJS(ctx, "window.scrollTo(0, 500)", tab.ID)
			if err != nil {
				b.Fatalf("Failed to scroll via EvalJS: %v", err)
			}
			// Scroll back up
			_, _ = c.EvalJS(ctx, "window.scrollTo(0, 0)", tab.ID)
		}
	})
}

// ==============================================================================
// JavaScript Evaluation Benchmarks
// ==============================================================================

func BenchmarkEvalJS(b *testing.B) {
	c, ctx, cleanup := setupBenchmark(b)
	defer cleanup()

	tab, _ := c.CreateTab(ctx, aboutBlankURL)
	defer c.CloseTab(ctx, tab.ID)

	b.Run("Simple_Arithmetic", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := c.EvalJS(ctx, "2 + 2", tab.ID)
			if err != nil {
				b.Fatalf("Failed to eval JS: %v", err)
			}
		}
	})

	b.Run("Simple_Typeof", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := c.EvalJS(ctx, "typeof window", tab.ID)
			if err != nil {
				b.Fatalf("Failed to eval JS: %v", err)
			}
		}
	})

	b.Run("Complex_DOMQuery", func(b *testing.B) {
		// Navigate to Wikipedia for complex DOM
		_, _ = c.Navigate(ctx, wikipediaCatsURL, tab.ID)
		time.Sleep(2 * time.Second)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := c.EvalJS(ctx, "document.querySelectorAll('a').length", tab.ID)
			if err != nil {
				b.Fatalf("Failed to eval JS: %v", err)
			}
		}
	})

	b.Run("Complex_ArrayOps", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := c.EvalJS(ctx, "Array.from({length: 1000}, (_, i) => i * 2).reduce((a, b) => a + b, 0)", tab.ID)
			if err != nil {
				b.Fatalf("Failed to eval JS: %v", err)
			}
		}
	})
}

// ==============================================================================
// Accessibility Benchmarks
// ==============================================================================

func BenchmarkReadPage(b *testing.B) {
	c, ctx, cleanup := setupBenchmark(b)
	defer cleanup()

	tab, _ := c.CreateTab(ctx)
	defer c.CloseTab(ctx, tab.ID)

	b.Run("ComplexDOM_Wikipedia", func(b *testing.B) {
		_, _ = c.Navigate(ctx, wikipediaCatsURL, tab.ID)
		time.Sleep(2 * time.Second)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result, err := c.ReadPage(ctx, false, tab.ID)
			if err != nil {
				b.Fatalf("Failed to read page: %v", err)
			}
			b.ReportMetric(float64(len(result.Elements)), "nodes")
		}
	})

	b.Run("SimpleDOM_AboutBlank", func(b *testing.B) {
		_, _ = c.Navigate(ctx, aboutBlankURL, tab.ID)
		time.Sleep(500 * time.Millisecond)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result, err := c.ReadPage(ctx, false, tab.ID)
			if err != nil {
				b.Fatalf("Failed to read page: %v", err)
			}
			b.ReportMetric(float64(len(result.Elements)), "nodes")
		}
	})
}

// ==============================================================================
// Screenshot Benchmarks
// ==============================================================================

func BenchmarkScreenshot(b *testing.B) {
	c, ctx, cleanup := setupBenchmark(b)
	defer cleanup()

	tab, _ := c.CreateTab(ctx, wikipediaCatsURL)
	defer c.CloseTab(ctx, tab.ID)

	// Wait for page to load
	time.Sleep(2 * time.Second)

	b.Run("PNG", func(b *testing.B) {
		tabIDCopy := tab.ID
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result, err := c.Screenshot(ctx, protocol.ScreenshotParams{
				Format: "png",
				TabID:  &tabIDCopy,
			})
			if err != nil {
				b.Fatalf("Failed to take screenshot: %v", err)
			}
			b.ReportMetric(float64(len(result.Data)), "bytes")
		}
	})

	b.Run("WithElements", func(b *testing.B) {
		tabIDCopy := tab.ID
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Take screenshot
			screenshotResult, err := c.Screenshot(ctx, protocol.ScreenshotParams{
				Format: "png",
				TabID:  &tabIDCopy,
			})
			if err != nil {
				b.Fatalf("Failed to take screenshot: %v", err)
			}

			// Get elements
			elements, err := c.GetElements(ctx, tab.ID)
			if err != nil {
				b.Fatalf("Failed to get elements: %v", err)
			}

			b.ReportMetric(float64(len(screenshotResult.Data)), "screenshot_bytes")
			b.ReportMetric(float64(len(elements)), "element_count")
		}
	})
}

// ==============================================================================
// Stress Test Benchmarks
// ==============================================================================

func BenchmarkStressConcurrentTabs(b *testing.B) {
	c, ctx, cleanup := setupBenchmark(b)
	defer cleanup()

	const numTabs = 10

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		tabIDs := make([]string, numTabs)
		b.StartTimer()

		// Create tabs concurrently
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				for j := 0; j < numTabs; j++ {
					tab, err := c.CreateTab(ctx)
					if err != nil {
						b.Fatalf("Failed to create tab: %v", err)
					}
					tabIDs[j] = tab.ID
				}
			}
		})

		b.StopTimer()
		// Cleanup tabs
		for _, tabID := range tabIDs {
			if tabID != "" {
				_ = c.CloseTab(ctx, tabID)
			}
		}
		b.StartTimer()
	}
}

func BenchmarkStressRapidTabSwitch(b *testing.B) {
	c, ctx, cleanup := setupBenchmark(b)
	defer cleanup()

	const numSwitches = 10

	// Create tabs to switch between
	tabs := make([]string, numSwitches)
	for i := 0; i < numSwitches; i++ {
		tab, _ := c.CreateTab(ctx)
		tabs[i] = tab.ID
	}
	defer func() {
		for _, tabID := range tabs {
			_ = c.CloseTab(ctx, tabID)
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tabID := range tabs {
			err := c.SwitchTab(ctx, tabID)
			if err != nil {
				b.Fatalf("Failed to switch tab: %v", err)
			}
		}
	}
}
