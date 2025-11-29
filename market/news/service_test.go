package news

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockNewsSource is a mock implementation of NewsSource for testing
type mockNewsSource struct {
	name  string
	items []NewsItem
	err   error
	delay time.Duration
}

func (m *mockNewsSource) Name() string {
	return m.name
}

func (m *mockNewsSource) FetchNews() ([]NewsItem, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.items, nil
}

func TestNewService(t *testing.T) {
	svc := NewService()
	require.NotNil(t, svc)
	assert.Len(t, svc.sources, 2) // HackerNews + CryptoPanic
}

func TestService_GetNews_Empty(t *testing.T) {
	svc := &Service{
		sources: []NewsSource{},
		cache:   make(map[string][]NewsItem),
		lastUpd: time.Now(), // Set to now to avoid refresh
	}

	items, err := svc.GetNews("all")
	require.NoError(t, err)
	assert.Len(t, items, 0)
}

func TestService_GetNews_WithCache(t *testing.T) {
	now := time.Now()
	cachedItems := []NewsItem{
		{ID: "1", Title: "Cached News 1", Category: "crypto", PublishedAt: now},
		{ID: "2", Title: "Cached News 2", Category: "tech", PublishedAt: now.Add(-time.Hour)},
	}

	svc := &Service{
		sources: []NewsSource{},
		cache: map[string][]NewsItem{
			"crypto": {cachedItems[0]},
			"tech":   {cachedItems[1]},
		},
		lastUpd: time.Now(), // Recent update - no refresh needed
	}

	t.Run("GetAll", func(t *testing.T) {
		items, err := svc.GetNews("all")
		require.NoError(t, err)
		assert.Len(t, items, 2)
	})

	t.Run("GetByCategory", func(t *testing.T) {
		items, err := svc.GetNews("crypto")
		require.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, "crypto", items[0].Category)
	})

	t.Run("GetEmptyCategory", func(t *testing.T) {
		items, err := svc.GetNews("nonexistent")
		require.NoError(t, err)
		assert.Len(t, items, 0)
	})
}

func TestService_GetNews_RefreshOnStaleCache(t *testing.T) {
	mockSource := &mockNewsSource{
		name: "MockSource",
		items: []NewsItem{
			{ID: "fresh-1", Title: "Fresh News", Category: "crypto", PublishedAt: time.Now()},
		},
	}

	svc := &Service{
		sources: []NewsSource{mockSource},
		cache:   make(map[string][]NewsItem),
		lastUpd: time.Now().Add(-10 * time.Minute), // Stale - should refresh
	}

	items, err := svc.GetNews("all")
	require.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "fresh-1", items[0].ID)
}

func TestService_GetNews_SortsByDate(t *testing.T) {
	now := time.Now()
	svc := &Service{
		sources: []NewsSource{},
		cache: map[string][]NewsItem{
			"all": {
				{ID: "old", Title: "Old News", PublishedAt: now.Add(-2 * time.Hour)},
				{ID: "new", Title: "New News", PublishedAt: now},
				{ID: "mid", Title: "Mid News", PublishedAt: now.Add(-time.Hour)},
			},
		},
		lastUpd: time.Now(),
	}

	items, err := svc.GetNews("all")
	require.NoError(t, err)
	require.Len(t, items, 3)
	// Should be sorted by date descending
	assert.Equal(t, "new", items[0].ID)
	assert.Equal(t, "mid", items[1].ID)
	assert.Equal(t, "old", items[2].ID)
}

func TestService_Refresh_ConcurrentSources(t *testing.T) {
	// Test that multiple sources are fetched concurrently
	source1 := &mockNewsSource{
		name:  "Source1",
		items: []NewsItem{{ID: "s1-1", Category: "crypto", PublishedAt: time.Now()}},
		delay: 50 * time.Millisecond,
	}
	source2 := &mockNewsSource{
		name:  "Source2",
		items: []NewsItem{{ID: "s2-1", Category: "tech", PublishedAt: time.Now()}},
		delay: 50 * time.Millisecond,
	}

	svc := &Service{
		sources: []NewsSource{source1, source2},
		cache:   make(map[string][]NewsItem),
		lastUpd: time.Time{}, // Force refresh
	}

	start := time.Now()
	items, err := svc.GetNews("all")
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Len(t, items, 2)
	// If sources ran concurrently, should take ~50ms, not ~100ms
	assert.Less(t, elapsed, 100*time.Millisecond)
}

func TestService_Refresh_PartialFailure(t *testing.T) {
	// One source fails, another succeeds - should still get partial results
	successSource := &mockNewsSource{
		name:  "SuccessSource",
		items: []NewsItem{{ID: "success-1", Category: "crypto", PublishedAt: time.Now()}},
	}
	failSource := &mockNewsSource{
		name: "FailSource",
		err:  assert.AnError,
	}

	svc := &Service{
		sources: []NewsSource{successSource, failSource},
		cache:   make(map[string][]NewsItem),
		lastUpd: time.Time{}, // Force refresh
	}

	items, err := svc.GetNews("all")
	require.NoError(t, err) // GetNews doesn't return error on partial failure
	assert.Len(t, items, 1)
	assert.Equal(t, "success-1", items[0].ID)
}

func TestService_Refresh_DoesNotRaceWithGets(t *testing.T) {
	// Test concurrent access doesn't cause race conditions
	mockSource := &mockNewsSource{
		name: "MockSource",
		items: []NewsItem{
			{ID: "item-1", Category: "crypto", PublishedAt: time.Now()},
		},
		delay: 10 * time.Millisecond,
	}

	svc := &Service{
		sources: []NewsSource{mockSource},
		cache:   make(map[string][]NewsItem),
		lastUpd: time.Time{}, // Force refresh
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = svc.GetNews("all")
		}()
	}
	wg.Wait()

	// Should complete without race conditions (run with -race to verify)
}
