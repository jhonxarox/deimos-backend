package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

type Video struct {
	URL       string `json:"url"`
	Thumbnail string `json:"thumbnail"`
	Caption   string `json:"caption"`
	User      string `json:"user"`
}

func isValidThumbnailURL(thumbnail string) bool {
	parsedURL, err := url.Parse(thumbnail)
	if err != nil {
		return false
	}
	return (parsedURL.Scheme == "http" || parsedURL.Scheme == "https") && !strings.HasPrefix(thumbnail, "data:image")
}

// SearchTikTokVideos fetches videos with pagination
func SearchTikTokVideos(ctx context.Context, tiktokBaseURL, query string, page int) ([]Video, error) {
	var videos []Video
	itemsPerPage := 3

	// Create a chromedp context for the session
	chromedpCtx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	// Add timeout to the context
	timeoutCtx, timeoutCancel := context.WithTimeout(chromedpCtx, 90*time.Second) // Increased timeout
	defer timeoutCancel()

	var htmlContent string
	tiktokSearchURL := fmt.Sprintf("%s/search?q=%s", tiktokBaseURL, query)

	for i := 0; i < page; i++ {
		log.Printf("Navigating to TikTok URL: %s", tiktokSearchURL)

		err := chromedp.Run(timeoutCtx,
			chromedp.Navigate(tiktokSearchURL),
			chromedp.WaitVisible(`body`, chromedp.ByQuery),
			chromedp.WaitVisible(`div[data-e2e="search_top-item-list"]`, chromedp.ByQuery),
			chromedp.ScrollIntoView(`div[data-e2e="search_top-item-list"]`, chromedp.ByQuery),
			chromedp.Sleep(7*time.Second), // Increased sleep for dynamic loading
			chromedp.OuterHTML("html", &htmlContent),
		)
		if err != nil {
			log.Printf("Scraping error: %v", err)
			continue
		}

		// Parse and extract video data
		extractedVideos, err := extractVideosFromHTML(htmlContent, tiktokBaseURL)
		if err != nil {
			log.Printf("Error extracting videos: %v", err)
			continue
		}

		videos = append(videos, extractedVideos...)
		if len(videos) >= itemsPerPage*page {
			break
		}
	}

	start := (page - 1) * itemsPerPage
	end := start + itemsPerPage
	if end > len(videos) {
		end = len(videos)
	}

	return videos[start:end], nil
}

func extractVideosFromHTML(htmlContent string, tiktokBaseURL string) ([]Video, error) {
	var videos []Video
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	doc.Find(`div[data-e2e="search_top-item"]`).Each(func(i int, s *goquery.Selection) {
		videoLink, exists := s.Find("a").Attr("href")
		if !exists {
			return
		}
		if !strings.HasPrefix(videoLink, "http") {
			videoLink = tiktokBaseURL + videoLink
		}

		thumbnail, exists := s.Find("img").Attr("src")
		if !exists || !isValidThumbnailURL(thumbnail) {
			return
		}

		descSection := s.Next()
		caption := descSection.Find(`div[data-e2e="search-card-video-caption"]`).Text()
		user, exists := descSection.Find(`a[data-e2e="search-card-user-link"]`).Attr("href")
		if !exists {
			return
		}

		videos = append(videos, Video{
			URL:       videoLink,
			Thumbnail: thumbnail,
			Caption:   caption,
			User:      tiktokBaseURL + user,
		})
	})

	return videos, nil
}

// GetVideoUrl scrapes the video URL from a TikTok video page
func GetVideoUrl(ctx context.Context, videoPageUrl string) (string, error) {
	_, err := url.ParseRequestURI(videoPageUrl)
	if err != nil {
		return "", errors.New("invalid video URL")
	}

	chromedpCtx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	var htmlContent string

	err = chromedp.Run(chromedpCtx,
		chromedp.Navigate(videoPageUrl),
		chromedp.Sleep(5*time.Second),
		chromedp.OuterHTML("html", &htmlContent),
	)
	if err != nil {
		return "", err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return "", err
	}

	var videoUrl string
	doc.Find("video source").EachWithBreak(func(i int, s *goquery.Selection) bool {
		if i == 2 {
			videoUrl, _ = s.Attr("src")
			return false
		}
		return true
	})

	if videoUrl == "" {
		return "", errors.New("video source not found")
	}

	return videoUrl, nil
}

// ProxyVideoContent fetches video content directly from the TikTok CDN
func ProxyVideoContent(videoUrl string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", videoUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://www.tiktok.com/")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received status code %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
