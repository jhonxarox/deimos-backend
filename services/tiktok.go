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

// SearchTikTokVideos with pagination
func SearchTikTokVideos(query string, page int) ([]Video, error) {
	var videos []Video
	itemsPerPage := 6
	scrollsNeeded := (page * itemsPerPage) / itemsPerPage // Number of scrolls needed to load required items

	// Create context for chromedp
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Initialize the HTML content
	var htmlContent string
	tiktokSearchURL := fmt.Sprintf("https://www.tiktok.com/search?q=%s", query)

	// Navigate and scroll to load more content
	for i := 0; i <= scrollsNeeded; i++ {
		err := chromedp.Run(ctx,
			chromedp.Navigate(tiktokSearchURL),
			chromedp.WaitVisible(`div[data-e2e="search_top-item-list"]`, chromedp.ByQuery),
			chromedp.ScrollIntoView(`div[data-e2e="search_top-item-list"]`, chromedp.ByQuery),
			chromedp.Sleep(2*time.Second), // Adjust sleep time if necessary
			chromedp.OuterHTML("html", &htmlContent),
		)
		if err != nil {
			log.Printf("Error while scrolling: %v", err)
			return nil, err
		}
	}

	// Parse the loaded HTML with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		log.Printf("Failed to parse HTML: %v", err)
		return nil, err
	}

	// Extract video data from HTML
	doc.Find(`div[data-e2e="search_top-item"]`).Each(func(i int, s *goquery.Selection) {
		// Stop once we have enough items for the page
		if len(videos) >= page*itemsPerPage {
			return
		}

		videoLink, exists := s.Find("a").Attr("href")
		if !exists {
			return
		}

		if !strings.HasPrefix(videoLink, "http") {
			videoLink = "https://www.tiktok.com" + videoLink
		}

		thumbnail, exists := s.Find("img").Attr("src")
		if !exists {
			return
		}

		// Check if the thumbnail is a valid URL
		if _, err := url.ParseRequestURI(thumbnail); err != nil {
			log.Printf("Invalid thumbnail URL: %s", thumbnail)
			return // Skip this video if the thumbnail is not a valid URL
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

	// Calculate the start and end index for pagination
	start := (page - 1) * itemsPerPage
	end := start + itemsPerPage

	// Safely slice videos based on pagination
	if start >= len(videos) {
		return nil, fmt.Errorf("no more data available")
	}
	if end > len(videos) {
		end = len(videos)
	}

	// Return only the requested page of videos
	return videos[start:end], nil
}

// GetVideoUrl scrapes the video URL from a TikTok video page and follows redirects
func GetVideoUrl(videoPageUrl string) (string, error) {
	// Validate if the input is a valid URL
	_, err := url.ParseRequestURI(videoPageUrl)
	if err != nil {
		return "", errors.New("invalid video URL")
	}

	// Set up a context with chromedp
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Variable to store the HTML content
	var htmlContent string

	// Use chromedp to navigate to the video page and retrieve the HTML
	err = chromedp.Run(ctx,
		chromedp.Navigate(videoPageUrl),
		chromedp.Sleep(2*time.Second), // Wait for page to load
		chromedp.OuterHTML("html", &htmlContent),
	)
	if err != nil {
		return "", err
	}

	// Load the HTML content into goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		log.Printf("Failed to parse HTML: %v", err)
		return "", err
	}

	// Find the <video> tag and select the third <source> element inside it
	var videoUrl string
	doc.Find("video source").EachWithBreak(func(i int, s *goquery.Selection) bool {
		if i == 2 { // Select the third <source> (index starts from 0)
			videoUrl, _ = s.Attr("src")
			return false // Stop iteration once the third <source> is found
		}
		return true
	})

	// Check if a video URL was found
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

	// Set headers to mimic a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.82 Safari/537.36")
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
