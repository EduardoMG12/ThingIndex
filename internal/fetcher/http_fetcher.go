package fetcher

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/user/legisgraph-crawler/internal/domain"
)

type HTTPFetcher struct {
	client  *http.Client
	retries int
}

func NewHTTPFetcher(timeout time.Duration, retries int) domain.Fetcher {
	return &HTTPFetcher{
		client: &http.Client{
			Timeout: timeout,
		},
		retries: retries,
	}
}

func (f *HTTPFetcher) Fetch(url string) (io.ReadCloser, error) {
	var err error
	for i := 0; i <= f.retries; i++ {
		resp, err := f.client.Get(url)
		if err == nil {
			if resp.StatusCode == http.StatusOK {
				return resp.Body, nil
			}
			resp.Body.Close()
			err = fmt.Errorf("status code %d", resp.StatusCode)
		}
		time.Sleep(time.Second * time.Duration(1<<i)) // Exponential backoff
	}
	return nil, fmt.Errorf("failed to fetch %s after %d retries: %w", url, f.retries, err)
}
