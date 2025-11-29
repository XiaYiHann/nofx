package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// handleGetNews returns the latest news
func (s *Server) handleGetNews(c *gin.Context) {
	category := c.Query("category")
	
	// Use the global news service (initialized in Server or singleton)
	// For now, let's assume we attach it to Server struct or use a singleton
	// Since we didn't modify Server struct yet, let's modify Server struct to hold NewsService
	// But first, let's just use a package level variable or create a new service if it's stateless enough (it has cache, so needs to be stateful)
	
	// Better approach: Add NewsService to Server struct.
	// For this step, I will assume s.newsService exists and I will update Server struct in next step.
	
	items, err := s.newsService.GetNews(category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch news"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"news": items,
	})
}
