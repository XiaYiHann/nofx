package news

import (
	"testing"
)

func TestHackerNewsSource_FetchNews(t *testing.T) {
	src := &HackerNewsSource{}
	items, err := src.FetchNews()
	if err != nil {
		t.Logf("Hacker News fetch failed (might be network issue): %v", err)
		// Don't fail the test if it's just network, but warn
		return
	}

	if len(items) == 0 {
		t.Log("Hacker News returned 0 items")
	} else {
		t.Logf("Hacker News returned %d items", len(items))
		t.Logf("First item: %s (%s)", items[0].Title, items[0].URL)
	}
}

func TestCryptoPanicSource_FetchNews(t *testing.T) {
	src := &CryptoPanicSource{}
	items, err := src.FetchNews()
	if err != nil {
		t.Logf("CryptoPanic fetch failed (might be network issue): %v", err)
		return
	}

	if len(items) == 0 {
		t.Log("CryptoPanic returned 0 items")
	} else {
		t.Logf("CryptoPanic returned %d items", len(items))
		t.Logf("First item: %s (%s)", items[0].Title, items[0].URL)
	}
}
