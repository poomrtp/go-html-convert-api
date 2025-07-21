package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/gin-gonic/gin"
)

var (
	globalAllocCtx     context.Context
	globalCancel       context.CancelFunc
	initOnce           sync.Once
	allocMutex         sync.Mutex
	lastHealthCheck    time.Time
	healthCheckResult  bool
	healthCheckMutex   sync.RWMutex
	consecutiveErrors  int

	// Chrome Context Pool
	chromeContextPool   = make(chan context.Context, 5)
	chromeContextCancel = make(chan context.CancelFunc, 5)
	poolInitialized     = false
	poolMutex          sync.Mutex
)

const (
	healthCheckCacheDuration = 30 * time.Second
	cacheMaxAge             = 10 * time.Minute
	cacheMaxSize            = 100
)

// Cache structures
type CacheKey struct {
	HTML   string
	Format string
}

type CacheEntry struct {
	Data      []byte
	Timestamp time.Time
	MimeType  string
}

var (
	resultCache = make(map[CacheKey]CacheEntry)
	cacheMutex  sync.RWMutex
)

// Conversion request structure
type ConversionRequest struct {
	HTML    string
	Format  string
	Context *gin.Context
}

// countChromeProcesses counts running Chrome processes
func countChromeProcesses() int {
	cmd := exec.Command("pgrep", "-f", "chrome")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return 0
	}
	return len(lines)
}

// initChromeAllocator สร้าง Chrome allocator ครั้งเดียวตอนเริ่มต้น
func initChromeAllocator() {
	initOnce.Do(func() {
		log.Printf("Initializing global Chrome allocator... (Current Chrome processes: %d)", countChromeProcesses())

		opts := append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("disable-dev-shm-usage", true),
			chromedp.Flag("disable-web-security", true),
			chromedp.Flag("disable-background-timer-throttling", true),
			chromedp.Flag("disable-renderer-backgrounding", true),
			chromedp.Flag("disable-backgrounding-occluded-windows", true),
			chromedp.Flag("disable-features", "TranslateUI,VizDisplayCompositor"),
			chromedp.Flag("memory-pressure-off", true),
			chromedp.Flag("max_old_space_size", "4096"),
		)

		allocMutex.Lock()
		globalAllocCtx, globalCancel = chromedp.NewExecAllocator(context.Background(), opts...)
		allocMutex.Unlock()

		log.Printf("Chrome allocator initialized successfully")
	})
}


func initChromeContextPool() {
	poolMutex.Lock()
	defer poolMutex.Unlock()

	if poolInitialized {
		return
	}

	initChromeAllocator()
	log.Printf("Initializing Chrome context pool...")

	for i := 0; i < 5; i++ {
		ctx, cancel := chromedp.NewContext(globalAllocCtx)

		go func() {
			chromedp.Run(ctx,
				chromedp.Navigate("about:blank"),
				chromedp.WaitReady("body"),
			)
		}()

		chromeContextPool <- ctx
		chromeContextCancel <- cancel
	}

	poolInitialized = true
	log.Printf("Context pool ready")
}


func getChromeContextFromPool() (context.Context, func()) {
	initChromeContextPool()

	select {
	case ctx := <-chromeContextPool:
		cancel := <-chromeContextCancel
		return ctx, func() {
			select {
			case chromeContextPool <- ctx:
				chromeContextCancel <- cancel
			default:
				cancel()
			}
		}
	default:
		ctx, cancel := chromedp.NewContext(globalAllocCtx)
		return ctx, cancel
	}
}


func getCacheKey(html, format string) CacheKey {
	hasher := sha256.New()
	hasher.Write([]byte(html + format))
	hashStr := fmt.Sprintf("%x", hasher.Sum(nil))[:16]

	return CacheKey{
		HTML:   hashStr,
		Format: format,
	}
}

func getCachedResult(html, format string) ([]byte, string, bool) {
	key := getCacheKey(html, format)

	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	entry, exists := resultCache[key]
	if !exists {
		return nil, "", false
	}

	// Check expiry
	if time.Since(entry.Timestamp) > cacheMaxAge {
		return nil, "", false
	}

	log.Printf("Cache HIT for %s", format)
	return entry.Data, entry.MimeType, true
}

func setCachedResult(html, format string, data []byte, mimeType string) {
	key := getCacheKey(html, format)

	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	if len(resultCache) >= cacheMaxSize {
		oldestKey := CacheKey{}
		oldestTime := time.Now()

		for k, v := range resultCache {
			if v.Timestamp.Before(oldestTime) {
				oldestTime = v.Timestamp
				oldestKey = k
			}
		}

		if oldestKey.HTML != "" {
			delete(resultCache, oldestKey)
		}
	}

	resultCache[key] = CacheEntry{
		Data:      data,
		Timestamp: time.Now(),
		MimeType:  mimeType,
	}

	log.Printf("Cache SET for %s", format)
}


func checkChromeHealth() bool {
	healthCheckMutex.RLock()
	if time.Since(lastHealthCheck) < healthCheckCacheDuration {
		result := healthCheckResult
		healthCheckMutex.RUnlock()

		return result
	}
	healthCheckMutex.RUnlock()

	healthCheckMutex.Lock()
	defer healthCheckMutex.Unlock()

	if time.Since(lastHealthCheck) < healthCheckCacheDuration {

		return healthCheckResult
	}

	ctx, returnToPool := getChromeContextFromPool()
	defer returnToPool()

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	err := chromedp.Run(ctx, chromedp.Navigate("about:blank"))

	lastHealthCheck = time.Now()
	healthCheckResult = err == nil

	if err != nil {
		consecutiveErrors++
		log.Printf("Health check failed: %v", err)
	} else {
		consecutiveErrors = 0
	}

	return healthCheckResult
}

