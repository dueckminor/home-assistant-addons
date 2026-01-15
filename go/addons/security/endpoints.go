package security

import (
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type FileInfo struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	IsDir   bool      `json:"isDir"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
	Type    string    `json:"type,omitempty"` // file, video, image
}

type DirectoryListing struct {
	Path        string     `json:"path"`
	Files       []FileInfo `json:"files"`
	Directories []FileInfo `json:"directories"`
}

// TODO: add the following endpoints:
//
// /api/cameras - List available cameras
// /api/stream/live/{cameraId} - Live stream endpoint
// /api/stream/snapshot/{cameraId} - Current snapshot
//
// add RTSP to WebRTC conversion for live streaming

func (s *security) setupSecurityEndpoints(r *gin.Engine) {
	api := r.Group("/api")
	{
		// List contents of FTP directory or subdirectory
		api.GET("/ftp/*path", s.listFTPDirectory)

		// Serve files from FTP directory (videos, images, etc.)
		api.GET("/files/*path", s.serveFTPFile)

		// Get file info/metadata
		api.GET("/fileinfo/*path", s.getFTPFileInfo)
	}
}

// listFTPDirectory lists the contents of the FTP directory
func (s *security) listFTPDirectory(c *gin.Context) {
	requestPath := c.Param("path")
	if requestPath == "" || requestPath == "/" {
		requestPath = ""
	} else {
		requestPath = strings.TrimPrefix(requestPath, "/")
	}

	ftpRoot := filepath.Join(s.dataDir, "ftp")
	fullPath := filepath.Join(ftpRoot, requestPath)

	// Ensure the path is within the FTP directory for security
	if !strings.HasPrefix(fullPath, ftpRoot) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Read directory entries
	dirEntries, err := os.ReadDir(fullPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read directory"})
		return
	}

	listing := DirectoryListing{
		Path:        requestPath,
		Files:       []FileInfo{},
		Directories: []FileInfo{},
	}

	for _, entry := range dirEntries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		fileInfo := FileInfo{
			Name:    entry.Name(),
			Path:    filepath.Join(requestPath, entry.Name()),
			IsDir:   entry.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		}

		if !entry.IsDir() {
			fileInfo.Type = s.getFileType(entry.Name())
			listing.Files = append(listing.Files, fileInfo)
		} else {
			listing.Directories = append(listing.Directories, fileInfo)
		}
	}

	// Sort for consistent output
	sort.Slice(listing.Files, func(i, j int) bool {
		return listing.Files[i].Name < listing.Files[j].Name
	})
	sort.Slice(listing.Directories, func(i, j int) bool {
		return listing.Directories[i].Name < listing.Directories[j].Name
	})

	c.JSON(http.StatusOK, listing)
}

// getFileType determines the file type based on extension
func (s *security) getFileType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".mp4", ".avi", ".mov", ".mkv", ".webm":
		return "video"
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp":
		return "image"
	case ".txt", ".log":
		return "text"
	default:
		return "file"
	}
}

// serveFTPFile serves files from the FTP directory
func (s *security) serveFTPFile(c *gin.Context) {
	requestPath := c.Param("path")
	if requestPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File path required"})
		return
	}

	requestPath = strings.TrimPrefix(requestPath, "/")
	ftpRoot := filepath.Join(s.dataDir, "ftp")
	fullPath := filepath.Join(ftpRoot, requestPath)

	// Ensure the path is within the FTP directory for security
	if !strings.HasPrefix(fullPath, ftpRoot) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Check if file exists and is not a directory
	info, err := os.Stat(fullPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	if info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is a directory, not a file"})
		return
	}

	// Set appropriate content type
	ext := filepath.Ext(fullPath)
	contentType := "application/octet-stream"
	if mimeType := mime.TypeByExtension(ext); mimeType != "" {
		contentType = mimeType
	}

	// Add content type and file info headers
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", fmt.Sprintf("%d", info.Size()))
	c.Header("Last-Modified", info.ModTime().UTC().Format(http.TimeFormat))

	// For videos and images, add cache headers
	fileType := s.getFileType(info.Name())
	if fileType == "video" || fileType == "image" {
		c.Header("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	}

	// Serve the file
	c.File(fullPath)
}

// getFTPFileInfo returns metadata about a specific file
func (s *security) getFTPFileInfo(c *gin.Context) {
	requestPath := c.Param("path")
	if requestPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File path required"})
		return
	}

	requestPath = strings.TrimPrefix(requestPath, "/")
	ftpRoot := filepath.Join(s.dataDir, "ftp")
	fullPath := filepath.Join(ftpRoot, requestPath)

	// Ensure the path is within the FTP directory for security
	if !strings.HasPrefix(fullPath, ftpRoot) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	fileInfo := FileInfo{
		Name:    info.Name(),
		Path:    requestPath,
		IsDir:   info.IsDir(),
		Size:    info.Size(),
		ModTime: info.ModTime(),
		Type:    s.getFileType(info.Name()),
	}

	c.JSON(http.StatusOK, fileInfo)
}
