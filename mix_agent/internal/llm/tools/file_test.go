package tools

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Helper function to clear file records for test isolation
func clearFileRecords() {
	fileRecordMutex.Lock()
	defer fileRecordMutex.Unlock()
	fileRecords = make(map[string]fileRecord)
}

// Test recordFileRead function
func TestRecordFileRead(t *testing.T) {
	clearFileRecords()

	testPath := "/test/path/file.txt"
	beforeTime := time.Now()

	// Record a file read
	recordFileRead(testPath)

	afterTime := time.Now()

	// Verify the record was created
	fileRecordMutex.RLock()
	record, exists := fileRecords[testPath]
	fileRecordMutex.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, testPath, record.path)
	assert.True(t, record.readTime.After(beforeTime) || record.readTime.Equal(beforeTime))
	assert.True(t, record.readTime.Before(afterTime) || record.readTime.Equal(afterTime))
	assert.True(t, record.writeTime.IsZero()) // Write time should not be set
}

// Test recordFileRead with existing record
func TestRecordFileRead_ExistingRecord(t *testing.T) {
	clearFileRecords()

	testPath := "/test/path/existing.txt"

	// First read
	recordFileRead(testPath)

	fileRecordMutex.RLock()
	firstRead := fileRecords[testPath].readTime
	fileRecordMutex.RUnlock()

	// Wait a small amount to ensure time difference
	time.Sleep(1 * time.Millisecond)

	// Second read
	recordFileRead(testPath)

	fileRecordMutex.RLock()
	record := fileRecords[testPath]
	fileRecordMutex.RUnlock()

	assert.Equal(t, testPath, record.path)
	assert.True(t, record.readTime.After(firstRead)) // Should be updated
}

// Test getLastReadTime function
func TestGetLastReadTime(t *testing.T) {
	clearFileRecords()

	testPath := "/test/path/read_time.txt"

	// Test non-existent file
	readTime := getLastReadTime(testPath)
	assert.True(t, readTime.IsZero())

	// Record a read and test retrieval
	beforeTime := time.Now()
	recordFileRead(testPath)
	afterTime := time.Now()

	readTime = getLastReadTime(testPath)
	assert.False(t, readTime.IsZero())
	assert.True(t, readTime.After(beforeTime) || readTime.Equal(beforeTime))
	assert.True(t, readTime.Before(afterTime) || readTime.Equal(afterTime))
}

// Test recordFileWrite function
func TestRecordFileWrite(t *testing.T) {
	clearFileRecords()

	testPath := "/test/path/write.txt"
	beforeTime := time.Now()

	// Record a file write
	recordFileWrite(testPath)

	afterTime := time.Now()

	// Verify the record was created
	fileRecordMutex.RLock()
	record, exists := fileRecords[testPath]
	fileRecordMutex.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, testPath, record.path)
	assert.True(t, record.writeTime.After(beforeTime) || record.writeTime.Equal(beforeTime))
	assert.True(t, record.writeTime.Before(afterTime) || record.writeTime.Equal(afterTime))
	assert.True(t, record.readTime.IsZero()) // Read time should not be set
}

// Test recordFileWrite with existing record
func TestRecordFileWrite_ExistingRecord(t *testing.T) {
	clearFileRecords()

	testPath := "/test/path/existing_write.txt"

	// First write
	recordFileWrite(testPath)

	fileRecordMutex.RLock()
	firstWrite := fileRecords[testPath].writeTime
	fileRecordMutex.RUnlock()

	// Wait a small amount to ensure time difference
	time.Sleep(1 * time.Millisecond)

	// Second write
	recordFileWrite(testPath)

	fileRecordMutex.RLock()
	record := fileRecords[testPath]
	fileRecordMutex.RUnlock()

	assert.Equal(t, testPath, record.path)
	assert.True(t, record.writeTime.After(firstWrite)) // Should be updated
}

