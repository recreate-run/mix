package interfaces

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockTool implements interfaces.BaseTool for testing
type MockTool struct {
	mock.Mock
}

func (m *MockTool) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockTool) Description() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockTool) Parameters() interface{} {
	args := m.Called()
	return args.Get(0)
}

func (m *MockTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	args := m.Called(ctx, input)
	return args.Get(0), args.Error(1)
}