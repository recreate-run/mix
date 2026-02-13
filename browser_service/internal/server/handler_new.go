package server

import (
	"context"
	"encoding/json"

	"github.com/sarathmenon/browser-service/pkg/protocol"
)

// handleUploadFile uploads files to a file input element
func (h *MessageHandler) handleUploadFile(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.UploadFileParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
	}

	if len(params.FilePaths) == 0 {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "FilePaths is required"))
	}

	result, err := h.client.Context.UploadFile(ctx, params.Index, params.FilePaths, params.TabID)
	if err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeBrowserError, err.Error()))
	}

	return protocol.NewResponse(req.ID, result)
}

// handleGetText extracts text content from the page
func (h *MessageHandler) handleGetText(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.GetTextParams
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return protocol.NewErrorResponse(req.ID,
				protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
		}
	}

	result, err := h.client.Context.GetText(ctx, params.Strategy, params.TabID)
	if err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeBrowserError, err.Error()))
	}

	return protocol.NewResponse(req.ID, result)
}

// handleFind searches for elements matching a query
func (h *MessageHandler) handleFind(ctx context.Context, req protocol.Request) protocol.Response {
	var params protocol.FindParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "Invalid params"))
	}

	if params.Query == "" {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeInvalidParams, "Query is required"))
	}

	result, err := h.client.Context.Find(ctx, params.Query, params.Limit, params.TabID)
	if err != nil {
		return protocol.NewErrorResponse(req.ID,
			protocol.NewError(protocol.ErrCodeBrowserError, err.Error()))
	}

	return protocol.NewResponse(req.ID, result)
}
