# Project Overview
This backend service provides a web API that allows users to search for TikTok videos by keywords and retrieve video details, including URLs for playback. The service uses headless browser scraping to gather video data from TikTok, as TikTok does not provide an official public API for this.

## Technologies Used
- Go: Main backend language.
- Gin: Lightweight HTTP web framework for Go.
- Chromedp: Headless browser automation for scraping TikTok's search results.
- Goquery: A Go library that simplifies HTML parsing.

### Prerequisites
- Go: Make sure you have Go installed. You can download it from [here](https://golang.org/dl/).
- Chromium Browser: Required for Chromedp to run a headless browser session.

## Getting Started
1. Clone the Repository

```bash
Copy code
git clone <your-repo-url>
cd backend
Install Dependencies
```

2. Copy code
```bash
go mod tidy
```

3. Run the Application
```bash
go run main.go
```

4. API Endpoints

- Search TikTok Videos
`GET /search/:query?page=1`

    - Parameters:
        - `query`: Keyword to search videos on TikTok.
        - `page`: Page number for paginated results.
    - Response:
        - Returns an array of videos with details like `URL`, `Thumbnail`, `Caption`, and `User`.

- Get Video URL
`GET /get-video-url?url=<TikTok_video_page_url>`

- Parameters:
    - `url`: Full URL of the TikTok video page.
- Response:
    - Returns the direct video URL for playback.
5. Environment Configuration

Make sure to adjust the following in the code if needed:
- Port: Currently hardcoded to `8080`.
- Logging: Check Chromedp logging for debugging scraping issues.

## Project Structure
```plaintext
backend/
├── main.go                # Application entry point
├── services/
│   └── tiktok.go          # TikTok scraping functions
├── go.mod                 # Go module dependencies
└── go.sum                 # Dependency locks
```