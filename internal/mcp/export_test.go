package mcp

import sdk "github.com/modelcontextprotocol/go-sdk/mcp"

// SDKServer exposes the underlying *sdk.Server to package tests so
// they can drive it through an in-memory transport. Not part of the
// public API.
func (s *Server) SDKServer() *sdk.Server { return s.sdk }
