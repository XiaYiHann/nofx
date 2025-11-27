package testhelpers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

// PerformRequest performs a request to a http.Handler (e.g. Gin router)
func PerformRequest(r http.Handler, method, path string, body interface{}) *httptest.ResponseRecorder {
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}
	req, _ := http.NewRequest(method, path, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
