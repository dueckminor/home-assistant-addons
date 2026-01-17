package ginutil

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ParseQueryStringArray parses query parameters that can be provided in multiple formats:
// - ?param=value1,value2 (comma-separated)
// - ?param=value1&param=value2 (multiple params)
// Returns a slice of trimmed, non-empty strings.
func ParseQueryStringArray(c *gin.Context, paramName string) []string {
	var values []string

	if param := c.Query(paramName); param != "" {
		// Split by comma and trim spaces
		for _, value := range strings.Split(param, ",") {
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				values = append(values, trimmed)
			}
		}
	} else if paramArray, ok := c.GetQueryArray(paramName); ok && len(paramArray) > 0 {
		values = paramArray
	}

	return values
}

func ParseQueryBool(c *gin.Context, paramName string) bool {
	// it should return true for "1", "true", "yes", "on" (case insensitive)
	// and also if the parameter is present without value
	// check if the parameter is present without value
	if value, exists := c.GetQuery(paramName); exists {
		lowerParam := strings.ToLower(value)
		return lowerParam == "" || lowerParam == "1" || lowerParam == "true" || lowerParam == "yes" || lowerParam == "on"
	}
	return false
}

func ParseQueryTime(c *gin.Context, paramName string) time.Time {
	timeStr := c.Query(paramName)
	if timeStr == "" {
		return time.Time{}
	}

	// Try RFC3339 format first
	parsedTime, err := time.Parse(time.RFC3339, timeStr)
	if err == nil {
		return parsedTime
	}

	// Try RFC3339 without timezone (assume UTC)
	parsedTime, err = time.Parse("2006-01-02T15:04:05", timeStr)
	if err == nil {
		return parsedTime.UTC()
	}

	return time.Time{}
}
