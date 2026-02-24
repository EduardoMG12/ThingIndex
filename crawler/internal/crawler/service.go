package crawler

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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
	bucketName     string

	// Statistics
	statsProcessed int64
	statsFetched   int64
	statsSaved     int64
	statsSkipped   int64

	// State
	isPurging      atomic.Bool
	activeWorkers  atomic.Int32
	ctx            context.Context
	cancelFunc     context.CancelFunc
}

func NewCrawlerService(fetcher domain.Fetcher, pdfParser domain.Parser, htmlParser HTMLParser, storage domain.Storage, queue domain.Queue, maxDepth int, ignoreKeywords []string, outputSubdir string, bucketName string) *CrawlerService {
	// Initialize Bloom Filter with 1M items and 0.01 false positive rate
	bf := NewBloomFilter(1000000, 0.01)

	c := &CrawlerService{
		fetcher:        fetcher,
		pdfParser:      pdfParser,
		htmlParser:     htmlParser,
		storage:        storage,
		queue:          queue,
		visited:        bf,
		maxDepth:       maxDepth,
		ignoreKeywords: ignoreKeywords,
		outputSubdir:   outputSubdir,
		bucketName:     bucketName,
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	c.ctx = ctx
	c.cancelFunc = cancel
	
	return c
}

func (c *CrawlerService) Crawl() error {
	// Consumer loop
	tasks, err := c.queue.Consume()
	if err != nil {
		return fmt.Errorf("failed to start consumer: %w", err)
	}

	// Start workers
	numWorkers := 10
	for i := 0; i < numWorkers; i++ {
		c.wg.Add(1)
		go c.worker(i, tasks)
	}
	
	// Periodic RAM cleanup to ensure memory is returned to OS after bursts
	go func() {
		// Run every minute
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			runtime.GC()
			debug.FreeOSMemory()
		}
	}()

	// Create a channel to block
	forever := make(chan bool)
	<-forever


	log.Printf("------------------------------------------")

	return nil
}

func (c *CrawlerService) worker(id int, deliveries <-chan domain.TaskDelivery) {
	defer c.wg.Done()
	log.Printf("[Worker %d] Started and waiting for tasks...", id)
	for delivery := range deliveries {
		c.processTask(id, delivery)
	}
	log.Printf("[Worker %d] Stopped.", id)
}

func (c *CrawlerService) processTask(workerID int, delivery domain.TaskDelivery) {
	task := delivery.Task
	u := task.URL
	
	log.Printf("[Worker %d] Processing: %s (Depth %d)", workerID, u, task.Depth)
	
	// Increase active workers count
	c.activeWorkers.Add(1)
	defer c.activeWorkers.Add(-1)

	// Check for special commands
	if u == "__PURGE__" {
		log.Println("Received PURGE command via queue. Purging all messages...")
		
		// Set purge flag to true so currently executing workers drop their tasks
		c.isPurging.Store(true)

		// Cancel any running requests immediately
		c.cancelFunc()
		
		// Give workers a moment to see the cancel before purging the queue
		time.Sleep(100 * time.Millisecond)

		if err := c.queue.Purge(); err != nil {
			log.Printf("Failed to purge queue: %v", err)
		} else {
			log.Println("Queue successfully purged.")
			// Reset the bloom filter
			c.visited = NewBloomFilter(1000000, 0.01)
			log.Println("Internal visited cache reset.")
		}
		
		// Initialize a fresh context for new tasks
		c.ctx, c.cancelFunc = context.WithCancel(context.Background())
		
		// Keep isPurging=true for a while to allow the internal RabbitMQ buffer (100 msgs) 
		// to be drained and dropped by workers.
		go func() {
			time.Sleep(5 * time.Second)
			c.isPurging.Store(false)
			
			// Force Garbage Collection to return memory to OS
			log.Println("Forcing garbage collection to free RAM...")
			runtime.GC()
			debug.FreeOSMemory()
			log.Println("RAM freed successfully.")
		}()
		
		delivery.Ack()
		return
	}

	// Wait out processing if we are actively purging to drain current tasks
	if c.isPurging.Load() {
		log.Printf("Dropping task %s (purge in progress)", u)
		delivery.Nack(false) // don't requeue
		return
	}

	// Determine the effective max depth for this task
	effectiveMaxDepth := c.maxDepth
	if task.MaxDepth > 0 {
		effectiveMaxDepth = task.MaxDepth
	}

	if task.Depth > effectiveMaxDepth {
		log.Printf("[Worker %d] Max depth reached (%d > %d), skipping: %s", workerID, task.Depth, effectiveMaxDepth, u)
		delivery.Ack()
		return
	}

	// Filter out non-http links
	if !strings.HasPrefix(u, "http") {
		delivery.Ack()
		return
	}

	log.Printf("[Worker %d][Depth %d] Processing task: %s", workerID, task.Depth, u)
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
	
	filename := c.generateFilename(u, ext, task.DirName)

	// Check if file already exists
	if c.storage.Exists(filename) {
		log.Printf("[Worker %d] File exists locally, processing: %s", workerID, filename)
		atomic.AddInt64(&c.statsSkipped, 1)
		if isPDF {
			c.handlePDF(u, task.Parent, task.ParentContext, task.Description, task.DirName, filename, task.Depth, task.MaxDepth)
		} else {
			c.handleHTML(u, task.Parent, task.ParentContext, task.Description, task.DirName, filename, task.Depth, task.MaxDepth)
		}
		// Stream per-file notification to parser
		c.publishToParser(filename, task.DirName)
		delivery.Ack()
		return
	}

	// Fetch and Stream to Storage
	fetchCtx, fetchCancel := context.WithTimeout(c.ctx, 2*time.Minute)
	defer fetchCancel()
	
	log.Printf("[Worker %d] Fetching: %s", workerID, u)
	body, err := c.fetcher.Fetch(fetchCtx, u)
	if err != nil {
		log.Printf("[Worker %d] Error fetching %s: %v", workerID, u, err)
		
		if c.isPurging.Load() {
			delivery.Nack(false) // Drop it if we are purging
			return
		}
		
		// If it's a permanent error (like 404 Not Found or DNS resolution failure), don't requeue it.
		if strings.Contains(err.Error(), "permanent") {
			log.Printf("[Worker %d] Discarding permanent error for %s", workerID, u)
			delivery.Ack()
		} else {
			delivery.Nack(true) // Requeue on transient fetch error
		}
		return
	}
	defer body.Close()

	log.Printf("[Worker %d] Saving to storage: %s", workerID, filename)
	_, err = c.storage.Save(filename, body)
	if err != nil {
		log.Printf("[Worker %d] Error saving %s: %v", workerID, u, err)
		if c.isPurging.Load() {
			delivery.Nack(false) // Drop it if we are purging
		} else {
			delivery.Nack(true) // Requeue on save error
		}
		return
	}
	atomic.AddInt64(&c.statsFetched, 1)
	atomic.AddInt64(&c.statsSaved, 1)

	// Post-process based on type
	if isPDF {
		c.handlePDF(u, task.Parent, task.ParentContext, task.Description, task.DirName, filename, task.Depth, task.MaxDepth)
	} else {
		c.handleHTML(u, task.Parent, task.ParentContext, task.Description, task.DirName, filename, task.Depth, task.MaxDepth)
	}

	// Stream per-file notification to parser
	c.publishToParser(filename, task.DirName)

	delivery.Ack() // Successfully processed
}

