package fetcher

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
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
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		retries: retries,
	}
}

func (f *HTTPFetcher) Fetch(url string) (io.ReadCloser, error) {
	var lastErr error
	for i := 0; i <= f.retries; i++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Add User-Agent to avoid being blocked by Gov.br
		req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

		resp, err := f.client.Do(req)
		if err == nil {
			if resp.StatusCode == http.StatusOK {
				return resp.Body, nil
			}
			resp.Body.Close()
			lastErr = fmt.Errorf("status code %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		
		log.Printf("Retry %d for %s: %v", i, url, lastErr)
		time.Sleep(time.Second * time.Duration(1<<i)) // Exponential backoff
	}
	return nil, fmt.Errorf("failed to fetch %s after %d retries: %w", url, f.retries, lastErr)
}
