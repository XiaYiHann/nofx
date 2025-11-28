package testhelpers

import (
	"nofx/mcp"
	"testing"
)

// NewMCPClientFromMockServer creates a mock LLM server and returns an MCP client configured to use it.
// The server will return the provided responseContent in an OpenAI-compatible envelope.
func NewMCPClientFromMockServer(t *testing.T, responseContent string) *mcp.Client {
	t.Helper()

	// 1. Setup the mock server
	server := SetupMockLLMServer(t, responseContent)

	// 2. Create a new MCP client
	client := mcp.New()

	// 3. Configure the client to use the mock server
	// We use "openai" provider style configuration since SetupMockLLMServer returns OpenAI-compatible responses
	client.SetCustomAPI(server.URL, "test-key", "test-model")

	return client
}

// NewFailingMCPClientFromMockServer creates a mock LLM server that always fails (500 Internal Server Error)
// and returns an MCP client configured to use it.
func NewFailingMCPClientFromMockServer(t *testing.T) *mcp.Client {
	t.Helper()

	// 1. Setup a failing mock server
	server := SetupFailingMockLLMServer(t)

	// 2. Create a new MCP client
	client := mcp.New()

	// 3. Configure the client to use the mock server
	client.SetCustomAPI(server.URL, "test-key", "test-model")

	return client
}
