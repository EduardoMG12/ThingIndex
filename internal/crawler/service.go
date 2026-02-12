package crawler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/user/legisgraph-crawler/internal/domain"
)

type CrawlTask struct {
	URL    string
	Parent string
}

type CrawlerService struct {
	fetcher domain.Fetcher
	parser  domain.Parser
	storage domain.Storage
	visited sync.Map
	queue   chan CrawlTask
	wg      sync.WaitGroup
}

func NewCrawlerService(fetcher domain.Fetcher, parser domain.Parser, storage domain.Storage) *CrawlerService {
	return &CrawlerService{
		fetcher: fetcher,
		parser:  parser,
		storage: storage,
		queue:   make(chan CrawlTask, 100),
	}
}

func (c *CrawlerService) Crawl(startURL string) error {
	c.queue <- CrawlTask{URL: startURL, Parent: ""}
	c.wg.Add(1)

	go c.worker()
	
	c.wg.Wait()
	return nil
}

func (c *CrawlerService) worker() {
	for task := range c.queue {
		c.processTask(task)
		c.wg.Done()
	}
}

func (c *CrawlerService) processTask(task CrawlTask) {
	u := task.URL
	if _, loaded := c.visited.LoadOrStore(u, true); loaded {
		return 
	}
	
	log.Printf("Visiting: %s (Parent: %s)", u, task.Parent)

	body, err := c.fetcher.Fetch(u)
	if err != nil {
		log.Printf("Error fetching %s: %v", u, err)
		return
	}
	defer body.Close()

	content, err := io.ReadAll(body)
	if err != nil {
		log.Printf("Error reading body of %s: %v", u, err)
		return
	}
	
	// Generate filenames
	pdfFilename := c.generateFilename(u)
	jsonFilename := strings.TrimSuffix(pdfFilename, ".pdf") + ".json"

	// Save PDF
	localPath, err := c.storage.Save(pdfFilename, bytes.NewReader(content))
	if err != nil {
		log.Printf("Error saving %s: %v", u, err)
	}

	// Extract Links
	readerAt := bytes.NewReader(content)
	links, err := c.parser.ExtractLinks(readerAt, int64(len(content)))
	if err != nil {
		log.Printf("Error parsing %s: %v", u, err)
		// We continue even if parsing fails, to save metadata at least
	}

	// Save Metadata
	metadata := domain.PDFMetadata{
		URL:            u,
		LocalPath:      localPath,
		ParentURL:      task.Parent,
		ExtractedLinks: links,
	}
	
	metaBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err == nil {
		if _, err := c.storage.Save(jsonFilename, bytes.NewReader(metaBytes)); err != nil {
			log.Printf("Error saving metadata for %s: %v", u, err)
		}
	} else {
		log.Printf("Error marshaling metadata for %s: %v", u, err)
	}

	// Enqueue new links
	for _, link := range links {
		absLink := resolveRelURL(u, link)
		if strings.HasSuffix(strings.ToLower(absLink), ".pdf") {
			if _, visited := c.visited.Load(absLink); !visited {
				c.wg.Add(1)
				go func(l string, p string) {
					c.queue <- CrawlTask{URL: l, Parent: p}
				}(absLink, u)
			}
		}
	}
}

func resolveRelURL(baseURL, href string) string {
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	return base.ResolveReference(u).String()
}

func (c *CrawlerService) generateFilename(u string) string {
	parsed, _ := url.Parse(u)
	filename := filepath.Base(parsed.Path)
	if filename == "" || filename == "." {
		filename = "index.pdf"
	}
	// Use timestamp to ensure uniqueness, but keep extension
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	if ext == "" {
		ext = ".pdf"
	}
	return fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), name, ext)
}


