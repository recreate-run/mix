package stdio

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"mix/internal/api"
)

// HandleJSONRPC is the main entry point for handling JSON-RPC requests from stdin
func HandleJSONRPC(ctx context.Context, handler *api.QueryHandler, outputFormat string) error {
	return handleJSONRPCFromStdin(ctx, handler)
}

// hasStdinData checks if stdin has data available without blocking
func hasStdinData() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	// Check if stdin is a pipe/file (has data) or if it's coming from terminal
	return (stat.Mode()&os.ModeCharDevice) == 0 && stat.Size() > 0
}

func handleJSONRPCFromStdin(ctx context.Context, handler *api.QueryHandler) error {
	// Check if stdin has data before trying to read
	if !hasStdinData() {
		return fmt.Errorf(`no JSON-RPC input provided

Usage examples:
  echo '{"method": "sessions.list", "id": 1}' | %s --query json --output-format json
  echo '{"method": "sessions.create", "params": {"title": "New Session"}, "id": 1}' | %s --query json --output-format json
  
Available methods: sessions.list, sessions.create, sessions.select, sessions.delete, tools.list, mcp.list, commands.list`,
			os.Args[0], os.Args[0])
	}

	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse JSON-RPC request
		var request api.QueryRequest
		if err := json.Unmarshal([]byte(line), &request); err != nil {
			// Output error response
			errorResponse := &api.QueryResponse{
				Error: &api.QueryError{
					Code:    -32700,
					Message: "Parse error: " + err.Error(),
				},
				ID: nil,
			}
			outputJSONRPCResponse(errorResponse)
			continue
		}

		// Handle the request
		response := handler.Handle(ctx, &request)
		outputJSONRPCResponse(response)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stdin: %w", err)
	}

	return nil
}

func outputJSONRPCResponse(response *api.QueryResponse) {
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		// Fallback error response
		fallbackResponse := &api.QueryResponse{
			Error: &api.QueryError{
				Code:    -32603,
				Message: "Internal error: " + err.Error(),
			},
			ID: response.ID,
		}
		jsonBytes, _ = json.Marshal(fallbackResponse)
	}

	fmt.Println(string(jsonBytes))
}
