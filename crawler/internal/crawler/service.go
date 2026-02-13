package crawler

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/user/legisgraph-crawler/internal/domain"
)

type CrawlTask struct {
	URL         string
	Parent      string
	Description string
	Depth       int
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

	// Configuration
	ignoreKeywords []string
	outputSubdir   string

	// Statistics
	statsProcessed int64
	statsFetched   int64
	statsSaved     int64
	statsSkipped   int64
}

func NewCrawlerService(fetcher domain.Fetcher, pdfParser domain.Parser, htmlParser HTMLParser, storage domain.Storage, maxDepth int, ignoreKeywords []string, outputSubdir string) *CrawlerService {
	return &CrawlerService{
		fetcher:        fetcher,
		pdfParser:      pdfParser,
		htmlParser:     htmlParser,
		storage:        storage,
		queue:          make(chan CrawlTask, 100000), // Very large buffer to avoid deadlock
		maxDepth:       maxDepth,
		ignoreKeywords: ignoreKeywords,
		outputSubdir:   outputSubdir,
	}
}

func (c *CrawlerService) Crawl(startURL string) error {
	// Start workers
	numWorkers := 10
	for i := 0; i < numWorkers; i++ {
		go c.worker()
	}

	c.enqueueTask(startURL, "", "", 0)
	
	c.wg.Wait()
	close(c.queue)

	log.Printf("------------------------------------------")
	log.Printf("CRAWLING SUMMARY:")
	log.Printf("- Tasks processed:     %d", atomic.LoadInt64(&c.statsProcessed))
	log.Printf("- Pages/Files fetched: %d", atomic.LoadInt64(&c.statsFetched))
	log.Printf("- New files saved:     %d", atomic.LoadInt64(&c.statsSaved))
	log.Printf("- Files skipped:       %d", atomic.LoadInt64(&c.statsSkipped))
	log.Printf("------------------------------------------")

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
	
	if task.Depth > c.maxDepth {
		log.Printf("Max depth reached, skipping: %s", u)
		return
	}

	// Filter out non-http links (like the file:/// links found on Planalto)
	if !strings.HasPrefix(u, "http") {
		return
	}

	log.Printf("[Depth %d] Visiting: %s", task.Depth, u)
	atomic.AddInt64(&c.statsProcessed, 1)

	// Check if file already exists before downloading
	lowerURL := strings.ToLower(u)
	parsedURL, _ := url.Parse(lowerURL)
	urlPath := parsedURL.Path

	isPDF := strings.HasSuffix(urlPath, ".pdf") || strings.Contains(lowerURL, "ext=.pdf")
	var content []byte
	var body io.ReadCloser
	var err error

	if isPDF {
		filename := c.generateFilename(u, ".pdf")
		if c.storage.Exists(filename) {
			log.Printf("File exists locally, reading for links: %s", filename)
			f, err := c.storage.Read(filename)
			if err == nil {
				defer f.Close()
				content, err = io.ReadAll(f)
				if err == nil {
					atomic.AddInt64(&c.statsSkipped, 1)
					c.handlePDF(u, task.Parent, task.Description, content, task.Depth)
					return
				}
			}
			log.Printf("Error reading local file %s, will re-download: %v", filename, err)
		}
	}

	body, err = c.fetcher.Fetch(u)
	if err != nil {
		log.Printf("Error fetching %s: %v", u, err)
		return
	}
	defer body.Close()

	content, err = io.ReadAll(body)
	if err != nil {
		log.Printf("Error reading body of %s: %v", u, err)
		return
	}
	
	atomic.AddInt64(&c.statsFetched, 1)
	
	// Determine type
	// 1. Check extension
	// 2. Ideally check Content-Type header (but fetcher hides it currently)
	// We'll rely on extension + simple detection for now.
	
	lowerURL = strings.ToLower(u)
	parsedURL, _ = url.Parse(lowerURL)
	urlPath = parsedURL.Path
	isPDF = strings.HasSuffix(urlPath, ".pdf") || strings.Contains(lowerURL, "ext=.pdf") || bytes.HasPrefix(content, []byte("%PDF-"))

	if isPDF {
		c.handlePDF(u, task.Parent, task.Description, content, task.Depth)
	} else {
		c.handleHTML(u, task.Parent, task.Description, content, task.Depth)
	}
}

func (c *CrawlerService) handlePDF(u, parent, description string, content []byte, depth int) {
	// Generate filenames
	pdfFilename := c.generateFilename(u, ".pdf")
	jsonFilename := strings.TrimSuffix(pdfFilename, ".pdf") + ".json"

	// Save PDF
	localPath, err := c.storage.Save(pdfFilename, bytes.NewReader(content))
	if err != nil {
		log.Printf("Error saving PDF %s: %v", u, err)
	} else {
		atomic.AddInt64(&c.statsSaved, 1)
	}

	// Extract Links
	readerAt := bytes.NewReader(content)
	links, err := c.pdfParser.ExtractLinks(readerAt, int64(len(content)))
	if err != nil {
		log.Printf("Error parsing PDF %s: %v", u, err)
	}

	// Save Metadata
	metadata := domain.DocumentMetadata{
		URL:            u,
		LocalPath:      localPath,
		ParentURL:      parent,
		Description:    description,
		ExtractedLinks: links,
	}
	
	metaBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err == nil {
		c.storage.Save(jsonFilename, bytes.NewReader(metaBytes))
	}

	log.Printf("Found %d potential links in PDF: %s", len(links), u)

	// Enqueue new links found in PDF (recursive)
	for _, link := range links {
		absLink := resolveRelURL(u, link)
		if absLink == "" {
			continue
		}
		
		// For PDF links, we use the link itself as text heuristic if needed, 
		// but usually we just check keywords in URL if it's not a direct PDF.
		if c.shouldVisit(absLink, "") {
			c.enqueueTask(absLink, u, "", depth+1)
		}
	}
}

