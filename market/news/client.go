package news

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"
)

// --- Hacker News Source ---

type HackerNewsSource struct{}

func (s *HackerNewsSource) Name() string {
	return "Hacker News"
}

func (s *HackerNewsSource) FetchNews() ([]NewsItem, error) {
	// 1. Get Top Stories IDs
	resp, err := http.Get("https://hacker-news.firebaseio.com/v0/topstories.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ids []int
	if err := json.NewDecoder(resp.Body).Decode(&ids); err != nil {
		return nil, err
	}

	// Limit to top 20 for performance
	if len(ids) > 20 {
		ids = ids[:20]
	}

	var news []NewsItem
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 2. Fetch details for each story concurrently
	for _, id := range ids {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			item, err := s.fetchItem(id)
			if err == nil && item != nil {
				mu.Lock()
				news = append(news, *item)
				mu.Unlock()
			}
		}(id)
	}
	wg.Wait()

	// Sort by score (descending)
	sort.Slice(news, func(i, j int) bool {
		return news[i].Score > news[j].Score
	})

	return news, nil
}

type hnItem struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Score       int    `json:"score"`
	Time        int64  `json:"time"`
	Type        string `json:"type"`
	Descendants int    `json:"descendants"` // comment count
}

func (s *HackerNewsSource) fetchItem(id int) (*NewsItem, error) {
	url := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var hItem hnItem
	if err := json.NewDecoder(resp.Body).Decode(&hItem); err != nil {
		return nil, err
	}

	if hItem.Type != "story" || hItem.URL == "" {
		return nil, nil
	}

	return &NewsItem{
		ID:          fmt.Sprintf("hn-%d", hItem.ID),
		Title:       hItem.Title,
		Summary:     fmt.Sprintf("Points: %d | Comments: %d", hItem.Score, hItem.Descendants),
		URL:         hItem.URL,
		Source:      "Hacker News",
		Category:    "tech",
		PublishedAt: time.Unix(hItem.Time, 0),
		Score:       hItem.Score,
	}, nil
}

// --- CryptoPanic Source (RSS) ---

type CryptoPanicSource struct{}

func (s *CryptoPanicSource) Name() string {
	return "CryptoPanic"
}

type rssFeed struct {
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	PubDate     string `xml:"pubDate"`
	Description string `xml:"description"`
	GUID        string `xml:"guid"`
}

func (s *CryptoPanicSource) FetchNews() ([]NewsItem, error) {
	resp, err := http.Get("https://cryptopanic.com/news/rss/")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var feed rssFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, err
	}

	var news []NewsItem
	for _, item := range feed.Channel.Items {
		pubDate, _ := time.Parse(time.RFC1123, item.PubDate)
		
		// Simple summary extraction (could be improved)
		summary := "Crypto News"
		if len(item.Description) > 0 {
			// Strip HTML tags if necessary, for now just take a substring if too long
			if len(item.Description) > 100 {
				summary = item.Description[:97] + "..."
			} else {
				summary = item.Description
			}
		}

		news = append(news, NewsItem{
			ID:          item.GUID,
			Title:       item.Title,
			Summary:     summary,
			URL:         item.Link,
			Source:      "CryptoPanic",
			Category:    "crypto",
			PublishedAt: pubDate,
			Score:       0, // RSS doesn't provide score usually
		})
	}

	return news, nil
}
