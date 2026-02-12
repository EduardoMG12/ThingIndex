package crawler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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
	Depth  int
}

type HTMLParser interface {
	ExtractLinks(content io.Reader) (map[string]string, error)
}

type CrawlerService struct {
	fetcher    domain.Fetcher
	pdfParser  domain.Parser
	htmlParser HTMLParser
	storage    domain.Storage
	visited    sync.Map
	queue      chan CrawlTask
	wg         sync.WaitGroup
	maxDepth   int
}

func NewCrawlerService(fetcher domain.Fetcher, pdfParser domain.Parser, htmlParser HTMLParser, storage domain.Storage, maxDepth int) *CrawlerService {
	return &CrawlerService{
		fetcher:    fetcher,
		pdfParser:  pdfParser,
		htmlParser: htmlParser,
		storage:    storage,
		queue:      make(chan CrawlTask, 1000), // Larger buffer
		maxDepth:   maxDepth,
	}
}

func (c *CrawlerService) Crawl(startURL string) error {
	c.queue <- CrawlTask{URL: startURL, Parent: "", Depth: 0}
	c.wg.Add(1)

	// Start distinct workers to handle concurrency better
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
	
	// Normalize URL for visited check
	normalizedURL := strings.ToLower(strings.TrimRight(u, "/"))
	if _, loaded := c.visited.LoadOrStore(normalizedURL, true); loaded {
		return 
	}
	
	if task.Depth > c.maxDepth {
		log.Printf("Max depth reached, skipping: %s", u)
		return
	}

	log.Printf("[Depth %d] Visiting: %s", task.Depth, u)

	// Fetch without consuming body yet
	// We need to check Content-Type. 
	// The current Fetcher returns ReadCloser directly. 
	// We should probably peek or trust the extension/server.
	// For this implementation, we will ReadAll and determine type.
	
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
	
	// Determine type
	// 1. Check extension
	// 2. Ideally check Content-Type header (but fetcher hides it currently)
	// We'll rely on extension + simple detection for now.
	
	lowerURL := strings.ToLower(u)
	isPDF := strings.HasSuffix(lowerURL, ".pdf") || bytes.HasPrefix(content, []byte("%PDF-"))

	if isPDF {
		c.handlePDF(u, task.Parent, content, task.Depth)
	} else {
		c.handleHTML(u, task.Parent, content, task.Depth)
	}
}

func (c *CrawlerService) handlePDF(u, parent string, content []byte, depth int) {
	// Generate filenames
	pdfFilename := c.generateFilename(u, ".pdf")
	jsonFilename := strings.TrimSuffix(pdfFilename, ".pdf") + ".json"

	// Save PDF
	localPath, err := c.storage.Save(pdfFilename, bytes.NewReader(content))
	if err != nil {
		log.Printf("Error saving PDF %s: %v", u, err)
	}

	// Extract Links
	readerAt := bytes.NewReader(content)
	links, err := c.pdfParser.ExtractLinks(readerAt, int64(len(content)))
	if err != nil {
		log.Printf("Error parsing PDF %s: %v", u, err)
	}

	// Save Metadata
	metadata := domain.PDFMetadata{
		URL:            u,
		LocalPath:      localPath,
		ParentURL:      parent,
		ExtractedLinks: links,
	}
	
	metaBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err == nil {
		c.storage.Save(jsonFilename, bytes.NewReader(metaBytes))
	}

	// Enqueue new PDF links found in PDF (recursive)
	for _, link := range links {
		absLink := resolveRelURL(u, link)
		if strings.HasSuffix(strings.ToLower(absLink), ".pdf") {
			norm := strings.ToLower(strings.TrimRight(absLink, "/"))
			if _, visited := c.visited.Load(norm); !visited {
				c.wg.Add(1)
				go func(l string, p string, d int) {
					c.queue <- CrawlTask{URL: l, Parent: p, Depth: d}
				}(absLink, u, depth+1)
			}
		}
	}
}

func (c *CrawlerService) handleHTML(u, parent string, content []byte, depth int) {
	linksMap, err := c.htmlParser.ExtractLinks(bytes.NewReader(content))
	if err != nil {
		log.Printf("Error parsing HTML %s: %v", u, err)
		return
	}

	// Filter Logic for Gov.br / Receita Federal
	// Path: Legislação -> Códigos -> CTB | CDC
	
	// Heuristics:
	// 1. Follow "Legislação"
	// 2. Follow "Códigos"
	// 3. Follow "Código de Trânsito" or "Código de Defesa do Consumidor"
	// 4. Download PDFs found in step 3
	
	for link, text := range linksMap {
		absLink := resolveRelURL(u, link)
		lowerText := strings.ToLower(text)
		lowerLink := strings.ToLower(absLink)
		
		shouldVisit := false
		
		// PDF Check
		if strings.HasSuffix(lowerLink, ".pdf") {
			shouldVisit = true
		} else if strings.HasSuffix(lowerLink, ".htm") || strings.HasSuffix(lowerLink, ".html") || !strings.Contains(filepath.Base(lowerLink), ".") {
			// HTML Navigation Navigation
			if strings.Contains(lowerText, "legislação") || 
			   strings.Contains(lowerText, "legislacao") ||
			   strings.Contains(lowerText, "códigos") ||
			   strings.Contains(lowerText, "codigos") ||
			   strings.Contains(lowerText, "trânsito") ||
			   strings.Contains(lowerText, "transito") ||
			   strings.Contains(lowerText, "consumidor") {
				shouldVisit = true
				log.Printf("Found relevant HTML link: %s (Text: %s)", absLink, text)
			}
		}

		if shouldVisit {
			norm := strings.ToLower(strings.TrimRight(absLink, "/"))
			if _, visited := c.visited.Load(norm); !visited {
				c.wg.Add(1)
				go func(l string, p string, d int) {
					c.queue <- CrawlTask{URL: l, Parent: p, Depth: d}
				}(absLink, u, depth+1)
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

func (c *CrawlerService) generateFilename(u string, ext string) string {
	parsed, _ := url.Parse(u)
	filename := filepath.Base(parsed.Path)
	if filename == "" || filename == "." || filename == "/" {
		filename = "index"
	}
	
	// Sanitize
	filename = strings.ReplaceAll(filename, "/", "_")
	
	// Ensure extension
	if !strings.HasSuffix(filename, ext) {
		filename += ext
	}
	
	return fmt.Sprintf("%d_%s", time.Now().UnixNano(), filename)
}

// Ensure http import is used or removed (it was unused in previous code but might be useful now)
// We didn't use net/http directly here yet, removing to avoid unused error.
var _ = http.Client{}


