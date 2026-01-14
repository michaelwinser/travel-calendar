// Package mcp implements the Model Context Protocol (MCP) handler for the Travel Calendar.
//
// This package provides:
// - JSON-RPC 2.0 handler for MCP protocol
// - Tool definitions generated from OpenAPI spec (tools.gen.go)
// - Tool handlers that call the service layer directly
// - Markdown formatters for LLM-friendly output
//
//go:generate go run ../../cmd/mcp-codegen ../../../api/openapi.yaml tools.gen.go
package mcp
