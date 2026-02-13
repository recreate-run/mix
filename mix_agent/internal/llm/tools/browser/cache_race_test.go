package browser

import (
	"context"
	"sync"
	"testing"
)

// TestElementCacheRaceCondition tests concurrent cache access patterns
// that previously caused race conditions in getBackendIDFromCache
func TestElementCacheRaceCondition(t *testing.T) {
	t.Helper()

	// Create browser tool with element cache
	tool := &browserTool{
		elementCache: make(map[string]map[int]int64),
	}

	sessionID := "test-session"
	tabID := "test-tab"
	cacheKey := sessionID + "_" + tabID

	// Populate cache with test data
	tool.cacheMu.Lock()
	tool.elementCache[cacheKey] = map[int]int64{
		0: 100,
		1: 101,
		2: 102,
	}
	tool.cacheMu.Unlock()

	ctx := context.Background()
	var wg sync.WaitGroup

	// Run concurrent reads and writes to trigger race conditions
	// if locks are not held correctly
	iterations := 100

	// Concurrent readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// This would previously race with clearCacheForTab
				_, _ = tool.getBackendIDFromCache(ctx, sessionID, tabID, j%3)
			}
		}()
	}

	// Concurrent cache updates
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// This would race with getBackendIDFromCache reads
				tool.cacheMu.Lock()
				tool.elementCache[cacheKey] = map[int]int64{
					0: 100,
					1: 101,
					2: 102,
				}
				tool.cacheMu.Unlock()
			}
		}()
	}

	// Concurrent cache clears
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// This would race with getBackendIDFromCache reads
				tool.clearCacheForTab(sessionID, tabID)

				// Re-populate for next iteration
				tool.cacheMu.Lock()
				tool.elementCache[cacheKey] = map[int]int64{
					0: 100,
					1: 101,
					2: 102,
				}
				tool.cacheMu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Test passes if no race is detected
	// Run with: go test -race -run TestElementCacheRaceCondition
}