func shouldPerformHealthCheck() bool {
	healthCheckMutex.RLock()
	defer healthCheckMutex.RUnlock()

	return consecutiveErrors > 0 ||
		lastHealthCheck.IsZero() ||
		time.Since(lastHealthCheck) >= 5*time.Minute
}

func reinitializeChrome() {
	log.Printf("Reinitializing Chrome...")

	if globalCancel != nil {
		globalCancel()
	}

	allocMutex.Lock()
	initOnce = sync.Once{}
	allocMutex.Unlock()

	healthCheckMutex.Lock()
	lastHealthCheck = time.Time{}
	healthCheckResult = false
	healthCheckMutex.Unlock()

	poolMutex.Lock()
	poolInitialized = false
	poolMutex.Unlock()

	initChromeAllocator()
	initChromeContextPool()
}


func executeHTMLToPNG(ctx context.Context, html string) ([]byte, string, error) {
	var buf []byte
	err := chromedp.Run(ctx,
		chromedp.Navigate("about:blank"),
		chromedp.Evaluate(fmt.Sprintf(`document.open();document.write(%q);document.close();`, html), nil),
		chromedp.WaitReady("body"),
		chromedp.FullScreenshot(&buf, 90),
	)
	return buf, "image/png", err
}

func executeHTMLToPDF(ctx context.Context, html string) ([]byte, string, error) {
	var buf []byte
	err := chromedp.Run(ctx,
		chromedp.Navigate("about:blank"),
		chromedp.Evaluate(fmt.Sprintf(`document.documentElement.innerHTML = %q;`, html), nil),
		chromedp.WaitVisible(`body`),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			buf, err = pagePrintToPDF(ctx)
			return err
		}),
	)
	return buf, "application/pdf", err
}


func convertHTML(req ConversionRequest) {
	start := time.Now()
	log.Printf("Converting HTML to %s", strings.ToUpper(req.Format))
	if data, mimeType, found := getCachedResult(req.HTML, req.Format); found {
		req.Context.Data(http.StatusOK, mimeType, data)
		log.Printf("%s conversion from cache: %v (%d bytes)",
			strings.ToUpper(req.Format), time.Since(start), len(data))
		return
	}

	if shouldPerformHealthCheck() {
		if !checkChromeHealth() {
			reinitializeChrome()
		}
	}
	ctx, returnToPool := getChromeContextFromPool()
	defer returnToPool()

	timeout := 60 * time.Second
	if req.Format == "pdf" {
		timeout = 45 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer func() {
		cancel()
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("Chrome timeout: %v", timeout)
			consecutiveErrors++
		}
	}()


	var data []byte
	var mimeType string
	var err error

	if req.Format == "png" {
		data, mimeType, err = executeHTMLToPNG(ctx, req.HTML)
	} else {
		data, mimeType, err = executeHTMLToPDF(ctx, req.HTML)
	}

	if err != nil {
		consecutiveErrors++
		log.Printf("Conversion error: %v", err)
		req.Context.JSON(http.StatusInternalServerError, gin.H{
			"error":   fmt.Sprintf("Failed to convert HTML to %s", strings.ToUpper(req.Format)),
			"details": err.Error(),
		})
		return
	}

	consecutiveErrors = 0
	setCachedResult(req.HTML, req.Format, data, mimeType)
	req.Context.Data(http.StatusOK, mimeType, data)

	log.Printf("%s conversion completed: %v (%d bytes)",
		strings.ToUpper(req.Format), time.Since(start), len(data))
}


func convertHTMLToPNG(c *gin.Context) {
	var req HTMLRequest
	if err := c.ShouldBindJSON(&req); err != nil {

		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	convertHTML(ConversionRequest{
		HTML:    req.HTML,
		Format:  "png",
		Context: c,
	})
}

func convertHTMLToPDF(c *gin.Context) {
	var req HTMLRequest
	if err := c.ShouldBindJSON(&req); err != nil {

		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	convertHTML(ConversionRequest{
		HTML:    req.HTML,
		Format:  "pdf",
		Context: c,
	})
}


func pagePrintToPDF(ctx context.Context) ([]byte, error) {
	var buf []byte
	err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		buf, _, err = pagePrintToPDFInternal(ctx)
		return err
	}))
	return buf, err
}

func pagePrintToPDFInternal(ctx context.Context) ([]byte, *string, error) {
	pdfParams := page.PrintToPDF()
	pdfBuf, _, err := pdfParams.Do(ctx)
	return pdfBuf, nil, err
}


func shutdownChrome() {
	if globalCancel != nil {
		log.Printf("Shutting down Chrome...")
		globalCancel()

		poolMutex.Lock()
		for len(chromeContextPool) > 0 {
			<-chromeContextPool
			cancel := <-chromeContextCancel
			cancel()
		}
		poolInitialized = false
		poolMutex.Unlock()

		time.Sleep(1 * time.Second)
		log.Printf("Chrome shutdown completed")
	}
}
