//go:build tools
// +build tools

// Package tools tracks tool dependencies for `go mod` to manage.
// This ensures all team members use consistent tool versions.
//
// To install the tools, run:
//   go install github.com/sqlc-dev/sqlc/cmd/sqlc
package tools

import (
	_ "github.com/sqlc-dev/sqlc/cmd/sqlc"
)
