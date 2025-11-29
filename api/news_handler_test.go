package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nofx/market/news"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockNewsService implements NewsServiceInterface for testing
type mockNewsService struct {
	items []news.NewsItem
	err   error
}

func (m *mockNewsService) GetNews(category string) ([]news.NewsItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	if category == "" || category == "all" {
		return m.items, nil
	}
	// Filter by category
	var filtered []news.NewsItem
	for _, item := range m.items {
		if item.Category == category {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

// Ensure mockNewsService implements NewsServiceInterface
var _ NewsServiceInterface = (*mockNewsService)(nil)

func TestHandleGetNews_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup mock service with test data
	mockService := &mockNewsService{
		items: []news.NewsItem{
			{
				ID:          "test-1",
				Title:       "Test News 1",
				Summary:     "Summary 1",
				URL:         "https://example.com/1",
				Source:      "TestSource",
				Category:    "crypto",
				PublishedAt: time.Now(),
				Score:       100,
			},
			{
				ID:          "test-2",
				Title:       "Test News 2",
				Summary:     "Summary 2",
				URL:         "https://example.com/2",
				Source:      "TestSource",
				Category:    "tech",
				PublishedAt: time.Now(),
				Score:       50,
			},
		},
	}

	// Create a test server with mock
	server := &Server{
		newsService: mockService,
	}

	router := gin.New()
	router.GET("/api/news", server.handleGetNews)

	// Test: Get all news
	t.Run("GetAllNews", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/news", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response struct {
			News []news.NewsItem `json:"news"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response.News, 2)
	})

	// Test: Get news by category
	t.Run("GetNewsByCategory", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/news?category=crypto", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response struct {
			News []news.NewsItem `json:"news"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response.News, 1)
		assert.Equal(t, "crypto", response.News[0].Category)
	})

	// Test: Verify JSON structure matches frontend expectations
	t.Run("VerifyJSONStructure", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/news?category=all", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Parse as raw JSON to verify field names
		var rawResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &rawResponse)
		require.NoError(t, err)

		// Verify "news" key exists
		newsArray, ok := rawResponse["news"].([]interface{})
		require.True(t, ok, "response should have 'news' array")
		require.Greater(t, len(newsArray), 0)

		// Verify first item has expected fields
		firstItem := newsArray[0].(map[string]interface{})
		assert.Contains(t, firstItem, "id")
		assert.Contains(t, firstItem, "title")
		assert.Contains(t, firstItem, "summary")
		assert.Contains(t, firstItem, "url")
		assert.Contains(t, firstItem, "source")
		assert.Contains(t, firstItem, "category")
		assert.Contains(t, firstItem, "published_at")
		assert.Contains(t, firstItem, "score")
	})
}

func TestHandleGetNews_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup mock service that returns error
	mockService := &mockNewsService{
		err: assert.AnError,
	}

	server := &Server{
		newsService: mockService,
	}

	router := gin.New()
	router.GET("/api/news", server.handleGetNews)

	req := httptest.NewRequest(http.MethodGet, "/api/news", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response, "error")
}

func TestHandleGetNews_EmptyResult(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup mock service with empty result
	mockService := &mockNewsService{
		items: []news.NewsItem{},
	}

	server := &Server{
		newsService: mockService,
	}

	router := gin.New()
	router.GET("/api/news", server.handleGetNews)

	req := httptest.NewRequest(http.MethodGet, "/api/news", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response struct {
		News []news.NewsItem `json:"news"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Len(t, response.News, 0)
}
