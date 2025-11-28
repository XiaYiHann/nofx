package testhelpers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// SetupMockLLMServer creates a mock LLM server that returns a predefined response.
// returns the server.
func SetupMockLLMServer(t *testing.T, responseContent string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		resp := map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{
					"message": map[string]string{
						"content": responseContent,
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))

	return server
}

// SetupFailingMockLLMServer creates a mock LLM server that always returns 500 Internal Server Error.
func SetupFailingMockLLMServer(t *testing.T) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))

	return server
}
