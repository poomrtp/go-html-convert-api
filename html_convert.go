package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/gin-gonic/gin"
)

func convertHTMLToPNG(c *gin.Context) {
	var req HTMLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var buf []byte
	if err := chromedp.Run(ctx,
		chromedp.Navigate("about:blank"),
		chromedp.Evaluate(fmt.Sprintf(`document.documentElement.innerHTML = %q;`, req.HTML), nil),
		chromedp.WaitVisible(`body`),
		chromedp.FullScreenshot(&buf, 90),
	); err != nil {
		log.Printf("Error converting HTML to PNG: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert HTML to PNG", "details": err.Error()})
		return
	}

	c.Data(http.StatusOK, "image/png", buf)
}

func convertHTMLToPDF(c *gin.Context) {
	var req HTMLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var pdfBuf []byte
	if err := chromedp.Run(ctx,
		chromedp.Navigate("about:blank"),
		chromedp.Evaluate(fmt.Sprintf(`document.documentElement.innerHTML = %q;`, req.HTML), nil),
		chromedp.WaitVisible(`body`),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfBuf, err = pagePrintToPDF(ctx)
			return err
		}),
	); err != nil {
		log.Printf("Error converting HTML to PDF: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert HTML to PDF", "details": err.Error()})
		return
	}

	c.Data(http.StatusOK, "application/pdf", pdfBuf)
}

// pagePrintToPDF is a helper to call chromedp's PDF printing
func pagePrintToPDF(ctx context.Context) ([]byte, error) {
	var buf []byte
	err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		buf, _, err = pagePrintToPDFInternal(ctx)
		return err
	}))
	return buf, err
}

// pagePrintToPDFInternal uses the chromedp/cdproto/page package to print to PDF
func pagePrintToPDFInternal(ctx context.Context) ([]byte, *string, error) {
	pdfParams := page.PrintToPDF()
	pdfBuf, _, err := pdfParams.Do(ctx)
	return pdfBuf, nil, err
}
