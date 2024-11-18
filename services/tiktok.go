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

// Video struct to hold the scraped video information
type Video struct {
	URL       string `json:"url"`
	Thumbnail string `json:"thumbnail"`
	Caption   string `json:"caption"`
	User      string `json:"user"`
}

// isValidThumbnailURL checks if the thumbnail URL is a valid HTTP/HTTPS URL
func isValidThumbnailURL(thumbnail string) bool {
	parsedURL, err := url.Parse(thumbnail)
	if err != nil {
		return false
	}
	return (parsedURL.Scheme == "http" || parsedURL.Scheme == "https") && !strings.HasPrefix(thumbnail, "data:image")
}

// SearchTikTokVideos fetches video data with pagination
func SearchTikTokVideos(query string, page int) ([]Video, error) {
	var videos []Video
	itemsPerPage := 6
	scrollsNeeded := page // Number of scrolls needed based on the page

	// Persistent chromedp context with optimized flags
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)
	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAllocator()

	chromedpCtx, cancel := chromedp.NewContext(allocatorCtx)
	defer cancel()

	var htmlContent string
	tiktokSearchURL := fmt.Sprintf("https://www.tiktok.com/search?q=%s", query)

	for i := 0; i < scrollsNeeded; i++ {
		err := chromedp.Run(chromedpCtx,
			chromedp.Navigate(tiktokSearchURL),
			chromedp.WaitVisible(`div[data-e2e="search_top-item-list"]`, chromedp.ByQuery),
			chromedp.ScrollIntoView(`div[data-e2e="search_top-item-list"]`, chromedp.ByQuery),
			chromedp.Sleep(1*time.Second),
			chromedp.OuterHTML("html", &htmlContent),
		)
		if err != nil {
			log.Printf("Error while scrolling: %v", err)
			return nil, err
		}

		// Parse and extract video data
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
		if err != nil {
			log.Printf("Failed to parse HTML: %v", err)
			return nil, err
		}

		doc.Find(`div[data-e2e="search_top-item"]`).Each(func(i int, s *goquery.Selection) {
			videoLink, exists := s.Find("a").Attr("href")
			if !exists {
				return
			}
			if !strings.HasPrefix(videoLink, "http") {
				videoLink = "https://www.tiktok.com" + videoLink
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
				User:      "https://www.tiktok.com" + user,
			})
		})

		// Stop early if we have enough videos
		if len(videos) >= page*itemsPerPage {
			break
		}
	}

	// Return only the requested page of videos
	start := (page - 1) * itemsPerPage
	end := start + itemsPerPage
	if end > len(videos) {
		end = len(videos)
	}
	return videos[start:end], nil
}

// GetVideoUrl fetches a direct video URL
func GetVideoUrl(videoPageUrl string) (string, error) {
	_, err := url.ParseRequestURI(videoPageUrl)
	if err != nil {
		return "", errors.New("invalid video URL")
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)
	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAllocator()

	chromedpCtx, cancel := chromedp.NewContext(allocatorCtx)
	defer cancel()

	var htmlContent string
	err = chromedp.Run(chromedpCtx,
		chromedp.Navigate(videoPageUrl),
		chromedp.Sleep(1*time.Second),
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

// ProxyVideoContent proxies video content
func ProxyVideoContent(videoUrl string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", videoUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
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
