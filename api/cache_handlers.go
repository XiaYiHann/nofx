package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// handleGetCacheStats 获取缓存统计信息
func (s *Server) handleGetCacheStats(c *gin.Context) {
	stats, err := s.database.GetCacheStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get cache stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// handleClearCache 清理缓存
func (s *Server) handleClearCache(c *gin.Context) {
	symbol := c.Query("symbol")
	olderThanDays := c.DefaultQuery("older_than_days", "")

	var olderThan time.Time
	if olderThanDays != "" {
		// 解析天数
		days := 30 // 默认30天
		if _, err := time.ParseDuration(olderThanDays + "h"); err == nil {
			olderThan = time.Now().AddDate(0, 0, -days)
		}
	}

	err := s.database.ClearCache(symbol, olderThan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear cache"})
		return
	}

	message := "All cache cleared"
	if symbol != "" {
		message = "Cache cleared for symbol: " + symbol
	} else if !olderThan.IsZero() {
		message = "Old cache cleared"
	}

	c.JSON(http.StatusOK, gin.H{"message": message})
}
