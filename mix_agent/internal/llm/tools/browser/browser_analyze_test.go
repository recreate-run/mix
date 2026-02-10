package browser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatBoundingBoxResponse(t *testing.T) {
	// Use 1000x1000 image for 1:1 scaling (keeps test expectations simple)
	imageWidth := 1000
	imageHeight := 1000

	tests := []struct {
		name           string
		input          string
		expectedOutput string
		expectError    bool
	}{
		{
			name:  "Empty array",
			input: `[]`,
			expectedOutput: `Found 0 element(s).

No elements detected in screenshot.`,
			expectError: false,
		},
		{
			name: "Single element with name",
			input: `[
				{"box_2d": [398, 37, 418, 104], "name": "button"}
			]`,
			expectedOutput: `Found 1 element(s):

- button (x=408,y=70)
`,
			expectError: false,
		},
		{
			name: "Multiple elements",
			input: `[
				{"box_2d": [398, 37, 418, 104], "name": "button"},
				{"box_2d": [140, 20, 160, 50], "name": "link"}
			]`,
			expectedOutput: `Found 2 element(s):

- button (x=408,y=70)
- link (x=150,y=35)
`,
			expectError: false,
		},
		{
			name: "Element without name (defaults to 'element')",
			input: `[
				{"box_2d": [100, 100, 200, 200]}
			]`,
			expectedOutput: `Found 1 element(s):

- element (x=150,y=150)
`,
			expectError: false,
		},
		{
			name:        "Invalid JSON",
			input:       `invalid json`,
			expectError: true,
		},
		{
			name:        "Missing box_2d",
			input:       `[{"name": "button"}]`,
			expectError: true,
		},
		{
			name:        "Invalid box_2d format",
			input:       `[{"box_2d": [1, 2, 3]}]`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formatBoundingBoxResponse(tt.input, imageWidth, imageHeight)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedOutput, result)
			}
		})
	}
}

func TestFormatBoundingBoxResponse_CenterCalculation(t *testing.T) {
	// Test precise center calculation with 1:1 image dimensions
	imageWidth := 1000
	imageHeight := 1000

	input := `[{"box_2d": [398, 37, 418, 104]}]`
	result, err := formatBoundingBoxResponse(input, imageWidth, imageHeight)

	require.NoError(t, err)
	require.Contains(t, result, "(x=408,y=70)", "Center should be calculated as (398+418)/2=408, (37+104)/2=70.5≈70")
}

func TestFormatBoundingBoxResponse_CoordinateScaling(t *testing.T) {
	// Test coordinate scaling with different image sizes (e.g., 2x Retina display)
	imageWidth := 1920  // 1.92x scaling
	imageHeight := 1080 // 1.08x scaling

	// Normalized coordinates: center at (500, 500)
	input := `[{"box_2d": [400, 400, 600, 600], "name": "test"}]`
	result, err := formatBoundingBoxResponse(input, imageWidth, imageHeight)

	require.NoError(t, err)
	// Expected: centerX = 500 * (1920/1000) = 960, centerY = 500 * (1080/1000) = 540
	require.Contains(t, result, "(x=960,y=540)", "Coordinates should be scaled from normalized [0,1000] to actual image pixel space")
}