func (c *CrawlerService) handlePDF(u, parent string, parentContext domain.LinkContext, description string, dirName string, filename string, depth int, maxDepth int) {
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
			c.enqueueTask(absLink, u, domain.LinkContext{}, "", dirName, depth+1, maxDepth)
		}
	}
}

func (c *CrawlerService) handleHTML(u, parent string, parentContext domain.LinkContext, description string, dirName string, filename string, depth int, maxDepth int) {
	jsonFilename := strings.TrimSuffix(filename, ".htm") + ".json"

	f, err := c.storage.Read(filename)
	if err != nil {
		log.Printf("Error reading HTML content for %s: %v", u, err)
		return
	}
	defer f.Close()

	// Parse HTML content into memory
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
			c.enqueueTask(absLink, u, ctx, text, dirName, depth+1, maxDepth)
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

func (c *CrawlerService) enqueueTask(u, parent string, parentContext domain.LinkContext, description string, dirName string, depth int, maxDepth int) {
	// Filter out non-http links
	if !strings.HasPrefix(u, "http") {
		return
	}

	// Determine effective max depth
	effectiveMaxDepth := c.maxDepth
	if maxDepth > 0 {
		effectiveMaxDepth = maxDepth
	}

	// Check depth limit before enqueuing to avoid marking as visited prematurely
	if depth > effectiveMaxDepth {
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
			MaxDepth:      maxDepth,
			DirName:       dirName,
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

func (c *CrawlerService) generateFilename(u string, ext string, dirName string) string {
	parsed, _ := url.Parse(u)
	base := filepath.Base(parsed.Path)
	if base == "" || base == "." || base == "/" {
		base = "index"
	}
	
	// Create deterministic hash of the URL
	h := sha1.New()
	h.Write([]byte(u))
	hash := fmt.Sprintf("%x", h.Sum(nil))
	
	// Sanitize base name - strictly keep only ASCII letters, numbers, dash, underscore, dot
	// MinIO crashes on ISO-8859-1 url decoded characters (e.g., constituiao.htm)
	var safeBuilder strings.Builder
	for _, r := range base {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			safeBuilder.WriteRune(r)
		}
	}
	base = safeBuilder.String()
	
	if base == "" || base == ext {
		base = "document"
	}
	
	// Ensure extension
	if !strings.HasSuffix(base, ext) {
		base += ext
	}
	
	filename := fmt.Sprintf("%s_%s", hash[:10], base)
	
	// Prepend dynamic folder from RabbitMQ task payload
	if dirName != "" {
		filename = filepath.Join(dirName, filename)
	}
	
	if c.outputSubdir != "" {
		return filepath.Join(c.outputSubdir, filename)
	}
	return filename
}

// publishToParser sends a per-file notification to the raw_documents queue
// so the parser can process it immediately without waiting for the whole crawl to finish.
func (c *CrawlerService) publishToParser(contentKey string, topic string) {
	// Derive metadata key from content key (same name, .json extension)
	metadataKey := contentKey
	for _, ext := range []string{".htm", ".html", ".pdf"} {
		if strings.HasSuffix(metadataKey, ext) {
			metadataKey = strings.TrimSuffix(metadataKey, ext) + ".json"
			break
		}
	}

	payload := map[string]string{
		"content_key":  contentKey,
		"metadata_key": metadataKey,
		"bucket":       c.bucketName,
		"topic":        topic,
	}
	payloadBytes, _ := json.Marshal(payload)

	err := c.queue.PublishToQueue(context.Background(), "raw_documents", payloadBytes)
	if err != nil {
		log.Printf("Failed to publish to raw_documents: %v", err)
	} else {
		log.Printf("Published to raw_documents: %s (topic=%s)", contentKey, topic)
	}
}
