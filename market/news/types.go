package news

import "time"

// NewsItem represents a single news article
type NewsItem struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Summary     string    `json:"summary"`
	URL         string    `json:"url"`
	Source      string    `json:"source"`      // e.g., "CryptoPanic", "Hacker News"
	Category    string    `json:"category"`    // e.g., "crypto", "tech"
	PublishedAt time.Time `json:"published_at"`
	Score       int       `json:"score"`       // Relevance score or upvotes
}

// NewsSource defines the interface for fetching news
type NewsSource interface {
	FetchNews() ([]NewsItem, error)
	Name() string
}
