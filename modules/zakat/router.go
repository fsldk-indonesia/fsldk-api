// Package zakat wires the routing of the zakat module — a single public
// gold-price proxy endpoint, no CMS side.
package zakat

import (
	"fsldk-api/middlewares"
	"fsldk-api/modules/zakat/zakat_handler"

	"github.com/gin-gonic/gin"
)

// RegisterPublicRoutes mounts GET /public/zakat/gold-price. Fully public (the
// calculator page is for the general public); the light rate limit only guards
// against trivial flooding of our own endpoint — the 1-hour cache already
// keeps upstream load near zero.
func RegisterPublicRoutes(pub *gin.RouterGroup, h zakat_handler.Handler) {
	pub.GET("/zakat/gold-price", middlewares.RateLimit(30, 10), h.GoldPrice)
}
