// Package statistic wires the routing of the public network statistics
// feature — read-only aggregate counts, no CMS side.
package statistic

import (
	"fsldk-api/modules/statistic/statistic_handler"
	"fsldk-api/modules/statistic/statistic_repository"
	"fsldk-api/modules/statistic/statistic_service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterPublicRoutes mounts the public "Statistik Jaringan" endpoints.
// Fully public and unauthenticated — the data exposed (org/kader counts
// grouped by type/province/level, plus a logo directory) is anonymized and
// aggregate, safe to share without login.
func RegisterPublicRoutes(pub *gin.RouterGroup, db *gorm.DB) {
	repo := statistic_repository.NewRepository(db)
	svc := statistic_service.NewService(repo)
	handler := statistic_handler.NewHandler(svc)

	pub.GET("/network-stats", handler.NetworkStats)
	pub.GET("/network-stats/directory", handler.Directory)
}
