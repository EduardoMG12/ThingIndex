package crawler

import (
	"bytes"
	"context"
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



type HTMLParser interface {
	ExtractLinks(content io.Reader) (map[string]string, error)
	ExtractLinksWithContext(content io.Reader) ([]domain.EnrichedLink, error)
}

type CrawlerService struct {
	fetcher    domain.Fetcher
	pdfParser  domain.Parser
	htmlParser HTMLParser
	storage    domain.Storage
	visited    *BloomFilter
	queue      domain.Queue
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

func NewCrawlerService(fetcher domain.Fetcher, pdfParser domain.Parser, htmlParser HTMLParser, storage domain.Storage, queue domain.Queue, maxDepth int, ignoreKeywords []string, outputSubdir string) *CrawlerService {
	// Initialize Bloom Filter with 1M items and 0.01 false positive rate
	bf := NewBloomFilter(1000000, 0.01)

	return &CrawlerService{
		fetcher:        fetcher,
		pdfParser:      pdfParser,
		htmlParser:     htmlParser,
		storage:        storage,
		queue:          queue,
		visited:        bf,
		maxDepth:       maxDepth,
		ignoreKeywords: ignoreKeywords,
		outputSubdir:   outputSubdir,
	}
}

func (c *CrawlerService) Crawl(startURL string) error {
	// Consumer loop
	tasks, err := c.queue.Consume()
	if err != nil {
		return fmt.Errorf("failed to start consumer: %w", err)
	}

	// Start workers
	numWorkers := 10
	for i := 0; i < numWorkers; i++ {
		c.wg.Add(1)
		go c.worker(tasks)
	}

	c.enqueueTask(startURL, "", domain.LinkContext{}, "", 0)
	
	// In a real long-running process, we might wait for a signal.
	// Here, we wait for waitgroup if we want to track active processing.
	// But with RabbitMQ, "active" is hard to define exactly without an external tracker or empty queue check.
	// For this task, we will just block forever or wait until user interrupts, 
	// OR we assume we want to run until manually stopped.
	// The previous implementation used WaitGroup which tracked active tasks.
	// We can keep using WaitGroup for *active processing* but not for "empty queue" termination easily without more logic.
	// BUT, the prompt says "it will read links from queue and process".
	// Let's Keep the WaitGroup for graceful shutdown if we implement it, but `c.wg.Wait()` might return early if queue is empty but messages are in flight?
	// No, `wg` tracks active `processTask`.
	// The problem is: how do we know when we are "done" if we are decoupled?
	// Usually decoupled consumers run forever.
	// I will make it run forever (block) until context cancellation (not implemented yet) or just block.
	// But `main` expects `Crawl` to block.
	
	// Create a channel to block
	forever := make(chan bool)
	<-forever
	
	// c.wg.Wait() // This logic was for the in-memory recursive crawler.
	// close(c.queue) // We don't close the queue connection here usually unless shutting down.

	log.Printf("------------------------------------------")
	log.Printf("CRAWLING SUMMARY:")
	log.Printf("- Tasks processed:     %d", atomic.LoadInt64(&c.statsProcessed))
	log.Printf("- Pages/Files fetched: %d", atomic.LoadInt64(&c.statsFetched))
	log.Printf("- New files saved:     %d", atomic.LoadInt64(&c.statsSaved))
	log.Printf("- Files skipped:       %d", atomic.LoadInt64(&c.statsSkipped))
	log.Printf("------------------------------------------")

	return nil
}

func (c *CrawlerService) worker(tasks <-chan domain.CrawlTask) {
	defer c.wg.Done()
	for task := range tasks {
		c.processTask(task)
	}
}

func (c *CrawlerService) processTask(task domain.CrawlTask) {
	u := task.URL
	
	if task.Depth > c.maxDepth {
		log.Printf("Max depth reached, skipping: %s", u)
		return
	}

	// Filter out non-http links
	if !strings.HasPrefix(u, "http") {
		return
	}

	log.Printf("[Depth %d] Processing task: %s", task.Depth, u)
	atomic.AddInt64(&c.statsProcessed, 1)

	// Determine file type and name
	lowerURL := strings.ToLower(u)
	parsedURL, _ := url.Parse(lowerURL)
	urlPath := parsedURL.Path
	isPDF := strings.HasSuffix(urlPath, ".pdf") || strings.Contains(lowerURL, "ext=.pdf")
	
	ext := ".htm"
	if isPDF {
		ext = ".pdf"
	}
	
	filename := c.generateFilename(u, ext)

	// Check if file already exists
	if c.storage.Exists(filename) {
		log.Printf("File exists locally, processing: %s", filename)
		atomic.AddInt64(&c.statsSkipped, 1)
		if isPDF {
			c.handlePDF(u, task.Parent, task.ParentContext, task.Description, filename, task.Depth)
		} else {
			c.handleHTML(u, task.Parent, task.ParentContext, task.Description, filename, task.Depth)
		}
		return
	}

	// Fetch and Stream to Storage
	body, err := c.fetcher.Fetch(u)
	if err != nil {
		log.Printf("Error fetching %s: %v", u, err)
		return
	}
	defer body.Close()

	_, err = c.storage.Save(filename, body)
	if err != nil {
		log.Printf("Error saving %s: %v", u, err)
		return
	}
	atomic.AddInt64(&c.statsFetched, 1)
	atomic.AddInt64(&c.statsSaved, 1)

	// Post-process based on type
	// Note: We might have saved an HTML as .pdf if detection failed earlier, 
	// but we trust extension for now as per previous logic.
	// Improvements could be made to sniff header/content before stream, but `Save` consumes the stream.
	
	if isPDF {
		c.handlePDF(u, task.Parent, task.ParentContext, task.Description, filename, task.Depth)
	} else {
		c.handleHTML(u, task.Parent, task.ParentContext, task.Description, filename, task.Depth)
	}
}

func (c *CrawlerService) handlePDF(u, parent string, parentContext domain.LinkContext, description string, filename string, depth int) {
	jsonFilename := strings.TrimSuffix(filename, ".pdf") + ".json"

	// Read from storage using ReadAt for PdfParser
	readerAt, size, err := c.storage.ReadAt(filename)
	if err != nil {
		log.Printf("Error reading PDF content for %s: %v", u, err)
		return
	}
	
	if closer, ok := readerAt.(io.Closer); ok {
		defer closer.Close()
	}

	// Extract Links
	links, err := c.pdfParser.ExtractLinks(readerAt, size)
	if err != nil {
		log.Printf("Error parsing PDF %s: %v", u, err)
	}

	// Save Metadata
	metadata := domain.DocumentMetadata{
		URL:            u,
		LocalPath:      filename, // using filename as local path for now, or we could ask storage for full path if needed
		ParentURL:      parent,
		Description:    description,
		ParentContext:  parentContext,
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
		
		if c.shouldVisit(absLink, "") {
			c.enqueueTask(absLink, u, domain.LinkContext{}, "", depth+1)
		}
	}
}

func (c *CrawlerService) handleHTML(u, parent string, parentContext domain.LinkContext, description string, filename string, depth int) {
	jsonFilename := strings.TrimSuffix(filename, ".htm") + ".json"

	// Read content for HTML parsing
	// For HTML we still need to read it all for parsing generally, but we do it from disk now
	// Ideally we could stream parsing too, but goquery/net/html works with readers, so we can stream read.
	f, err := c.storage.Read(filename)
	if err != nil {
		log.Printf("Error reading HTML content for %s: %v", u, err)
		return
	}
	defer f.Close()

	// Parse HTML (we need to read it twice or tee it if we want both flat and enriched...
	// or we just read it into memory here if it's HTML (usually smaller than PDFs).
	// Let's read into memory for simplicity in parsing logic for now, as HTMLs are small.
	content, err := io.ReadAll(f)
	if err != nil {
		log.Printf("Error reading HTML content %s: %v", u, err)
		return
	}

	// Extract flat links (for backward compat and shouldVisit filtering)
	linksMap, err := c.htmlParser.ExtractLinks(bytes.NewReader(content))
	if err != nil {
		log.Printf("Error parsing HTML %s: %v", u, err)
		return
	}

	// Extract enriched links with hierarchical context
	enrichedLinks, err := c.htmlParser.ExtractLinksWithContext(bytes.NewReader(content))
	if err != nil {
		log.Printf("Error extracting links with context from %s: %v", u, err)
		// Non-fatal: continue with flat links only
	}

	log.Printf("Found %d potential links in HTML (%d with context): %s", len(linksMap), len(enrichedLinks), u)

	var links []string
	for link, text := range linksMap {
		absLink := resolveRelURL(u, link)
		if absLink == "" {
			continue
		}
		links = append(links, absLink)

		if c.shouldVisit(absLink, text) {
			// Find context for this link if available
			var ctx domain.LinkContext
			for _, el := range enrichedLinks {
				// Simple matching by URL; ideally enrichedLinks would be the source of truth
				// but we're keeping backward compat logic here for now
				if strings.Contains(el.URL, link) || el.URL == absLink {
					ctx = el.Context
					break
				}
			}
			c.enqueueTask(absLink, u, ctx, text, depth+1)
		}
	}

	// Resolve relative URLs in enriched links
	var resolvedEnriched []domain.EnrichedLink
	for _, el := range enrichedLinks {
		absURL := resolveRelURL(u, el.URL)
		if absURL == "" {
			continue
		}
		el.URL = absURL
		resolvedEnriched = append(resolvedEnriched, el)
	}

	// Save Metadata
	metadata := domain.DocumentMetadata{
		URL:            u,
		LocalPath:      filename,
		ParentURL:      parent,
		Description:    description,
		ParentContext:  parentContext,
		ExtractedLinks: links,
		LinksContext:   resolvedEnriched,
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

func (c *CrawlerService) enqueueTask(u, parent string, parentContext domain.LinkContext, description string, depth int) {
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
	if !c.visited.TestAndAdd(normalizedURL) {
		log.Printf("Enqueuing [Depth %d]: %s", depth, uWithoutFragment)
		
		err := c.queue.Publish(context.Background(), domain.CrawlTask{
			URL:           uWithoutFragment,
			Parent:        parent,
			ParentContext: parentContext,
			Description:   description,
			Depth:         depth,
		})
		if err != nil {
			log.Printf("Failed to publish task: %v", err)
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


