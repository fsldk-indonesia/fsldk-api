// Package statistic_handler defines the HTTP transport layer for the public
// network statistics feature.
package statistic_handler

import "github.com/gin-gonic/gin"

// Handler defines HTTP handlers for the public "Statistik Jaringan" endpoints.
type Handler interface {
	NetworkStats(c *gin.Context)
	Directory(c *gin.Context)
}