// Test mixed read and write operations
func TestMixedReadWriteOperations(t *testing.T) {
	clearFileRecords()

	testPath := "/test/path/mixed.txt"

	// Record a read first
	recordFileRead(testPath)

	fileRecordMutex.RLock()
	readTime := fileRecords[testPath].readTime
	writeTimeBefore := fileRecords[testPath].writeTime
	fileRecordMutex.RUnlock()

	assert.False(t, readTime.IsZero())
	assert.True(t, writeTimeBefore.IsZero())

	// Wait a small amount to ensure time difference
	time.Sleep(1 * time.Millisecond)

	// Now record a write
	recordFileWrite(testPath)

	fileRecordMutex.RLock()
	record := fileRecords[testPath]
	fileRecordMutex.RUnlock()

	assert.Equal(t, testPath, record.path)
	assert.Equal(t, readTime, record.readTime)       // Read time should be preserved
	assert.False(t, record.writeTime.IsZero())       // Write time should now be set
	assert.True(t, record.writeTime.After(readTime)) // Write should be after read
}

// Test thread safety with concurrent operations
func TestConcurrentFileOperations(t *testing.T) {
	clearFileRecords()

	testPath := "/test/path/concurrent.txt"
	numGoroutines := 10
	operationsPerGoroutine := 100

	var wg sync.WaitGroup

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				recordFileRead(testPath)
				getLastReadTime(testPath)
			}
		}()
	}

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				recordFileWrite(testPath)
			}
		}()
	}

	wg.Wait()

	// Verify final state is consistent
	fileRecordMutex.RLock()
	record, exists := fileRecords[testPath]
	fileRecordMutex.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, testPath, record.path)
	assert.False(t, record.readTime.IsZero())
	assert.False(t, record.writeTime.IsZero())
}

// Test multiple different files
func TestMultipleFiles(t *testing.T) {
	clearFileRecords()

	files := []string{
		"/test/file1.txt",
		"/test/file2.go",
		"/test/dir/file3.md",
		"/another/path/file4.json",
	}

	// Record reads for all files
	for i, file := range files {
		recordFileRead(file)
		if i%2 == 0 {
			recordFileWrite(file) // Write to every other file
		}
	}

	// Verify all records
	fileRecordMutex.RLock()
	defer fileRecordMutex.RUnlock()

	assert.Len(t, fileRecords, len(files))

	for i, file := range files {
		record, exists := fileRecords[file]
		assert.True(t, exists, "File %s should have a record", file)
		assert.Equal(t, file, record.path)
		assert.False(t, record.readTime.IsZero(), "File %s should have read time", file)

		if i%2 == 0 {
			assert.False(t, record.writeTime.IsZero(), "File %s should have write time", file)
		} else {
			assert.True(t, record.writeTime.IsZero(), "File %s should not have write time", file)
		}
	}
}

// Test fileRecord struct
func TestFileRecord(t *testing.T) {
	now := time.Now()
	record := fileRecord{
		path:      "/test/path.txt",
		readTime:  now,
		writeTime: now.Add(time.Second),
	}

	assert.Equal(t, "/test/path.txt", record.path)
	assert.Equal(t, now, record.readTime)
	assert.Equal(t, now.Add(time.Second), record.writeTime)
}

// Test with edge case paths
func TestEdgeCasePaths(t *testing.T) {
	clearFileRecords()

	edgeCases := []string{
		"",                           // empty path
		"/",                          // root path
		"relative/path.txt",          // relative path
		"/path/with spaces/file.txt", // path with spaces
		"/path/with/unicode/文件.txt",  // unicode filename
		"/very/long/path/that/goes/deep/into/nested/directories/file.txt", // long path
	}

	for _, path := range edgeCases {
		t.Run("path_"+path, func(t *testing.T) {
			// Should not panic or error
			recordFileRead(path)
			readTime := getLastReadTime(path)
			assert.False(t, readTime.IsZero())

			recordFileWrite(path)

			fileRecordMutex.RLock()
			record, exists := fileRecords[path]
			fileRecordMutex.RUnlock()

			assert.True(t, exists)
			assert.Equal(t, path, record.path)
			assert.False(t, record.readTime.IsZero())
			assert.False(t, record.writeTime.IsZero())
		})
	}
}

// Test time precision
func TestTimePrecision(t *testing.T) {
	clearFileRecords()

	testPath := "/test/precision.txt"

	// Record first operation
	recordFileRead(testPath)
	firstTime := getLastReadTime(testPath)

	// Very small sleep to test precision
	time.Sleep(100 * time.Microsecond)

	// Record second operation
	recordFileRead(testPath)
	secondTime := getLastReadTime(testPath)

	// Second time should be after first time (or equal if system doesn't have enough precision)
	assert.True(t, secondTime.After(firstTime) || secondTime.Equal(firstTime))
}
