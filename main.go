package main

import (
	"deimosbackend/services"
	"net/http"
	"strconv"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize a Gin router
	router := gin.Default()

	// Use the CORS middleware with default settings
	router.Use(cors.Default())

	// Define the search route with pagination
	router.GET("/search/:query", func(c *gin.Context) {
		query := c.Param("query")

		// Get the page number from query parameters, defaulting to 1 if not provided
		pageParam := c.DefaultQuery("page", "1")
		page, err := strconv.Atoi(pageParam)
		if err != nil || page < 1 {
			page = 1 // Ensure page is at least 1
		}

		// Call SearchTikTokVideos with the query and page
		videos, err := services.SearchTikTokVideos(query, page)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"videos": videos})
	})

	// New endpoint to get the video URL
	router.GET("/get-video-url", func(c *gin.Context) {
		url := c.Query("url")
		if url == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "url parameter is required"})
			return
		}

		videoUrl, err := services.GetVideoUrl(url)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"videoUrl": videoUrl})
	})

	// Proxy endpoint for the video content
	router.GET("/proxy-video", func(c *gin.Context) {
		videoUrl := c.Query("url")
		if videoUrl == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "url parameter is required"})
			return
		}

		videoContent, err := services.ProxyVideoContent(videoUrl)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Stream the video content to the client
		c.Data(http.StatusOK, "video/mp4", videoContent)
	})

	// Run the server on port 8080
	router.Run(":8080")
}
