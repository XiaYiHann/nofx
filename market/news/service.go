package news

import (
	"log"
	"sort"
	"sync"
	"time"
)

// Service manages news fetching and caching
type Service struct {
	sources []NewsSource
	cache   map[string][]NewsItem // category -> items
	mu      sync.RWMutex
	lastUpd time.Time
}

func NewService() *Service {
	return &Service{
		sources: []NewsSource{
			&HackerNewsSource{},
			&CryptoPanicSource{},
		},
		cache: make(map[string][]NewsItem),
	}
}

// GetNews returns news, optionally filtered by category
// It refreshes cache if older than 5 minutes
func (s *Service) GetNews(category string) ([]NewsItem, error) {
	s.mu.RLock()
	elapsed := time.Since(s.lastUpd)
	s.mu.RUnlock()

	if elapsed > 5*time.Minute {
		if err := s.refresh(); err != nil {
			log.Printf("Failed to refresh news: %v", err)
			// Return stale data if available
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []NewsItem
	if category == "" || category == "all" {
		// Combine all
		for _, items := range s.cache {
			result = append(result, items...)
		}
		// Sort by date desc
		sortNews(result)
	} else {
		result = s.cache[category]
		// Sort by date desc (already sorted per source? maybe not combined)
		sortNews(result)
	}

	return result, nil
}

func (s *Service) refresh() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double check
	if time.Since(s.lastUpd) < 5*time.Minute {
		return nil
	}

	var wg sync.WaitGroup
	newCache := make(map[string][]NewsItem)
	var errs []error
	var errMu sync.Mutex

	for _, src := range s.sources {
		wg.Add(1)
		go func(source NewsSource) {
			defer wg.Done()
			items, err := source.FetchNews()
			if err != nil {
				errMu.Lock()
				errs = append(errs, err)
				errMu.Unlock()
				return
			}

			// Group by category (assumed from item)
			errMu.Lock()
			for _, item := range items {
				newCache[item.Category] = append(newCache[item.Category], item)
			}
			errMu.Unlock()
		}(src)
	}
	wg.Wait()

	if len(errs) > 0 {
		// Log errors but if we got some data, update cache
		log.Printf("News refresh encountered errors: %v", errs)
	}

	if len(newCache) > 0 {
		s.cache = newCache
		s.lastUpd = time.Now()
	}

	return nil
}

func sortNews(items []NewsItem) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].PublishedAt.After(items[j].PublishedAt)
	})
}
