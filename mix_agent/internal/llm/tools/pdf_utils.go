package tools

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"mix/internal/logging"
)

const (
	// MaxPDFPagesWithoutTruncation is the maximum number of pages to process
	// from a PDF when no explicit page range is specified
	MaxPDFPagesWithoutTruncation = 10
)

// ParsePageSelection parses a page selection string into individual page numbers
// Supports formats like: "5", "1-3", "1,3,5", "1-3,7,10-12"
func ParsePageSelection(pages string) ([]int, error) {
	if pages == "" {
		return nil, nil
	}

	var pageNumbers []int
	parts := strings.Split(pages, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "-") {
			// Handle range like "1-3"
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range format: %s", part)
			}

			start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid start page: %s", rangeParts[0])
			}

			end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid end page: %s", rangeParts[1])
			}

			if start > end {
				return nil, fmt.Errorf("start page cannot be greater than end page: %d > %d", start, end)
			}

			for i := start; i <= end; i++ {
				pageNumbers = append(pageNumbers, i)
			}
		} else {
			// Handle single page like "5"
			pageNum, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid page number: %s", part)
			}
			pageNumbers = append(pageNumbers, pageNum)
		}
	}

	if len(pageNumbers) == 0 {
		return nil, fmt.Errorf("no valid page numbers found")
	}

	// Remove duplicates and sort
	uniquePages := make(map[int]bool)
	var result []int
	for _, page := range pageNumbers {
		if page <= 0 {
			return nil, fmt.Errorf("page numbers must be positive: %d", page)
		}
		if !uniquePages[page] {
			uniquePages[page] = true
			result = append(result, page)
		}
	}

	return result, nil
}

// ExtractPDFPages extracts specific pages from a PDF and returns the new PDF as bytes.
// Returns: (pdfData, wasTruncated, error)
// - If pages is empty and PDF has > 10 pages: automatically extracts first 10 pages and returns wasTruncated=true
// - If pages is empty and PDF has <= 10 pages: returns original PDF with wasTruncated=false
// - If pages is specified: extracts those pages with wasTruncated=false
func ExtractPDFPages(pdfData []byte, pages string) ([]byte, bool, error) {
	// Create a reader from the PDF bytes
	reader := bytes.NewReader(pdfData)

	// Read and validate the PDF context to properly populate page count
	ctx, err := api.ReadAndValidate(reader, model.NewDefaultConfiguration())
	if err != nil {
		return nil, false, fmt.Errorf("failed to read and validate PDF: %w", err)
	}

	pageCount := ctx.PageCount
	logging.Debug("PDF context loaded", "pageCount", pageCount, "requestedPages", pages)

	// Determine which pages to extract
	var pageNumbers []int
	var wasTruncated bool

	if pages == "" {
		// No pages specified - check if auto-truncation is needed
		if pageCount > MaxPDFPagesWithoutTruncation {
			// Auto-truncate to first MaxPDFPagesWithoutTruncation pages
			pageNumbers = make([]int, MaxPDFPagesWithoutTruncation)
			for i := range MaxPDFPagesWithoutTruncation {
				pageNumbers[i] = i + 1
			}
			wasTruncated = true
		} else {
			// PDF has MaxPDFPagesWithoutTruncation or fewer pages, return original
			return pdfData, false, nil
		}
	} else {
		// Pages explicitly specified - parse the selection
		pageNumbers, err = ParsePageSelection(pages)
		if err != nil {
			return nil, false, fmt.Errorf("failed to parse page selection: %w", err)
		}
		wasTruncated = false
	}

	// Validate that requested pages exist
	for _, pageNum := range pageNumbers {
		if pageNum > pageCount {
			return nil, false, fmt.Errorf("page %d does not exist (PDF has %d pages)", pageNum, pageCount)
		}
	}

	// Extract each requested page individually
	logging.Debug("Extracting PDF pages", "pages", pageNumbers, "wasTruncated", wasTruncated)
	var pageReaders []io.ReadSeeker
	for _, pageNum := range pageNumbers {
		// Extract the page as an io.Reader
		pageReader, err := api.ExtractPage(ctx, pageNum)
		if err != nil {
			logging.Error("Failed to extract PDF page", "pageNum", pageNum, "error", err)
			return nil, false, fmt.Errorf("failed to extract page %d: %w", pageNum, err)
		}

		// Read the page data into bytes
		pageData, err := io.ReadAll(pageReader)
		if err != nil {
			logging.Error("Failed to read extracted page data", "pageNum", pageNum, "error", err)
			return nil, false, fmt.Errorf("failed to read page %d data: %w", pageNum, err)
		}

		// Convert to ReadSeeker for merging
		pageReaders = append(pageReaders, bytes.NewReader(pageData))
	}

	// Create output buffer for the merged PDF
	var outputBuffer bytes.Buffer

	// Merge all extracted pages into a single PDF
	logging.Debug("Merging extracted PDF pages", "pageCount", len(pageReaders))
	err = api.MergeRaw(pageReaders, &outputBuffer, false, model.NewDefaultConfiguration())
	if err != nil {
		logging.Error("Failed to merge extracted PDF pages", "error", err)
		return nil, false, fmt.Errorf("failed to merge extracted pages: %w", err)
	}

	logging.Debug("PDF pages extracted successfully", "outputSize", outputBuffer.Len(), "wasTruncated", wasTruncated)
	return outputBuffer.Bytes(), wasTruncated, nil
}

// ValidatePageSelection validates a page selection string format
func ValidatePageSelection(pages string) error {
	if pages == "" {
		return nil
	}

	_, err := ParsePageSelection(pages)
	return err
}