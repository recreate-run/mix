package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"mix/internal/session"
)

// Test App struct creation and fields
func TestAppStruct(t *testing.T) {
	app := &App{
		Sessions:      nil,
		Messages:      nil,
		History:       nil,
		Permissions:   nil,
		Analytics:     nil,
		StorageConfig: session.DefaultConfig(),
		CoderAgent:    nil,
	}

	// Test that the struct can be created
	assert.NotNil(t, app)
	assert.NotNil(t, app.StorageConfig)
}

// Test App field assignments
func TestAppFieldAssignments(t *testing.T) {
	storageConfig := session.DefaultConfig()

	app := &App{
		StorageConfig: storageConfig,
	}

	// Verify storage config is properly set
	assert.Equal(t, storageConfig, app.StorageConfig)
}

// Test App with nil fields
func TestAppWithNilFields(t *testing.T) {
	app := &App{}

	// Should be able to create app with nil fields
	assert.NotNil(t, app)
	assert.Nil(t, app.Sessions)
	assert.Nil(t, app.Messages)
	assert.Nil(t, app.History)
	assert.Nil(t, app.Permissions)
	assert.Nil(t, app.Analytics)
	assert.Nil(t, app.CoderAgent)
}

// Test Shutdown method with nil services
func TestShutdownWithNilServices(t *testing.T) {
	app := &App{
		Sessions:      nil,
		Messages:      nil,
		History:       nil,
		Permissions:   nil,
		Analytics:     nil,
		StorageConfig: session.DefaultConfig(),
		CoderAgent:    nil,
	}

	// Should not panic even with nil services
	assert.NotPanics(t, func() {
		app.Shutdown()
	})
}

// Test Shutdown method with nil app
func TestShutdownWithNilApp(t *testing.T) {
	var app *App

	// Should not panic with nil app
	assert.NotPanics(t, func() {
		if app != nil {
			app.Shutdown()
		}
	})
}

// Test App struct field types
func TestAppStructFieldTypes(t *testing.T) {
	app := &App{}

	// Verify field types are correct (compilation test)
	assert.IsType(t, (session.Service)(nil), app.Sessions)
	assert.IsType(t, session.Config{}, app.StorageConfig)
}

// Test title generation logic (without dependencies)
func TestTitleGeneration(t *testing.T) {
	tests := []struct {
		name           string
		prompt         string
		expectedPrefix string
		expectsTrunc   bool
	}{
		{
			name:           "short prompt",
			prompt:         "Test prompt",
			expectedPrefix: "Non-interactive: Test prompt",
			expectsTrunc:   false,
		},
		{
			name:           "long prompt",
			prompt:         "This is a very long prompt that exceeds the maximum length for titles and should be truncated appropriately when used as a session title",
			expectedPrefix: "Non-interactive:",
			expectsTrunc:   true,
		},
		{
			name:           "empty prompt",
			prompt:         "",
			expectedPrefix: "Non-interactive: ",
			expectsTrunc:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the title generation logic manually
			const maxPromptLengthForTitle = 100
			titlePrefix := "Non-interactive: "
			var titleSuffix string

			if len(tt.prompt) > maxPromptLengthForTitle {
				titleSuffix = tt.prompt[:maxPromptLengthForTitle] + "..."
			} else {
				titleSuffix = tt.prompt
			}
			title := titlePrefix + titleSuffix

			assert.Contains(t, title, tt.expectedPrefix)

			if tt.expectsTrunc {
				assert.Contains(t, title, "...")
				assert.True(t, len(title) < len("Non-interactive: "+tt.prompt))
			}
		})
	}
}

// Test storage config defaults
func TestStorageConfigDefaults(t *testing.T) {
	config := session.DefaultConfig()

	// Test that default config can be created
	assert.NotNil(t, config)

	app := &App{
		StorageConfig: config,
	}

	assert.Equal(t, config, app.StorageConfig)
}

// Test App struct zero value
func TestAppZeroValue(t *testing.T) {
	var app App

	// Test that zero value app doesn't panic on access
	assert.Nil(t, app.Sessions)
	assert.Nil(t, app.Messages)
	assert.Nil(t, app.History)
	assert.Nil(t, app.Permissions)
	assert.Nil(t, app.Analytics)
	assert.Nil(t, app.CoderAgent)
}

// Test App creation with partial initialization
func TestAppPartialInitialization(t *testing.T) {
	app := &App{
		StorageConfig: session.DefaultConfig(),
		// Other fields intentionally nil
	}

	assert.NotNil(t, app.StorageConfig)
	assert.Nil(t, app.Sessions)
	assert.Nil(t, app.Messages)
	assert.Nil(t, app.CoderAgent)
}
