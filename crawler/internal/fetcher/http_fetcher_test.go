package fetcher

import (
	"context"
	"testing"
	"time"
)

func TestNewHTTPFetcher(t *testing.T) {
	f := NewHTTPFetcher(10*time.Second, 3)
	if f == nil {
		t.Fatal("NewHTTPFetcher() returned nil")
	}
}

func TestFetchInvalidURL(t *testing.T) {
	f := NewHTTPFetcher(2*time.Second, 0)

	_, err := f.Fetch(context.Background(), "://invalid-url")
	if err == nil {
		t.Error("Expected error for invalid URL, but got nil")
	}
}

func TestFetchNonExistentHost(t *testing.T) {
	f := NewHTTPFetcher(2*time.Second, 0)

	_, err := f.Fetch(context.Background(), "http://this-host-does-not-exist-xyz123.com")
	if err == nil {
		t.Error("Expected error for non-existent host, but got nil")
	}
}
