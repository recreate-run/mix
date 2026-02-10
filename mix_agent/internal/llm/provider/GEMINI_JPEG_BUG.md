# Gemini Vision API - JPEG Bounding Box Bug

## Critical Finding: JPEG Coordinate Transposition

**Always use PNG format for bounding box detection with Gemini Vision API.**

### Issue Summary

Gemini's vision model has a reproducible bug when processing JPEG images for bounding box detection. The coordinate system gets transposed, resulting in catastrophic accuracy loss.

### Test Results (100% Reproducible)

| Format | Coordinates | Avg Error | Pattern |
|--------|-------------|-----------|---------|
| **PNG** | `[397, 37, 413, 105]` | **1.75 units** ✅ | Correct |
| **JPEG** | `[37, 398, 104, 415]` | **336.75 units** ❌ | Transposed (y,x,y,x) |

**Ground truth:** `[398, 37, 418, 104]`

### Key Observations

- JPEG coordinates appear swapped: `(y₁, x₁, y₂, x₂)` instead of `(x₁, y₁, x₂, y₂)`
- Image conversion verified correct (no rotation, no EXIF metadata, identical dimensions)
- Tested with temperature=0 for deterministic results
- Three independent test runs produced identical transposed coordinates

### Recommendation

**Production Impact:** Browser automation screenshots must use PNG format. JPEG will cause bounding box detection to fail completely.

---
*Tested: 2026-02-10 | Model: gemini-3-flash | Image: 2426×1916px*