func (c *CrawlerService) handleHTML(u, parent, description string, content []byte, depth int) {
	// Generate filenames
	htmlFilename := c.generateFilename(u, ".htm")
	jsonFilename := strings.TrimSuffix(htmlFilename, ".htm") + ".json"

	// Save HTML
	localPath, err := c.storage.Save(htmlFilename, bytes.NewReader(content))
	if err != nil {
		log.Printf("Error saving HTML %s: %v", u, err)
	} else {
		atomic.AddInt64(&c.statsSaved, 1)
	}

	linksMap, err := c.htmlParser.ExtractLinks(bytes.NewReader(content))
	if err != nil {
		log.Printf("Error parsing HTML %s: %v", u, err)
		return
	}

	log.Printf("Found %d potential links in HTML: %s", len(linksMap), u)
	
	var links []string
	for link, text := range linksMap {
		absLink := resolveRelURL(u, link)
		if absLink == "" {
			continue
		}
		links = append(links, absLink)

		if c.shouldVisit(absLink, text) {
			c.enqueueTask(absLink, u, text, depth+1)
		}
	}

	// Save Metadata
	metadata := domain.DocumentMetadata{
		URL:            u,
		LocalPath:      localPath,
		ParentURL:      parent,
		Description:    description,
		ExtractedLinks: links,
	}
	
	metaBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err == nil {
		c.storage.Save(jsonFilename, bytes.NewReader(metaBytes))
	}
}

func (c *CrawlerService) shouldVisit(absLink, text string) bool {
	lowerText := strings.ToLower(text)
	lowerLink := strings.ToLower(absLink)

	parsed, err := url.Parse(lowerLink)
	if err != nil {
		return false
	}
	path := parsed.Path

	// Direct PDF check (ignoring query params)
	if strings.HasSuffix(path, ".pdf") || strings.Contains(lowerLink, "ext=.pdf") {
		return true
	}

	// Navigation filtering (only HTML or URLs without extension)
	base := filepath.Base(path)
	ext := strings.ToLower(filepath.Ext(path))
	
	isHTML := ext == ".htm" || 
		      ext == ".html" || 
			  ext == ".asp" || 
			  ext == ".php" || 
			  ext == ".jsp" || 
		      !strings.Contains(base, ".") ||
		      strings.Contains(lowerLink, "ext=.htm") ||
			  strings.Contains(lowerLink, "ext=.html")

	if !isHTML {
		return false
	}

	// Permissive check for Planalto patterns
	if strings.Contains(lowerLink, "planalto.gov.br") {
		// allow any .htm / .html in planalto for discovery
		return true
	}

	// Keyword check (Invert: Skip if found)
	for _, kw := range c.ignoreKeywords {
		if strings.Contains(lowerText, kw) || strings.Contains(lowerLink, kw) {
			return false
		}
	}

	return true
}

func (c *CrawlerService) enqueueTask(u, parent, description string, depth int) {
	// Filter out non-http links
	if !strings.HasPrefix(u, "http") {
		return
	}

	// Check depth limit before enqueuing to avoid marking as visited prematurely
	if depth > c.maxDepth {
		return
	}

	// Strip fragments to avoid visiting the same page many times for different anchors
	uWithoutFragment := u
	if idx := strings.Index(u, "#"); idx != -1 {
		uWithoutFragment = u[:idx]
	}

	normalizedURL := strings.ToLower(strings.TrimRight(uWithoutFragment, "/"))
	if _, loaded := c.visited.LoadOrStore(normalizedURL, true); !loaded {
		c.wg.Add(1)
		log.Printf("Enqueuing [Depth %d]: %s", depth, uWithoutFragment)
		// Send the cleaned URL to the queue to save space and avoid duplicate fetches
		c.queue <- CrawlTask{URL: uWithoutFragment, Parent: parent, Description: description, Depth: depth}
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
	base := filepath.Base(parsed.Path)
	if base == "" || base == "." || base == "/" {
		base = "index"
	}
	
	// Create deterministic hash of the URL
	h := sha1.New()
	h.Write([]byte(u))
	hash := fmt.Sprintf("%x", h.Sum(nil))
	
	// Sanitize base name
	base = strings.ReplaceAll(base, "/", "_")
	
	// Ensure extension
	if !strings.HasSuffix(base, ext) {
		base += ext
	}
	
	filename := fmt.Sprintf("%s_%s", hash[:10], base)
	if c.outputSubdir != "" {
		return filepath.Join(c.outputSubdir, filename)
	}
	return filename
}

// Ensure http import is used or removed (it was unused in previous code but might be useful now)
// We didn't use net/http directly here yet, removing to avoid unused error.
var _ = http.Client{}


